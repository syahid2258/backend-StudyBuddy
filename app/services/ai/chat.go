package ai

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/genai"
)

// ChatTurn merepresentasikan satu giliran percakapan yang sudah ada.
type ChatTurn struct {
	Role    string // "user" | "model"
	Content string
}

// Reply memanggil Gemini untuk menjawab pertanyaan user seputar materi modul
// tertentu, dengan konteks percakapan sebelumnya (kalau ada).
//
// CATATAN DESAIN: Gemini punya "Interactions API" (previous_interaction_id)
// untuk state management di sisi server, tapi per saat kode ini ditulis itu
// masih berstatus beta dan dukungan resminya baru terverifikasi jelas untuk
// SDK Python/JS — belum untuk Go. Supaya tidak bergantung pada API yang belum
// pasti tersedia di google.golang.org/genai, dipakai pendekatan yang sudah
// pasti didukung dan didokumentasikan resmi: histori percakapan dikelola di
// sisi kita (tabel chat_messages) dan dikirim ulang penuh tiap request.
func Reply(ctx context.Context, moduleTitle, moduleMaterial string, history []ChatTurn, question string) (string, error) {
	if strings.TrimSpace(question) == "" {
		return "", fmt.Errorf("pertanyaan kosong")
	}

	systemInstruction := fmt.Sprintf(chatSystemInstructionTemplate, moduleTitle, moduleMaterial)

	contents := make([]*genai.Content, 0, len(history)+1)
	for _, turn := range history {
		role := turn.Role
		if role != "user" && role != "model" {
			continue // lewati baris korup, jangan sampai bikin request gagal total
		}
		contents = append(contents, &genai.Content{
			Role:  role,
			Parts: []*genai.Part{{Text: turn.Content}},
		})
	}
	contents = append(contents, &genai.Content{
		Role:  "user",
		Parts: []*genai.Part{{Text: question}},
	})

	config := &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{{Text: systemInstruction}},
		},
		Temperature: genai.Ptr(float32(0.5)),
	}

	result, err := generateContentWithFailover(ctx, Models(), contents, config)
	if err != nil {
		return "", fmt.Errorf("gagal memanggil Gemini API: %w", err)
	}

	reply := strings.TrimSpace(result.Text())
	if reply == "" {
		return "", fmt.Errorf("AI tidak memberikan jawaban")
	}

	return reply, nil
}

// GlobalReply memanggil Gemini untuk menjawab pertanyaan user tanpa konteks
// modul yang spesifik (hanya bertindak sebagai asisten belajar umum).
func GlobalReply(ctx context.Context, history []ChatTurn, question string) (string, error) {
	if strings.TrimSpace(question) == "" {
		return "", fmt.Errorf("pertanyaan kosong")
	}

	contents := make([]*genai.Content, 0, len(history)+1)
	for _, turn := range history {
		role := turn.Role
		if role != "user" && role != "model" {
			continue 
		}
		contents = append(contents, &genai.Content{
			Role:  role,
			Parts: []*genai.Part{{Text: turn.Content}},
		})
	}
	contents = append(contents, &genai.Content{
		Role:  "user",
		Parts: []*genai.Part{{Text: question}},
	})

	config := &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{{Text: globalChatSystemInstructionTemplate}},
		},
		Temperature: genai.Ptr(float32(0.5)),
	}

	result, err := generateContentWithFailover(ctx, Models(), contents, config)
	if err != nil {
		return "", fmt.Errorf("gagal memanggil Gemini API: %w", err)
	}

	reply := strings.TrimSpace(result.Text())
	if reply == "" {
		return "", fmt.Errorf("AI tidak memberikan jawaban")
	}

	return reply, nil
}
