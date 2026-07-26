package controllers

import (
	"context"
	"fmt"
	stdhttp "net/http"
	"os"
	"strconv"
	"strings"

	httpcontract "github.com/goravel/framework/contracts/http"

	"goravel/app/facades"
	"goravel/app/models"
	"goravel/app/services/ai"
)

type TopicController struct{}

func NewTopicController() *TopicController {
	return &TopicController{}
}

// Index returns all projects/topics for the authenticated user.
func (c *TopicController) Index(ctx httpcontract.Context) httpcontract.Response {
	userID := ctx.Value("auth_user_id")

	var projects []models.Project
	query := facades.Orm().Query().OrderBy("created_at", "desc")
	if userID != nil {
		query = query.Where("user_id", userID)
	}

	if err := query.Get(&projects); err != nil {
		return ctx.Response().Json(stdhttp.StatusInternalServerError, httpcontract.Json{
			"status":  "error",
			"message": "Gagal mengambil data topik: " + err.Error(),
		})
	}

	return ctx.Response().Json(stdhttp.StatusOK, httpcontract.Json{
		"status": "success",
		"data":   projects,
	})
}

// Store creates a new project/topic for the authenticated user.
func (c *TopicController) Store(ctx httpcontract.Context) httpcontract.Response {
	title := ctx.Request().Input("title")
	completed, _ := strconv.Atoi(ctx.Request().Input("completed"))
	total, _ := strconv.Atoi(ctx.Request().Input("total"))

	if title == "" {
		return ctx.Response().Json(stdhttp.StatusBadRequest, httpcontract.Json{
			"status":  "error",
			"message": "Judul topik diperlukan.",
		})
	}
	if total <= 0 {
		total = 1
	}

	userID, _ := ctx.Value("auth_user_id").(uint)

	project := models.Project{
		UserID:    &userID,
		Title:     title,
		Completed: completed,
		Total:     total,
	}

	// Methods bersifat opsional — request lama dari frontend yang belum
	// mengirim field ini tetap jalan normal (Methods akan tersimpan nil).
	if ctx.Request().Input("feynman") != "" || ctx.Request().Input("pomodoro") != "" ||
		ctx.Request().Input("spaced_repetition") != "" {
		project.Methods = &models.ProjectMethods{
			Feynman:          ctx.Request().InputBool("feynman"),
			Pomodoro:         ctx.Request().InputBool("pomodoro"),
			SpacedRepetition: ctx.Request().InputBool("spaced_repetition"),
		}
	}

	if err := facades.Orm().Query().Create(&project); err != nil {
		return ctx.Response().Json(stdhttp.StatusInternalServerError, httpcontract.Json{
			"status":  "error",
			"message": "Gagal menyimpan topik: " + err.Error(),
		})
	}

	return ctx.Response().Json(stdhttp.StatusOK, httpcontract.Json{
		"status": "success",
		"data":   project,
	})
}

// Update modifies a project/topic. Only allows updating own topics.
func (c *TopicController) Update(ctx httpcontract.Context) httpcontract.Response {
	idStr := ctx.Request().Route("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return ctx.Response().Json(stdhttp.StatusBadRequest, httpcontract.Json{
			"status":  "error",
			"message": "ID tidak valid.",
		})
	}

	userID, _ := ctx.Value("auth_user_id").(uint)

	var project models.Project
	if err := facades.Orm().Query().Where("id", id).Where("user_id", userID).First(&project); err != nil {
		return ctx.Response().Json(stdhttp.StatusInternalServerError, httpcontract.Json{
			"status":  "error",
			"message": "Gagal mengambil topik.",
		})
	}
	if project.ID == 0 {
		return ctx.Response().Json(stdhttp.StatusNotFound, httpcontract.Json{
			"status":  "error",
			"message": "Topik tidak ditemukan.",
		})
	}

	if title := ctx.Request().Input("title"); title != "" {
		project.Title = title
	}
	if completedStr := ctx.Request().Input("completed"); completedStr != "" {
		project.Completed, _ = strconv.Atoi(completedStr)
	}
	if totalStr := ctx.Request().Input("total"); totalStr != "" {
		project.Total, _ = strconv.Atoi(totalStr)
	}

	if err := facades.Orm().Query().Save(&project); err != nil {
		return ctx.Response().Json(stdhttp.StatusInternalServerError, httpcontract.Json{
			"status":  "error",
			"message": "Gagal memperbarui topik: " + err.Error(),
		})
	}

	return ctx.Response().Json(stdhttp.StatusOK, httpcontract.Json{
		"status": "success",
		"data":   project,
	})
}

// Modules mengembalikan daftar modul (rute belajar) milik sebuah project,
// terurut sesuai Order — dipakai halaman Rute Belajar di frontend.
func (c *TopicController) Modules(ctx httpcontract.Context) httpcontract.Response {
	idStr := ctx.Request().Route("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return ctx.Response().String(400, "ID topik tidak valid")
	}

	var modules []models.Module
	if err := facades.Orm().Query().Where("project_id", id).OrderBy("modules.order", "asc").Get(&modules); err != nil {
		return ctx.Response().String(500, "Gagal mengambil modul: "+err.Error())
	}

	return ctx.Response().Success().Json(httpcontract.Json{
		"data": modules,
	})
}

// GenerateSyllabus adalah endpoint inti Fase 2: terima materi (file upload
// dan/atau teks) + judul + toggle metode belajar, panggil AI untuk memecah
// materi jadi silabus, lalu simpan Project + Module ke DB.
//
// Menerima multipart/form-data dengan field:
//   - title (wajib)
//   - file (opsional, PDF/PPT/dll — salah satu dari file/prompt_text wajib ada)
//   - prompt_text (opsional, materi diketik langsung)
//   - feynman, pomodoro, spaced_repetition (opsional, boolean)
func (c *TopicController) GenerateSyllabus(ctx httpcontract.Context) httpcontract.Response {
	title := ctx.Request().Input("title")
	if title == "" {
		return ctx.Response().String(400, "Judul topik diperlukan")
	}

	promptText := ctx.Request().Input("prompt_text")

	var fileBytes []byte
	var fileMimeType string
	var sourceFileURL string

	// FINDING-013 FIX: Validate file type and size
	const maxFileSize = 10 * 1024 * 1024 // 10 MB
	allowedMimeTypes := map[string]bool{
		"application/pdf":  true,
		"text/plain":        true,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
		"application/vnd.openxmlformats-officedocument.presentationml.presentation": true,
		"application/vnd.ms-powerpoint": true,
		"application/msword": true,
	}

	if uploaded, err := ctx.Request().File("file"); err == nil && uploaded != nil {
		path := uploaded.File()
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return ctx.Response().Json(400, httpcontract.Json{"error": "Gagal membaca file yang diunggah."})
		}

		// Validate file size
		if len(data) > maxFileSize {
			return ctx.Response().Json(400, httpcontract.Json{"error": "Ukuran file melebihi batas maksimum 10 MB."})
		}

		// Detect MIME type from content (magic bytes), not from request header
		detectedMime := stdhttp.DetectContentType(data)
		// Strip charset suffix if present
		if idx := strings.Index(detectedMime, ";"); idx != -1 {
			detectedMime = strings.TrimSpace(detectedMime[:idx])
		}
		// PDF magic bytes override
		if len(data) >= 4 && string(data[:4]) == "%PDF" {
			detectedMime = "application/pdf"
		}
		if !allowedMimeTypes[detectedMime] {
			return ctx.Response().Json(400, httpcontract.Json{"error": "Tipe file tidak diizinkan. Gunakan PDF, TXT, DOCX, atau PPTX."})
		}

		fileBytes = data
		fileMimeType = detectedMime

		uploadedPath, storeErr := uploaded.Store("topics")
		if storeErr == nil {
			sourceFileURL = "/storage/" + uploadedPath
		}
	}

	if promptText == "" && len(fileBytes) == 0 {
		return ctx.Response().String(400, "Perlu mengirim file materi atau prompt_text")
	}

	userID, _ := ctx.Value("auth_user_id").(uint)


	reqCtx, cancel := context.WithTimeout(context.Background(), ai.RequestTimeout())
	defer cancel()

	syllabus, err := ai.GenerateSyllabus(reqCtx, promptText, fileBytes, fileMimeType)
	if err != nil {
		return ctx.Response().Json(502, httpcontract.Json{
			"error": "Gagal generate silabus dari AI: " + err.Error(),
		})
	}

	project := models.Project{
		UserID:        &userID,
		Title:         title,
		Total:         len(syllabus.Modules),
		SourceFileURL: sourceFileURL,
	}
	if ctx.Request().Input("feynman") != "" || ctx.Request().Input("pomodoro") != "" ||
		ctx.Request().Input("spaced_repetition") != "" {
		project.Methods = &models.ProjectMethods{
			Feynman:          ctx.Request().InputBool("feynman"),
			Pomodoro:         ctx.Request().InputBool("pomodoro"),
			SpacedRepetition: ctx.Request().InputBool("spaced_repetition"),
		}
	}

	if err := facades.Orm().Query().Create(&project); err != nil {
		facades.Log().Errorf("GenerateSyllabus: create project error=%v", err)
		return ctx.Response().String(500, "Terjadi kesalahan internal saat menyimpan project.")
	}

	modules := make([]models.Module, 0, len(syllabus.Modules))
	for i, m := range syllabus.Modules {
		modules = append(modules, models.Module{
			ProjectID: project.ID,
			Title:     m.Title,
			Order:     i + 1,
			IsLocked:  i != 0, // modul pertama terbuka, sisanya terkunci
			Status: func() string {
				if i == 0 {
					return "not_started"
				}
				return "locked"
			}(),
		})
	}

	if err := facades.Orm().Query().Create(&modules); err != nil {
		facades.Log().Errorf("GenerateSyllabus: create modules error=%v", err)
		return ctx.Response().String(500, "Terjadi kesalahan internal saat menyimpan modul.")
	}

	if project.Methods != nil && project.Methods.SpacedRepetition {
		var instructionBuilder strings.Builder
		instructionBuilder.WriteString("Cakup materi dari subbab-subbab berikut:\n")
		for i, m := range syllabus.Modules {
			instructionBuilder.WriteString(fmt.Sprintf("%d. %s\n", i+1, m.Title))
		}
		instructionBuilder.WriteString(fmt.Sprintf("\nPastikan Anda membuat setidaknya %d flashcard, yaitu minimal 1 flashcard untuk setiap subbab.", len(syllabus.Modules)))

		go func(projID uint, projTitle string, instruction string) {
			bgCtx, cancel := context.WithTimeout(context.Background(), ai.RequestTimeout())
			defer cancel()
			flashcards, err := ai.GenerateFlashcards(bgCtx, projTitle, instruction)
			if err == nil && len(flashcards) > 0 {
				var fcs []models.Flashcard
				for _, fc := range flashcards {
					fcs = append(fcs, models.Flashcard{
						ProjectID:    projID,
						FrontText:    fc.FrontText,
						BackText:     fc.BackText,
						EaseFactor:   2.5,
						IntervalDays: 0,
					})
				}
				facades.Orm().Query().Create(&fcs)
			}
		}(project.ID, project.Title, instructionBuilder.String())
	}

	return ctx.Response().Success().Json(httpcontract.Json{
		"project": project,
		"modules": modules,
	})
}

// Destroy removes a project/topic. Only allows deleting own topics.
func (c *TopicController) Destroy(ctx httpcontract.Context) httpcontract.Response {
	idStr := ctx.Request().Route("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return ctx.Response().Json(stdhttp.StatusBadRequest, httpcontract.Json{
			"status":  "error",
			"message": "ID tidak valid.",
		})
	}

	userID, _ := ctx.Value("auth_user_id").(uint)

	var project models.Project
	if err := facades.Orm().Query().Where("id", id).Where("user_id", userID).First(&project); err != nil {
		return ctx.Response().Json(stdhttp.StatusInternalServerError, httpcontract.Json{
			"status":  "error",
			"message": "Gagal mengambil topik.",
		})
	}
	if project.ID == 0 {
		return ctx.Response().Json(stdhttp.StatusNotFound, httpcontract.Json{
			"status":  "error",
			"message": "Topik tidak ditemukan.",
		})
	}

	if _, err := facades.Orm().Query().Delete(&project); err != nil {
		return ctx.Response().Json(stdhttp.StatusInternalServerError, httpcontract.Json{
			"status":  "error",
			"message": "Gagal menghapus topik: " + err.Error(),
		})
	}

	return ctx.Response().Json(stdhttp.StatusOK, httpcontract.Json{
		"status":  "success",
		"message": "Topik berhasil dihapus.",
	})
}
