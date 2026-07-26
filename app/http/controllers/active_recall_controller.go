package controllers

import (
	"encoding/json"
	"strconv"

	"github.com/goravel/framework/contracts/http"

	"goravel/app/facades"
	"goravel/app/models"
)

type ActiveRecallController struct{}

func NewActiveRecallController() *ActiveRecallController {
	return &ActiveRecallController{}
}

func (c *ActiveRecallController) Submit(ctx http.Context) http.Response {
	userID, _ := ctx.Value("auth_user_id").(uint)

	moduleIDStr := ctx.Request().Route("id")
	moduleID, err := strconv.Atoi(moduleIDStr)
	if err != nil {
		return ctx.Response().Json(http.StatusBadRequest, http.Json{"error": "Invalid module ID"})
	}

	var req struct {
		Answer string `json:"answer"`
	}
	if err := ctx.Request().Bind(&req); err != nil {
		return ctx.Response().Json(http.StatusBadRequest, http.Json{"error": "Invalid JSON body"})
	}

	if req.Answer == "" {
		return ctx.Response().Json(http.StatusBadRequest, http.Json{"error": "Answer cannot be empty"})
	}

	var activeRecall models.ActiveRecall
	if err := facades.Orm().Query().Where("module_id", moduleID).First(&activeRecall); err != nil || activeRecall.ID == 0 {
		activeRecall = models.ActiveRecall{
			ModuleID: uint(moduleID),
			Question: "Jelaskan dengan bahasamu sendiri mengenai konsep utama pada modul ini.",
		}
		facades.Orm().Query().Create(&activeRecall)
	}

	activeRecall.Answer = req.Answer
	facades.Orm().Query().Save(&activeRecall)

	var module models.Module
	facades.Orm().Query().Find(&module, moduleID)

	payload := map[string]any{
		"module_id": moduleID,
		"module_title": module.Title,
		"active_recall_id": activeRecall.ID,
		"question": activeRecall.Question,
		"answer": req.Answer,
	}
	payloadBytes, _ := json.Marshal(payload)

	subTopicID := uint(moduleID)

	adminReq := models.AdminRequest{
		UserID:     userID,
		SubTopicID: &subTopicID,
		Type:       "active_recall_evaluation",
		Payload:    string(payloadBytes),
		Status:     "pending",
	}

	if err := facades.Orm().Query().Create(&adminReq); err != nil {
		return ctx.Response().Json(http.StatusInternalServerError, http.Json{"error": "Failed to create evaluation request"})
	}

	return ctx.Response().Json(http.StatusOK, http.Json{
		"message": "Answer submitted for evaluation successfully",
		"request_id": adminReq.ID,
	})
}
