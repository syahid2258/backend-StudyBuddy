package controllers

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/goravel/framework/contracts/http"

	"goravel/app/facades"
	"goravel/app/models"
	"goravel/app/services/ai"
)

type ModuleController struct{}

func NewModuleController() *ModuleController {
	return &ModuleController{}
}

// Content mengembalikan rangkuman materi + jembatan keledai untuk sebuah
// modul (Fase 3). Kalau modul ini sudah pernah di-generate sebelumnya
// (content_blocks sudah terisi), langsung dikembalikan dari DB tanpa
// memanggil AI lagi — supaya tidak boros biaya/latency tiap kali user
// membuka ulang modul yang sama.
func (c *ModuleController) Content(ctx http.Context) http.Response {
	idStr := ctx.Request().Route("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return ctx.Response().String(400, "ID modul tidak valid")
	}

	var module models.Module
	if err := facades.Orm().Query().With("ActiveRecall").With("Attempts").Find(&module, id); err != nil {
		facades.Log().Errorf("ModuleController.Content: DB error id=%d err=%v", id, err)
		return ctx.Response().String(500, "Terjadi kesalahan internal.")
	}
	if module.ID == 0 {
		return ctx.Response().String(404, "Modul tidak ditemukan")
	}

	// FINDING-012 FIX: Validate ownership through project
	userID, _ := ctx.Value("auth_user_id").(uint)
	var ownerProject models.Project
	if err := facades.Orm().Query().Where("id", module.ProjectID).Where("user_id", userID).First(&ownerProject); err != nil || ownerProject.ID == 0 {
		return ctx.Response().String(403, "Akses ditolak — modul ini bukan milik Anda")
	}

	if module.IsLocked {
		return ctx.Response().String(403, "Modul ini masih terkunci — selesaikan modul sebelumnya dulu")
	}

	// Cache: sudah pernah digenerate sebelumnya, langsung balikin dari DB.
	if module.ContentBlocks != nil && len(*module.ContentBlocks) > 0 {
		var project models.Project
		_ = facades.Orm().Query().Find(&project, module.ProjectID)
		return ctx.Response().Success().Json(http.Json{
			"data":            module,
			"project_methods": project.Methods,
		})
	}

	var project models.Project
	if err := facades.Orm().Query().Find(&project, module.ProjectID); err != nil {
		facades.Log().Errorf("ModuleController.Content: find project error=%v", err)
		return ctx.Response().String(500, "Terjadi kesalahan internal.")
	}
	if project.ID == 0 {
		return ctx.Response().String(404, "Project untuk modul ini tidak ditemukan")
	}

	reqCtx, cancel := context.WithTimeout(context.Background(), ai.RequestTimeout())
	defer cancel()

	result, err := ai.GenerateModuleContent(reqCtx, module.Title, project.Title)
	if err != nil {
		return ctx.Response().String(502, "Gagal generate materi dari AI: "+err.Error())
	}

	blocks := make(models.ContentBlockList, 0, len(result.ContentBlocks))
	for _, b := range result.ContentBlocks {
		blocks = append(blocks, models.ContentBlock{
			Type:  b.Type,
			Title: b.Title,
			Text:  b.Text,
		})
	}
	module.ContentBlocks = &blocks
	if module.Status == "not_started" {
		module.Status = "in_progress"
	}

	if err := facades.Orm().Query().Save(&module); err != nil {
		facades.Log().Errorf("ModuleController.Content: save error=%v", err)
		return ctx.Response().String(500, "Terjadi kesalahan internal.")
	}

	return ctx.Response().Success().Json(http.Json{
		"data":            module,
		"project_methods": project.Methods,
	})
}

// Evaluate adalah endpoint inti Fase 5: terima penjelasan Active Recall user
// (teks ATAU audio), evaluasi lewat AI dibanding materi asli modul, lalu:
//  1. Simpan hasilnya sebagai ModuleAttempt (log audit).
//  2. Kalau lulus: tandai modul "mastered", unlock modul berikutnya, update
//     progress project, dan simpan flashcard dari bagian yang terlewat.
//
// Menerima multipart/form-data dengan field:
//   - user_explanation (teks, opsional)
//   - audio (file, opsional — salah satu dari user_explanation/audio wajib ada)
func (c *ModuleController) Evaluate(ctx http.Context) http.Response {
	idStr := ctx.Request().Route("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return ctx.Response().String(400, "ID modul tidak valid")
	}

	userExplanation := ctx.Request().Input("user_explanation")

	var audioBytes []byte
	var audioMimeType string
	if uploaded, err := ctx.Request().File("audio"); err == nil && uploaded != nil {
		data, readErr := os.ReadFile(uploaded.File())
		if readErr != nil {
			return ctx.Response().String(400, "Gagal membaca file audio: "+readErr.Error())
		}
		audioBytes = data
		mimeType, mimeErr := uploaded.MimeType()
		if mimeErr != nil || mimeType == "" {
			mimeType = "audio/mpeg"
		}
		audioMimeType = mimeType
	}

	if strings.TrimSpace(userExplanation) == "" && len(audioBytes) == 0 {
		return ctx.Response().String(400, "Kirim user_explanation (teks) atau audio")
	}

	var module models.Module
	if err := facades.Orm().Query().Find(&module, id); err != nil {
		return ctx.Response().String(500, "Gagal mengambil modul: "+err.Error())
	}
	if module.ID == 0 {
		return ctx.Response().String(404, "Modul tidak ditemukan")
	}
	if module.IsLocked {
		return ctx.Response().String(403, "Modul ini masih terkunci")
	}
	if module.ContentBlocks == nil || len(*module.ContentBlocks) == 0 {
		return ctx.Response().String(400, "Materi modul ini belum digenerate — buka halaman Materi dulu")
	}

	var materialBuilder strings.Builder
	for _, b := range *module.ContentBlocks {
		materialBuilder.WriteString(b.Text)
		materialBuilder.WriteString("\n")
	}

	reqCtx, cancel := context.WithTimeout(context.Background(), ai.RequestTimeout())
	defer cancel()

	eval, err := ai.EvaluateFeynman(reqCtx, materialBuilder.String(), userExplanation, audioBytes, audioMimeType)
	if err != nil {
		return ctx.Response().String(502, "Gagal evaluasi Feynman dari AI: "+err.Error())
	}

	score := eval.FeynmanScore
	attempt := models.ModuleAttempt{
		ModuleID:        uint(module.ID),
		FeynmanScore:    &score,
		UserExplanation: userExplanation,
		Feedback:        fmt.Sprintf("Pujian:\n%s\n\nKekurangan:\n%s\n\nSaran:\n%s", eval.Feedback.Pujian, eval.Feedback.Kekurangan, eval.Feedback.Saran),
	}
	_ = facades.Orm().Query().Create(&attempt)

	if eval.IsPassed {
		module.Status = "mastered"
		_ = facades.Orm().Query().Save(&module)

		var nextModule models.Module
		if err := facades.Orm().Query().Where("project_id", module.ProjectID).Where("order", module.Order+1).First(&nextModule); err == nil && nextModule.ID != 0 {
			nextModule.IsLocked = false
			nextModule.Status = "not_started"
			_ = facades.Orm().Query().Save(&nextModule)
		}

		var project models.Project
		if err := facades.Orm().Query().Find(&project, module.ProjectID); err == nil && project.ID != 0 {
			project.Completed++
			_ = facades.Orm().Query().Save(&project)
		}
	} else if len(eval.GenerateFlashcards) > 0 {
		var fcs []models.Flashcard
		for _, fc := range eval.GenerateFlashcards {
			fcs = append(fcs, models.Flashcard{
				ProjectID: module.ProjectID,
				FrontText: fc.FrontText,
				BackText:  fc.BackText,
			})
		}
		_ = facades.Orm().Query().Create(&fcs)
	}

	return ctx.Response().Success().Json(eval)
}

// Complete allows user to bypass Active Recall and complete the module directly,
// ONLY IF Feynman Technique is disabled in the project preferences.
func (c *ModuleController) Complete(ctx http.Context) http.Response {
	idStr := ctx.Request().Route("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return ctx.Response().String(400, "ID modul tidak valid")
	}

	var module models.Module
	if err := facades.Orm().Query().Find(&module, id); err != nil || module.ID == 0 {
		return ctx.Response().String(404, "Modul tidak ditemukan")
	}

	if module.IsLocked {
		return ctx.Response().String(403, "Modul ini masih terkunci")
	}

	var project models.Project
	if err := facades.Orm().Query().Find(&project, module.ProjectID); err != nil || project.ID == 0 {
		return ctx.Response().String(404, "Project tidak ditemukan")
	}

	// Security check: ensure Feynman is actually disabled
	if project.Methods != nil && project.Methods.Feynman {
		return ctx.Response().String(403, "Feynman Technique diaktifkan. Anda harus menyelesaikan Active Recall.")
	}

	// Mark as completed
	if module.Status != "mastered" {
		module.Status = "mastered"
		_ = facades.Orm().Query().Save(&module)

		var nextModule models.Module
		if err := facades.Orm().Query().Where("project_id", module.ProjectID).Where("`order`", module.Order+1).First(&nextModule); err == nil && nextModule.ID != 0 {
			nextModule.IsLocked = false
			nextModule.Status = "not_started"
			_ = facades.Orm().Query().Save(&nextModule)
		}

		project.Completed++
		_ = facades.Orm().Query().Save(&project)
	}

	return ctx.Response().Success().Json(http.Json{
		"message": "Modul berhasil diselesaikan",
	})
}

// maxChatHistoryTurns membatasi berapa banyak giliran percakapan lama yang
// dikirim ulang ke Gemini tiap request (Fase 7 pakai histori stateless —
// tanpa batas ini, biaya token akan terus membesar seiring panjang chat).
const maxChatHistoryTurns = 20

// ChatHistory mengembalikan seluruh riwayat percakapan "Tanya AI" untuk
// sebuah modul, dipakai frontend untuk render ulang chat saat halaman dibuka.
//
// GET /api/modules/{id}/chat
func (c *ModuleController) ChatHistory(ctx http.Context) http.Response {
	idStr := ctx.Request().Route("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return ctx.Response().String(400, "ID modul tidak valid")
	}

	var messages []models.ChatMessage
	if err := facades.Orm().Query().Where("module_id", id).OrderBy("id").Get(&messages); err != nil {
		return ctx.Response().String(500, "Gagal mengambil riwayat chat: "+err.Error())
	}

	return ctx.Response().Success().Json(messages)
}

// Chat adalah endpoint inti Fase 7 (Tanya AI): terima pertanyaan user seputar
// modul yang sedang dibuka, jawab lewat AI dengan konteks materi modul +
// histori percakapan sebelumnya, lalu simpan kedua giliran (user & model)
// ke chat_messages.
//
// POST /api/modules/{id}/chat
// Body (form): message (wajib)
func (c *ModuleController) Chat(ctx http.Context) http.Response {
	idStr := ctx.Request().Route("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return ctx.Response().String(400, "ID modul tidak valid")
	}

	message := ctx.Request().Input("message")
	if strings.TrimSpace(message) == "" {
		return ctx.Response().String(400, "message diperlukan")
	}

	var module models.Module
	if err := facades.Orm().Query().Find(&module, id); err != nil {
		return ctx.Response().String(500, "Gagal mengambil modul: "+err.Error())
	}
	if module.ID == 0 {
		return ctx.Response().String(404, "Modul tidak ditemukan")
	}
	if module.ContentBlocks == nil || len(*module.ContentBlocks) == 0 {
		return ctx.Response().String(400, "Materi modul ini belum digenerate — buka halaman Materi dulu")
	}

	var materialBuilder strings.Builder
	for _, b := range *module.ContentBlocks {
		materialBuilder.WriteString(b.Text)
		materialBuilder.WriteString("\n")
	}

	var previousMessages []models.ChatMessage
	_ = facades.Orm().Query().Where("module_id", id).OrderBy("id").Get(&previousMessages)
	if len(previousMessages) > maxChatHistoryTurns {
		previousMessages = previousMessages[len(previousMessages)-maxChatHistoryTurns:]
	}

	history := make([]ai.ChatTurn, 0, len(previousMessages))
	for _, m := range previousMessages {
		history = append(history, ai.ChatTurn{Role: m.Role, Content: m.Content})
	}

	reqCtx, cancel := context.WithTimeout(context.Background(), ai.RequestTimeout())
	defer cancel()

	reply, err := ai.Reply(reqCtx, module.Title, materialBuilder.String(), history, message)
	if err != nil {
		return ctx.Response().String(502, "Gagal mendapat jawaban dari AI: "+err.Error())
	}

	userMsg := models.ChatMessage{ModuleID: uint(module.ID), Role: "user", Content: message}
	_ = facades.Orm().Query().Create(&userMsg)
	modelMsg := models.ChatMessage{ModuleID: uint(module.ID), Role: "model", Content: reply}
	_ = facades.Orm().Query().Create(&modelMsg)

	return ctx.Response().Success().Json(map[string]any{
		"reply": reply,
	})
}

// Regenerate is called when user triggers regeneration for a specific sub-topic
// POST /api/modules/{id}/regenerate
func (c *ModuleController) Regenerate(ctx http.Context) http.Response {
	idStr := ctx.Request().Route("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return ctx.Response().String(400, "ID modul tidak valid")
	}

	reason := ctx.Request().Input("reason")
	if strings.TrimSpace(reason) == "" {
		reason = "Tolong perbaiki dan tulis ulang sub-topik ini menjadi lebih baik."
	}

	var module models.Module
	if err := facades.Orm().Query().Find(&module, id); err != nil || module.ID == 0 {
		return ctx.Response().String(404, "Modul tidak ditemukan")
	}

	var project models.Project
	if err := facades.Orm().Query().Find(&project, module.ProjectID); err != nil || project.ID == 0 {
		return ctx.Response().String(404, "Project tidak ditemukan")
	}

	var oldMaterialBuilder strings.Builder
	if module.ContentBlocks != nil {
		for _, b := range *module.ContentBlocks {
			oldMaterialBuilder.WriteString(b.Text)
			oldMaterialBuilder.WriteString("\n")
		}
	}

	reqCtx, cancel := context.WithTimeout(context.Background(), ai.RequestTimeout())
	defer cancel()

	result, err := ai.RegenerateModuleContent(reqCtx, module.Title, project.Title, reason, oldMaterialBuilder.String())
	if err != nil {
		return ctx.Response().String(502, "Gagal regenerate materi dari AI: "+err.Error())
	}

	blocks := make(models.ContentBlockList, 0, len(result.ContentBlocks))
	for _, b := range result.ContentBlocks {
		blocks = append(blocks, models.ContentBlock{
			Type:  b.Type,
			Title: b.Title,
			Text:  b.Text,
		})
	}
	
	module.ContentBlocks = &blocks
	if err := facades.Orm().Query().Save(&module); err != nil {
		return ctx.Response().String(500, "Gagal menyimpan materi hasil regenerasi: "+err.Error())
	}



	return ctx.Response().Success().Json(module)
}
