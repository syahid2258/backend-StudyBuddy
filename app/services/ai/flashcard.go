package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/genai"
)

// GenerateFlashcardsOutput is a container for the JSON response
type GenerateFlashcardsOutput struct {
	GenerateFlashcards []GeneratedFlashcard `json:"generate_flashcards"`
}

var flashcardJSONSchema = &genai.Schema{
	Type: genai.TypeObject,
	Properties: map[string]*genai.Schema{
		"generate_flashcards": {
			Type: genai.TypeArray,
			Items: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"front_text": {Type: genai.TypeString, Description: "Pertanyaan atau istilah inti (1-2 kalimat)"},
					"back_text":  {Type: genai.TypeString, Description: "Jawaban atau penjelasan singkat (3-4 kalimat)"},
				},
				Required: []string{"front_text", "back_text"},
			},
		},
	},
	Required: []string{"generate_flashcards"},
}

// GenerateFlashcards calls Gemini to create flashcards based on topic and instructions
func GenerateFlashcards(ctx context.Context, topicTitle, instructions string) ([]GeneratedFlashcard, error) {
	prompt := fmt.Sprintf("Topik: %s\nInstruksi Tambahan: %s\nBuatkan flashcards berkualitas tinggi berdasarkan topik dan instruksi tersebut.", topicTitle, instructions)

	parts := []*genai.Part{
		{Text: prompt},
	}

	config := &genai.GenerateContentConfig{
		ResponseMIMEType:   "application/json",
		ResponseJsonSchema: flashcardJSONSchema,
		SystemInstruction: &genai.Content{
			Role: "system",
			Parts: []*genai.Part{
				{Text: FlashcardSystemPrompt()},
			},
		},
		Temperature: genai.Ptr(float32(0.7)),
	}

	result, err := generateContentWithFailover(
		ctx,
		Models(),
		[]*genai.Content{{Parts: parts}},
		config,
	)
	if err != nil {
		return nil, fmt.Errorf("gagal generate flashcard: %w", err)
	}

	var output GenerateFlashcardsOutput
	if err := json.Unmarshal([]byte(result.Text()), &output); err != nil {
		return nil, fmt.Errorf("gagal parsing response AI: %w", err)
	}

	return output.GenerateFlashcards, nil
}
