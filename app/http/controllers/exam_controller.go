package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/goravel/framework/contracts/http"

	"goravel/app/facades"
	"goravel/app/models"
	"goravel/app/services/ai"
)

type ExamController struct{}

func NewExamController() *ExamController {
	return &ExamController{}
}

// GenerateExam adalah endpoint Fase 6: generate soal ujian akhir untuk
// sebuah project, berdasarkan seluruh materi modul yang sudah di-generate.
//
// POST /api/topics/{id}/exam/generate
// Body (form): question_types (comma-separated, mis. "multiple_choice,essay"),
//
//	total_questions (default 5)
func (c *ExamController) GenerateExam(ctx http.Context) http.Response {
	idStr := ctx.Request().Route("id")
	projectID, err := strconv.Atoi(idStr)
	if err != nil {
		return ctx.Response().String(400, "ID topik tidak valid")
	}

	var project models.Project
	if err := facades.Orm().Query().Find(&project, projectID); err != nil {
		return ctx.Response().String(500, "Gagal mengambil project: "+err.Error())
	}
	if project.ID == 0 {
		return ctx.Response().String(404, "Project tidak ditemukan")
	}

	var modules []models.Module
	if err := facades.Orm().Query().Where("project_id", projectID).OrderBy("modules.order", "asc").Get(&modules); err != nil {
		return ctx.Response().String(500, "Gagal mengambil modul: "+err.Error())
	}

	var materialBuilder strings.Builder
	for _, m := range modules {
		if m.ContentBlocks == nil {
			continue
		}
		for _, b := range *m.ContentBlocks {
			materialBuilder.WriteString(b.Text)
			materialBuilder.WriteString("\n")
		}
	}
	if materialBuilder.Len() == 0 {
		return ctx.Response().String(400, "Belum ada materi yang digenerate untuk project ini — selesaikan minimal satu modul dulu")
	}

	questionTypesRaw := ctx.Request().Input("question_types")
	if questionTypesRaw == "" {
		questionTypesRaw = "multiple_choice"
	}
	questionTypes := strings.Split(questionTypesRaw, ",")
	for i := range questionTypes {
		questionTypes[i] = strings.TrimSpace(questionTypes[i])
	}

	totalQuestions, _ := strconv.Atoi(ctx.Request().Input("total_questions"))
	if totalQuestions <= 0 {
		totalQuestions = 5
	}

	// Fitur "Generate Soal Baru" (retake): ambil soal-soal lama milik project
	// ini supaya AI diminta membuat variasi berbeda, bukan mengulang persis.
	var previousExams []models.Exam
	_ = facades.Orm().Query().Where("project_id", projectID).Get(&previousExams)
	var previousQuestionTexts []string
	for _, e := range previousExams {
		var qs []ai.ExamQuestion
		if err := json.Unmarshal([]byte(e.Questions), &qs); err == nil {
			for _, q := range qs {
				previousQuestionTexts = append(previousQuestionTexts, q.Question)
			}
		}
	}

	reqCtx, cancel := context.WithTimeout(context.Background(), ai.RequestTimeout())
	defer cancel()

	examResp, err := ai.GenerateExam(reqCtx, materialBuilder.String(), questionTypes, totalQuestions, previousQuestionTexts)
	if err != nil {
		return ctx.Response().String(502, "Gagal generate soal dari AI: "+err.Error())
	}

	questionTypesJSON, _ := json.Marshal(questionTypes)
	questionsJSON, _ := json.Marshal(examResp.Questions)

	exam := models.Exam{
		ProjectID:     project.ID,
		Title:         project.Title,
		QuestionTypes: string(questionTypesJSON),
		Questions:     string(questionsJSON),
	}
	if err := facades.Orm().Query().Create(&exam); err != nil {
		return ctx.Response().String(500, "Gagal menyimpan ujian: "+err.Error())
	}

	var safeQuestions []http.Json
	for _, q := range examResp.Questions {
		safeQuestions = append(safeQuestions, http.Json{
			"id":       q.ID,
			"type":     q.Type,
			"question": q.Question,
			"options":  q.Options,
		})
	}


	return ctx.Response().Success().Json(http.Json{
		"message":   "Ujian berhasil di-generate",
		"exam_id":   exam.ID,
		"questions": safeQuestions,
	})
}

type examAnswerInput struct {
	QuestionID string `json:"question_id"`
	Answer     string `json:"answer"`
}

// Submit adalah endpoint Fase 6: terima jawaban user, grading pilihan ganda
// langsung di Go (pencocokan eksak, tidak perlu AI), grading essay via AI
// kalau ada, lalu digabung jadi satu final_score + analysis.
//
// POST /api/exams/{id}/submit
// Body JSON: {"answers": [{"question_id": "Q1", "answer": "..."}]}
func (c *ExamController) Submit(ctx http.Context) http.Response {
	idStr := ctx.Request().Route("id")
	examID, err := strconv.Atoi(idStr)
	if err != nil {
		return ctx.Response().String(400, "ID ujian tidak valid")
	}

	var exam models.Exam
	if err := facades.Orm().Query().Find(&exam, examID); err != nil {
		return ctx.Response().String(500, "Gagal mengambil ujian: "+err.Error())
	}
	if exam.ID == 0 {
		return ctx.Response().String(404, "Ujian tidak ditemukan")
	}

	var body struct {
		Answers []examAnswerInput `json:"answers"`
	}
	if err := ctx.Request().Bind(&body); err != nil {
		return ctx.Response().String(400, "Body request tidak valid: "+err.Error())
	}
	if len(body.Answers) == 0 {
		return ctx.Response().String(400, "Jawaban tidak boleh kosong")
	}

	var questions []ai.ExamQuestion
	if err := json.Unmarshal([]byte(exam.Questions), &questions); err != nil {
		return ctx.Response().String(500, "Gagal membaca soal ujian: "+err.Error())
	}

	answerByID := map[string]string{}
	for _, a := range body.Answers {
		answerByID[a.QuestionID] = a.Answer
	}

	mcTotal, mcCorrect := 0, 0
	var essays []ai.EssayAnswer

	for _, q := range questions {
		studentAnswer := answerByID[q.ID]
		if q.Type == "multiple_choice" {
			mcTotal++
			if strings.EqualFold(strings.TrimSpace(studentAnswer), strings.TrimSpace(q.CorrectAnswer)) {
				mcCorrect++
			}
		} else {
			essays = append(essays, ai.EssayAnswer{
				Question:      q.Question,
				ModelAnswer:   q.CorrectAnswer,
				StudentAnswer: studentAnswer,
			})
		}
	}

	var finalScore int
	var analysis string

	if len(essays) == 0 {
		// Semua soal pilihan ganda — tidak perlu panggil AI sama sekali, hemat biaya.
		if mcTotal > 0 {
			finalScore = int(float64(mcCorrect) / float64(mcTotal) * 100)
		}
		analysis = fmt.Sprintf("Kamu menjawab benar %d dari %d soal pilihan ganda.", mcCorrect, mcTotal)
	} else {
		var materialBuilder strings.Builder
		var modules []models.Module
		_ = facades.Orm().Query().Where("project_id", exam.ProjectID).Get(&modules)
		for _, m := range modules {
			if m.ContentBlocks == nil {
				continue
			}
			for _, b := range *m.ContentBlocks {
				materialBuilder.WriteString(b.Text)
				materialBuilder.WriteString("\n")
			}
		}

		reqCtx, cancel := context.WithTimeout(context.Background(), ai.RequestTimeout())
		defer cancel()

		grading, err := ai.GradeEssayAnswers(reqCtx, materialBuilder.String(), essays, mcCorrect, mcTotal)
		if err != nil {
			return ctx.Response().String(502, "Gagal menilai jawaban esai dari AI: "+err.Error())
		}
		finalScore = grading.FinalScore
		analysis = grading.Analysis
	}

	answersJSON, _ := json.Marshal(body.Answers)
	attempt := models.ExamAttempt{
		ExamID:     exam.ID,
		Answers:    string(answersJSON),
		FinalScore: &finalScore,
		Analysis:   &analysis,
	}
	if err := facades.Orm().Query().Create(&attempt); err != nil {
		return ctx.Response().String(500, "Gagal menyimpan hasil ujian: "+err.Error())
	}

	return ctx.Response().Success().Json(http.Json{
		"final_score": finalScore,
		"analysis":    analysis,
		"can_retake":  true,
	})
}

// GetExams returns the list of generated exams for a topic
// GET /api/topics/{id}/exams
func (c *ExamController) GetExams(ctx http.Context) http.Response {
	idStr := ctx.Request().Route("id")
	projectID, err := strconv.Atoi(idStr)
	if err != nil {
		return ctx.Response().String(400, "ID topik tidak valid")
	}

	var exams []models.Exam
	if err := facades.Orm().Query().Where("project_id", projectID).OrderBy("created_at").Get(&exams); err != nil {
		return ctx.Response().String(500, "Gagal mengambil daftar ujian: "+err.Error())
	}

	var results []http.Json
	for i, exam := range exams {
		// Try to find the latest attempt
		var attempt models.ExamAttempt
		facades.Orm().Query().Where("exam_id", exam.ID).OrderByDesc("created_at").First(&attempt)

		var questions []any
		json.Unmarshal([]byte(exam.Questions), &questions)

		var types []string
		json.Unmarshal([]byte(exam.QuestionTypes), &types)
		
		title := exam.Title
		if title == "" {
			title = fmt.Sprintf("Grup Soal Ujian %d", i+1)
		}

		examData := http.Json{
			"id":              exam.ID,
			"title":           title,
			"total_questions": len(questions),
			"types":           types,
			"created_at":      exam.CreatedAt,
		}

		if attempt.ID != 0 && attempt.FinalScore != nil {
			examData["final_score"] = *attempt.FinalScore
			examData["completed"] = true
		} else {
			examData["completed"] = false
		}

		results = append(results, examData)
	}

	return ctx.Response().Success().Json(results)
}

// GetExam returns a specific exam by ID with safe questions
// GET /api/exams/{id}
func (c *ExamController) GetExam(ctx http.Context) http.Response {
	idStr := ctx.Request().Route("id")
	examID, err := strconv.Atoi(idStr)
	if err != nil {
		return ctx.Response().String(400, "ID ujian tidak valid")
	}

	var exam models.Exam
	if err := facades.Orm().Query().Find(&exam, examID); err != nil {
		return ctx.Response().String(500, "Gagal mengambil ujian: "+err.Error())
	}
	if exam.ID == 0 {
		return ctx.Response().String(404, "Ujian tidak ditemukan")
	}

	var questions []ai.ExamQuestion
	json.Unmarshal([]byte(exam.Questions), &questions)

	safeQuestions := make([]http.Json, 0, len(questions))
	for _, q := range questions {
		safeQuestions = append(safeQuestions, http.Json{
			"id":       q.ID,
			"type":     q.Type,
			"question": q.Question,
			"options":  q.Options,
		})
	}

	title := exam.Title
	if title == "" {
		title = "Ujian Akhir"
	}

	// Fetch result if any
	var attempt models.ExamAttempt
	facades.Orm().Query().Where("exam_id", exam.ID).OrderByDesc("created_at").First(&attempt)
	var result *http.Json
	if attempt.ID != 0 && attempt.FinalScore != nil {
		result = &http.Json{
			"final_score": *attempt.FinalScore,
			"analysis":    *attempt.Analysis,
			"can_retake":  true,
		}
	}

	return ctx.Response().Success().Json(http.Json{
		"exam_id":   exam.ID,
		"title":     title,
		"questions": safeQuestions,
		"result":    result,
	})
}
