package controllers

import (
	"context"
	"strconv"
	"time"

	"github.com/goravel/framework/contracts/http"

	"goravel/app/facades"
	"goravel/app/models"
	"goravel/app/services/ai"
	"goravel/app/services/srs"
)

type FlashcardController struct{}

func NewFlashcardController() *FlashcardController {
	return &FlashcardController{}
}

// Due mengembalikan semua flashcard dari pengguna, agar dapat dipelajari kapan saja (Deck Mode).
func (c *FlashcardController) Due(ctx http.Context) http.Response {
	userID, ok := ctx.Value("auth_user_id").(uint)
	if !ok || userID == 0 {
		return ctx.Response().String(401, "Unauthorized")
	}

	// Cari semua topik (project) milik user
	var projects []models.Project
	if err := facades.Orm().Query().Where("user_id", userID).Get(&projects); err != nil {
		return ctx.Response().String(500, "Gagal mengambil topik pengguna: "+err.Error())
	}

	if len(projects) == 0 {
		return ctx.Response().Success().Json(http.Json{"cards": []http.Json{}})
	}

	projectIDs := []any{}
	projectTitles := map[uint]string{}
	for _, p := range projects {
		projectIDs = append(projectIDs, p.ID)
		projectTitles[p.ID] = p.Title
	}

	// Ambil semua flashcard tanpa mempedulikan next_review_date
	var flashcards []models.Flashcard
	if err := facades.Orm().Query().WhereIn("project_id", projectIDs).Get(&flashcards); err != nil {
		return ctx.Response().String(500, "Gagal mengambil flashcard: "+err.Error())
	}

	cards := make([]http.Json, 0, len(flashcards))
	for _, fc := range flashcards {
		title := projectTitles[fc.ProjectID]

		cards = append(cards, http.Json{
			"card_id":      fc.ID,
			"project_name": title,
			"front_text":   fc.FrontText,
			"back_text":    fc.BackText,
		})
	}

	return ctx.Response().Success().Json(http.Json{
		"total_due": len(cards),
		"cards":     cards,
	})
}

// Review menerima rating kesulitan user ("hard" | "good" | "easy") untuk
// sebuah flashcard, lalu update jadwal review berikutnya lewat algoritma
// SM-2 sederhana.
func (c *FlashcardController) Review(ctx http.Context) http.Response {
	idStr := ctx.Request().Route("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return ctx.Response().String(400, "ID flashcard tidak valid")
	}

	rating := ctx.Request().Input("difficulty_rating")
	if rating != "hard" && rating != "good" && rating != "easy" {
		return ctx.Response().String(400, "difficulty_rating harus salah satu dari: hard, good, easy")
	}

	var flashcard models.Flashcard
	if err := facades.Orm().Query().Find(&flashcard, id); err != nil {
		return ctx.Response().String(500, "Gagal mengambil flashcard: "+err.Error())
	}
	if flashcard.ID == 0 {
		return ctx.Response().String(404, "Flashcard tidak ditemukan")
	}

	newEase, newInterval := srs.ScheduleNextReview(flashcard.EaseFactor, flashcard.IntervalDays, rating)
	nextReviewDate := time.Now().AddDate(0, 0, newInterval)

	flashcard.EaseFactor = newEase
	flashcard.IntervalDays = newInterval
	flashcard.NextReviewDate = &nextReviewDate

	if err := facades.Orm().Query().Save(&flashcard); err != nil {
		return ctx.Response().String(500, "Gagal memperbarui jadwal review: "+err.Error())
	}

	return ctx.Response().Success().Json(http.Json{
		"message": "Jadwal review diperbarui",
		"data": http.Json{
			"next_review_date": nextReviewDate,
		},
	})
}

// Generate menerima permintaan dari user untuk membuat/membuat ulang flashcard untuk topik tertentu.
func (c *FlashcardController) Generate(ctx http.Context) http.Response {
	userID, _ := ctx.Value("auth_user_id").(uint)

	idStr := ctx.Request().Route("id")
	projectID, err := strconv.Atoi(idStr)
	if err != nil {
		return ctx.Response().Json(400, http.Json{"error": "ID topik tidak valid"})
	}

	message := ctx.Request().Input("message")
	
	var project models.Project
	if err := facades.Orm().Query().Where("id", projectID).Where("user_id", userID).FirstOrFail(&project); err != nil {
		return ctx.Response().Json(404, http.Json{"error": "Topik tidak ditemukan atau bukan milik Anda"})
	}

	// Call AI to generate flashcards
	reqCtx, cancel := context.WithTimeout(context.Background(), ai.RequestTimeout())
	defer cancel()

	flashcards, err := ai.GenerateFlashcards(reqCtx, project.Title, message)
	if err != nil {
		return ctx.Response().Json(500, http.Json{"error": "Gagal membuat flashcard: " + err.Error()})
	}

	// Hapus flashcard lama (Regenerate berarti mengganti yang lama)
	if _, err := facades.Orm().Query().Where("project_id", projectID).Delete(&models.Flashcard{}); err != nil {
		return ctx.Response().Json(500, http.Json{"error": "Gagal menghapus flashcard lama: " + err.Error()})
	}

	// Save to DB
	for _, fc := range flashcards {
		newCard := models.Flashcard{
			ProjectID:    uint(projectID),
			FrontText:    fc.FrontText,
			BackText:     fc.BackText,
			EaseFactor:   2.5,
			IntervalDays: 0,
		}
		facades.Orm().Query().Create(&newCard)
	}



	return ctx.Response().Json(200, http.Json{
		"status":  "success",
		"message": "Flashcard berhasil dibuat",
	})
}

// Delete menghapus semua flashcard untuk suatu topik
func (c *FlashcardController) Delete(ctx http.Context) http.Response {
	userID, _ := ctx.Value("auth_user_id").(uint)

	idStr := ctx.Request().Route("id")
	projectID, err := strconv.Atoi(idStr)
	if err != nil {
		return ctx.Response().Json(400, http.Json{"error": "ID topik tidak valid"})
	}

	var project models.Project
	if err := facades.Orm().Query().Where("id", projectID).Where("user_id", userID).FirstOrFail(&project); err != nil {
		return ctx.Response().Json(404, http.Json{"error": "Topik tidak ditemukan atau bukan milik Anda"})
	}

	if _, err := facades.Orm().Query().Where("project_id", projectID).Delete(&models.Flashcard{}); err != nil {
		return ctx.Response().Json(500, http.Json{"error": "Gagal menghapus flashcard: " + err.Error()})
	}

	return ctx.Response().Json(200, http.Json{
		"status":  "success",
		"message": "Semua flashcard pada topik ini berhasil dihapus",
	})
}
