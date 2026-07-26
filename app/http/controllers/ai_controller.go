package controllers

import (
	"context"
	"encoding/json"
	stdhttp "net/http"
	"strings"

	httpcontract "github.com/goravel/framework/contracts/http"

	"goravel/app/facades"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/services/ai"
)

type AIController struct {
	aiService *services.AIService
}

type AiController struct{}

func NewAIController() *AIController {
	return &AIController{
		aiService: services.NewAIService(),
	}
}

func NewAiController() *AiController {
	return &AiController{}
}

// Show renders the AI chat page (GET /tanya-ai).
func (c *AIController) Show(ctx httpcontract.Context) httpcontract.Response {
	userID := ctx.Value("auth_user_id")

	var user models.User
	var history []models.AdminRequest
	if userID != nil {
		facades.Orm().Query().Find(&user, userID)
		facades.Orm().Query().Where("user_id", userID).Where("type", "tanya_ai").OrderBy("created_at").Get(&history)
	}

	historyJSON, _ := json.Marshal(history)

	return ctx.Response().View().Make("tanyaAI.tmpl", map[string]any{
		"user": user,
		"historyJSON": string(historyJSON),
	})
}

// ChatHistory returns the history of general TanyaAI chats (Fase 1/2) for the logged-in user.
// GET /api/ai/chat
func (c *AIController) ChatHistory(ctx httpcontract.Context) httpcontract.Response {
	userID, _ := ctx.Value("auth_user_id").(uint)

	var history []models.AdminRequest
	if err := facades.Orm().Query().Where("user_id", userID).Where("type", "tanya_ai").OrderBy("created_at").Get(&history); err != nil {
		return ctx.Response().Json(stdhttp.StatusInternalServerError, httpcontract.Json{"error": "Gagal mengambil riwayat chat"})
	}

	return ctx.Response().Json(stdhttp.StatusOK, history)
}

// Chat handles POST /api/ai/chat — the main AI interaction endpoint.
// Accepts a JSON or form body with "message" field.
func (c *AIController) Chat(ctx httpcontract.Context) httpcontract.Response {
	message := ctx.Request().Input("message")
	if strings.TrimSpace(message) == "" {
		return ctx.Response().Json(stdhttp.StatusBadRequest, httpcontract.Json{
			"status":  "error",
			"message": "Pesan tidak boleh kosong.",
		})
	}

	userID, _ := ctx.Value("auth_user_id").(uint)

	// Fetch previous messages for context
	var previousRequests []models.AdminRequest
	_ = facades.Orm().Query().Where("user_id", userID).Where("type", "tanya_ai").OrderBy("id", "asc").Get(&previousRequests)
	
	const maxChatHistoryTurns = 20
	if len(previousRequests) > maxChatHistoryTurns {
		previousRequests = previousRequests[len(previousRequests)-maxChatHistoryTurns:]
	}

	var history []ai.ChatTurn
	for _, req := range previousRequests {
		var payload map[string]any
		if err := json.Unmarshal([]byte(req.Payload), &payload); err == nil {
			if userMsg, ok := payload["message"].(string); ok && userMsg != "" {
				history = append(history, ai.ChatTurn{Role: "user", Content: userMsg})
				if req.Response != "" {
					history = append(history, ai.ChatTurn{Role: "model", Content: req.Response})
				}
			}
		}
	}

	reqCtx, cancel := context.WithTimeout(context.Background(), ai.RequestTimeout())
	defer cancel()

	reply, err := ai.GlobalReply(reqCtx, history, message)
	if err != nil {
		return ctx.Response().Json(stdhttp.StatusBadGateway, httpcontract.Json{
			"status":  "error",
			"message": "Gagal mendapat jawaban dari AI: " + err.Error(),
		})
	}

	payloadData := map[string]any{
		"message": message,
	}
	payloadBytes, _ := json.Marshal(payloadData)

	adminReq := models.AdminRequest{
		UserID:   userID,
		Type:     "tanya_ai",
		Payload:  string(payloadBytes),
		Status:   "completed",
		Response: reply,
	}

	if err := facades.Orm().Query().Create(&adminReq); err != nil {
		return ctx.Response().Json(stdhttp.StatusInternalServerError, httpcontract.Json{
			"status":  "error",
			"message": "Gagal menyimpan histori chat: " + err.Error(),
		})
	}

	return ctx.Response().Json(stdhttp.StatusOK, httpcontract.Json{
		"status": "success",
		"reply":  reply,
	})
}

// PoolStatus adalah endpoint diagnostik untuk melihat status key pool:
// berapa banyak key terdaftar, dan mana yang sedang cooldown (baru kena
// error retryable) — berguna untuk memantau apakah auto-rotate/failover
// benar-benar berjalan. Tidak pernah menampilkan API key asli secara penuh.
//
// GET /api/ai/pool-status
func (c *AiController) PoolStatus(ctx httpcontract.Context) httpcontract.Response {
	reqCtx, cancel := context.WithTimeout(context.Background(), ai.RequestTimeout())
	defer cancel()

	entries, err := ai.GetPoolStatus(reqCtx)
	if err != nil {
		return ctx.Response().String(500, "Gagal mengambil status pool: "+err.Error())
	}

	return ctx.Response().Success().Json(httpcontract.Json{
		"total_keys": len(entries),
		"keys":       entries,
	})
}
