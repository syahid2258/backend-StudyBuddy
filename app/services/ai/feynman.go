package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/genai"
)

// FeynmanPassThreshold adalah ambang batas skor untuk lulus (unlock modul
// berikutnya). Dihitung ulang di Go, TIDAK dipercaya mentah dari field
// is_passed yang dikirim AI — supaya progresi user tidak bisa "dibujuk"
// lewat prompt injection di teks penjelasan yang mengaku sudah lulus.
const FeynmanPassThreshold = 70

// maxFeynmanAttempts: percobaan awal + retry kalau response tidak valid
// (skor di luar rentang, feedback kosong, dsb). Bagian ini paling kritis
// di seluruh AI layer karena hasilnya mengontrol unlock modul berikutnya,
// jadi kita tidak langsung fallback ke error di percobaan pertama yang gagal.
const maxFeynmanAttempts = 2

// EvaluateFeynman memanggil Gemini untuk menilai penjelasan Active Recall
// milik user dibanding materi asli modul. Salah satu dari userExplanationText
// atau audioBytes wajib diisi (input teks atau suara, sesuai briefing).
func EvaluateFeynman(ctx context.Context, materialText, userExplanationText string, audioBytes []byte, audioMimeType string) (*FeynmanEvaluationResponse, error) {
	if strings.TrimSpace(userExplanationText) == "" && len(audioBytes) == 0 {
		return nil, fmt.Errorf("penjelasan user kosong: perlu teks atau audio")
	}
	if strings.TrimSpace(materialText) == "" {
		return nil, fmt.Errorf("materi asli modul kosong — tidak bisa evaluasi tanpa pembanding")
	}

	var explanationInstruction string
	parts := []*genai.Part{}
	if len(audioBytes) > 0 {
		explanationInstruction = "Penjelasan siswa dilampirkan sebagai audio pada part berikutnya. Dengarkan dan transkripsikan secara internal sebelum menilai."
	} else {
		explanationInstruction = fmt.Sprintf("Penjelasan siswa (teks):\n\"\"\"\n%s\n\"\"\"", userExplanationText)
	}

	prompt := fmt.Sprintf(feynmanPromptTemplate, materialText, explanationInstruction)
	parts = append(parts, &genai.Part{Text: prompt})
	if len(audioBytes) > 0 {
		parts = append(parts, &genai.Part{InlineData: &genai.Blob{Data: audioBytes, MIMEType: audioMimeType}})
	}

	config := &genai.GenerateContentConfig{
		ResponseMIMEType:   "application/json",
		ResponseJsonSchema: feynmanJSONSchema,
		// Temperature rendah supaya skor konsisten antar percobaan — ini
		// bagian paling kritis di seluruh AI layer.
		Temperature: genai.Ptr(float32(0.2)),
	}

	var lastErr error
	for attempt := 1; attempt <= maxFeynmanAttempts; attempt++ {
		result, err := generateContentWithFailover(
			ctx,
			Models(),
			[]*genai.Content{{Parts: parts}},
			config,
		)
		if err != nil {
			lastErr = fmt.Errorf("gagal memanggil Gemini API: %w", err)
			continue
		}

		var evaluation FeynmanEvaluationResponse
		if err := json.Unmarshal([]byte(result.Text()), &evaluation); err != nil {
			lastErr = fmt.Errorf("gagal parsing response AI: %w", err)
			continue
		}

		if err := validateFeynmanEvaluation(&evaluation); err != nil {
			lastErr = err
			continue
		}

		// is_passed selalu dihitung ulang di sini berdasarkan threshold kita
		// sendiri, menimpa apa pun yang dikirim AI di field is_passed.
		evaluation.IsPassed = evaluation.FeynmanScore >= FeynmanPassThreshold

		return &evaluation, nil
	}

	return nil, fmt.Errorf("evaluasi AI gagal setelah %d percobaan: %w", maxFeynmanAttempts, lastErr)
}

// validateFeynmanEvaluation memastikan response AI minimal masuk akal
// sebelum dipakai untuk mengontrol progresi user.
func validateFeynmanEvaluation(e *FeynmanEvaluationResponse) error {
	if e.FeynmanScore < 0 || e.FeynmanScore > 100 {
		return fmt.Errorf("skor di luar rentang 0-100: %d", e.FeynmanScore)
	}
	if strings.TrimSpace(e.Feedback.Pujian) == "" ||
		strings.TrimSpace(e.Feedback.Kekurangan) == "" ||
		strings.TrimSpace(e.Feedback.Saran) == "" {
		return fmt.Errorf("feedback tidak lengkap (pujian/kekurangan/saran kosong)")
	}
	for i, fc := range e.GenerateFlashcards {
		if strings.TrimSpace(fc.FrontText) == "" || strings.TrimSpace(fc.BackText) == "" {
			return fmt.Errorf("generate_flashcards ke-%d tidak lengkap", i+1)
		}
	}
	return nil
}
