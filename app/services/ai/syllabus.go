package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/genai"
)

// GenerateSyllabus memanggil Gemini untuk memecah materi (file dan/atau teks)
// menjadi daftar sub-materi berurutan (silabus). Minimal salah satu dari
// materialText atau fileBytes harus diisi.
//
// fileMimeType wajib diisi kalau fileBytes tidak kosong, contoh:
// "application/pdf", "application/vnd.openxmlformats-officedocument.presentationml.presentation".
func GenerateSyllabus(ctx context.Context, materialText string, fileBytes []byte, fileMimeType string) (*SyllabusResponse, error) {
	if strings.TrimSpace(materialText) == "" && len(fileBytes) == 0 {
		return nil, fmt.Errorf("materi kosong: perlu file atau teks materi")
	}

	parts := []*genai.Part{
		{Text: syllabusPrompt},
	}
	if len(fileBytes) > 0 {
		parts = append(parts, &genai.Part{
			InlineData: &genai.Blob{Data: fileBytes, MIMEType: fileMimeType},
		})
	}
	if strings.TrimSpace(materialText) != "" {
		parts = append(parts, &genai.Part{Text: "Materi (teks):\n" + materialText})
	}

	config := &genai.GenerateContentConfig{
		ResponseMIMEType:   "application/json",
		ResponseJsonSchema: syllabusJSONSchema,
		// Temperature rendah supaya struktur silabus lebih konsisten antar percobaan.
		Temperature: genai.Ptr(float32(0.3)),
	}

	result, err := generateContentWithFailover(
		ctx,
		Models(),
		[]*genai.Content{{Parts: parts}},
		config,
	)
	if err != nil {
		return nil, fmt.Errorf("gagal memanggil Gemini API: %w", err)
	}

	var syllabus SyllabusResponse
	if err := json.Unmarshal([]byte(result.Text()), &syllabus); err != nil {
		return nil, fmt.Errorf("gagal parsing response AI: %w", err)
	}

	// Validasi minimal — jangan percaya penuh walau sudah pakai structured output.
	if len(syllabus.Modules) == 0 {
		return nil, fmt.Errorf("AI tidak menghasilkan modul yang valid")
	}
	for i, m := range syllabus.Modules {
		if strings.TrimSpace(m.Title) == "" {
			return nil, fmt.Errorf("modul ke-%d tidak memiliki judul", i+1)
		}
	}

	return &syllabus, nil
}
