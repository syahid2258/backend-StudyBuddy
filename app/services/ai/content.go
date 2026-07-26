package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/genai"
)

// GenerateModuleContent memanggil Gemini untuk membuat rangkuman materi +
// minimal satu jembatan keledai untuk sebuah modul, dalam konteks judul
// project tempat modul itu berada.
func GenerateModuleContent(ctx context.Context, moduleTitle, projectTitle string) (*ModuleContentResponse, error) {
	if strings.TrimSpace(moduleTitle) == "" {
		return nil, fmt.Errorf("judul modul kosong")
	}

	prompt := fmt.Sprintf(moduleContentPromptTemplate, moduleTitle, projectTitle)

	config := &genai.GenerateContentConfig{
		ResponseMIMEType:   "application/json",
		ResponseJsonSchema: moduleContentJSONSchema,
		// Temperature lebih tinggi dari silabus supaya jembatan keledai lebih
		// kreatif/variatif, bukan generik.
		Temperature: genai.Ptr(float32(0.7)),
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

	var content ModuleContentResponse
	if err := json.Unmarshal([]byte(result.Text()), &content); err != nil {
		return nil, fmt.Errorf("gagal parsing response AI: %w", err)
	}

	// Validasi minimal.
	if len(content.ContentBlocks) == 0 {
		return nil, fmt.Errorf("AI tidak menghasilkan konten yang valid")
	}
	hasJembatanKeledai := false
	for i, b := range content.ContentBlocks {
		if strings.TrimSpace(b.Text) == "" {
			return nil, fmt.Errorf("blok konten ke-%d kosong", i+1)
		}
		if b.Type == "jembatan_keledai" {
			hasJembatanKeledai = true
		}
	}
	if !hasJembatanKeledai {
		return nil, fmt.Errorf("AI tidak menyertakan blok jembatan_keledai")
	}

	return &content, nil
}

// RegenerateModuleContent memanggil Gemini untuk menulis ulang rangkuman materi
// beserta jembatan keledai untuk sebuah modul, berdasarkan revisi/alasan dari user.
func RegenerateModuleContent(ctx context.Context, moduleTitle, projectTitle, reason, oldContent string) (*ModuleContentResponse, error) {
	if strings.TrimSpace(moduleTitle) == "" {
		return nil, fmt.Errorf("judul modul kosong")
	}

	prompt := fmt.Sprintf(regenerateModulePromptTemplate, moduleTitle, projectTitle, reason, oldContent)

	config := &genai.GenerateContentConfig{
		ResponseMIMEType:   "application/json",
		ResponseJsonSchema: moduleContentJSONSchema,
		Temperature:        genai.Ptr(float32(0.7)),
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

	var content ModuleContentResponse
	if err := json.Unmarshal([]byte(result.Text()), &content); err != nil {
		return nil, fmt.Errorf("gagal parsing response AI: %w", err)
	}

	// Validasi minimal.
	if len(content.ContentBlocks) == 0 {
		return nil, fmt.Errorf("AI tidak menghasilkan konten yang valid")
	}
	hasJembatanKeledai := false
	for i, b := range content.ContentBlocks {
		if strings.TrimSpace(b.Text) == "" {
			return nil, fmt.Errorf("blok konten ke-%d kosong", i+1)
		}
		if b.Type == "jembatan_keledai" {
			hasJembatanKeledai = true
		}
	}
	if !hasJembatanKeledai {
		return nil, fmt.Errorf("AI tidak menyertakan blok jembatan_keledai")
	}

	return &content, nil
}
