package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/genai"
)

// GenerateExam memanggil Gemini untuk membuat soal ujian akhir berdasarkan
// seluruh materi project. previousQuestions (boleh nil/kosong) dipakai untuk
// fitur "Generate Soal Baru" (retake) — supaya AI diminta membuat variasi
// yang berbeda, bukan mengulang persis soal lama.
func GenerateExam(ctx context.Context, materialText string, questionTypes []string, totalQuestions int, previousQuestions []string) (*ExamGenerationResponse, error) {
	if strings.TrimSpace(materialText) == "" {
		return nil, fmt.Errorf("materi kosong — tidak bisa generate soal tanpa materi")
	}
	if totalQuestions <= 0 {
		totalQuestions = 5
	}
	if len(questionTypes) == 0 {
		questionTypes = []string{"multiple_choice"}
	}

	extraInstruction := ""
	if len(previousQuestions) > 0 {
		var sb strings.Builder
		sb.WriteString(examRetakeInstruction)
		for i, q := range previousQuestions {
			sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, q))
		}
		extraInstruction = sb.String()
	}

	prompt := fmt.Sprintf(
		examGenerationPromptTemplate,
		materialText,
		totalQuestions,
		strings.Join(questionTypes, ", "),
		extraInstruction,
	)

	config := &genai.GenerateContentConfig{
		ResponseMIMEType:   "application/json",
		ResponseJsonSchema: examJSONSchema,
		Temperature:        genai.Ptr(float32(0.6)),
	}

	result, err := generateContentWithFailover(
		ctx,
		Models(),
		[]*genai.Content{{Parts: []*genai.Part{{Text: prompt}}}},
		config,
	)
	if err != nil {
		return nil, fmt.Errorf("gagal memanggil Gemini API: %w", err)
	}

	var exam ExamGenerationResponse
	if err := json.Unmarshal([]byte(result.Text()), &exam); err != nil {
		return nil, fmt.Errorf("gagal parsing response AI: %w", err)
	}

	if err := validateExamGeneration(&exam); err != nil {
		return nil, err
	}

	return &exam, nil
}

func validateExamGeneration(exam *ExamGenerationResponse) error {
	if len(exam.Questions) == 0 {
		return fmt.Errorf("AI tidak menghasilkan soal yang valid")
	}
	seenIDs := map[string]bool{}
	for i, q := range exam.Questions {
		if strings.TrimSpace(q.ID) == "" {
			return fmt.Errorf("soal ke-%d tidak punya ID", i+1)
		}
		if seenIDs[q.ID] {
			return fmt.Errorf("ID soal duplikat: %s", q.ID)
		}
		seenIDs[q.ID] = true

		if strings.TrimSpace(q.Question) == "" {
			return fmt.Errorf("soal ke-%d tidak punya teks pertanyaan", i+1)
		}
		if strings.TrimSpace(q.CorrectAnswer) == "" {
			return fmt.Errorf("soal ke-%d tidak punya correct_answer", i+1)
		}
		if q.Type == "multiple_choice" {
			if len(q.Options) < 2 {
				return fmt.Errorf("soal pilihan ganda ke-%d punya kurang dari 2 opsi", i+1)
			}
			found := false
			for _, opt := range q.Options {
				if opt == q.CorrectAnswer {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("correct_answer soal ke-%d tidak ada di daftar options", i+1)
			}
		}
	}
	return nil
}

// GradeEssayAnswers memanggil Gemini untuk menilai jawaban essay siswa,
// digabung dengan hasil pilihan ganda (yang sudah dinilai eksak di Go)
// menjadi satu final_score + analysis.
func GradeEssayAnswers(ctx context.Context, materialText string, essays []EssayAnswer, mcCorrect, mcTotal int) (*ExamGradingResponse, error) {
	if len(essays) == 0 {
		return nil, fmt.Errorf("tidak ada soal essay untuk dinilai")
	}

	var essaysBlock strings.Builder
	for i, e := range essays {
		essaysBlock.WriteString(fmt.Sprintf(
			"%d. Soal: %s\n   Poin kunci jawaban: %s\n   Jawaban siswa: %s\n\n",
			i+1, e.Question, e.ModelAnswer, e.StudentAnswer,
		))
	}

	prompt := fmt.Sprintf(examGradingPromptTemplate, materialText, mcCorrect, mcTotal, essaysBlock.String())

	config := &genai.GenerateContentConfig{
		ResponseMIMEType:   "application/json",
		ResponseJsonSchema: examGradingJSONSchema,
		Temperature:        genai.Ptr(float32(0.2)),
	}

	result, err := generateContentWithFailover(
		ctx,
		Models(),
		[]*genai.Content{{Parts: []*genai.Part{{Text: prompt}}}},
		config,
	)
	if err != nil {
		// FALLBACK: Simple Offline Grading if AI fails (e.g. no API key)
		essayScore := 0
		essayTotal := len(essays)
		var fallbackAnalysis strings.Builder
		fallbackAnalysis.WriteString("Analisis Otomatis (Offline Mode):\n")
		
		for _, e := range essays {
			studentAnsLower := strings.ToLower(strings.TrimSpace(e.StudentAnswer))
			modelAnsLower := strings.ToLower(strings.TrimSpace(e.ModelAnswer))
			
			// Simple heuristic: check if any keyword from model answer exists in student answer
			words := strings.Fields(modelAnsLower)
			matchCount := 0
			for _, w := range words {
				// skip very short words like 'di', 'ke', 'dan'
				if len(w) > 3 && strings.Contains(studentAnsLower, w) {
					matchCount++
				}
			}
			
			// Condition for passing: at least 1 significant keyword matched or exact match
			if matchCount > 0 || studentAnsLower == modelAnsLower {
				essayScore++
				fallbackAnalysis.WriteString(fmt.Sprintf("- Soal: %s\n  Status: Benar (Terdapat kecocokan kata kunci)\n", e.Question))
			} else {
				fallbackAnalysis.WriteString(fmt.Sprintf("- Soal: %s\n  Status: Kurang Tepat (Ekspektasi: %s)\n", e.Question, e.ModelAnswer))
			}
		}
		
		finalScore := 0
		totalQuestions := mcTotal + essayTotal
		if totalQuestions > 0 {
			finalScore = int((float64(mcCorrect+essayScore) / float64(totalQuestions)) * 100)
		}
		
		fallbackAnalysis.WriteString(fmt.Sprintf("\nPilihan Ganda: %d/%d Benar\nEsai: %d/%d Benar", mcCorrect, mcTotal, essayScore, essayTotal))

		return &ExamGradingResponse{
			FinalScore: finalScore,
			Analysis:   fallbackAnalysis.String(),
		}, nil
	}

	var grading ExamGradingResponse
	if err := json.Unmarshal([]byte(result.Text()), &grading); err != nil {
		return nil, fmt.Errorf("gagal parsing response AI: %w", err)
	}

	if grading.FinalScore < 0 || grading.FinalScore > 100 {
		return nil, fmt.Errorf("final_score di luar rentang 0-100: %d", grading.FinalScore)
	}
	if strings.TrimSpace(grading.Analysis) == "" {
		return nil, fmt.Errorf("analysis kosong")
	}

	return &grading, nil
}
