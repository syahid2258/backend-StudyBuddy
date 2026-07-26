package ai

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"google.golang.org/genai"

	"goravel/app/facades"
)

var (
	client   *genai.Client
	initOnce sync.Once
	initErr  error
)

// GetClient mengembalikan instance genai.Client yang di-inisialisasi sekali
// (singleton) dan dipakai ulang di seluruh service AI, sesuai rekomendasi
// resmi Google Gen AI SDK.
func GetClient(ctx context.Context) (*genai.Client, error) {
	initOnce.Do(func() {
		apiKey := facades.Config().GetString("ai.gemini_api_key")
		if apiKey == "" {
			initErr = fmt.Errorf("GEMINI_API_KEY belum diset di .env")
			return
		}

		client, initErr = genai.NewClient(ctx, &genai.ClientConfig{
			APIKey:  apiKey,
			Backend: genai.BackendGeminiAPI,
		})
	})

	return client, initErr
}

// Models mengembalikan daftar model fallback dari config/ai.go.
func Models() []string {
	modelsStr := facades.Config().GetString("ai.gemini_models", "gemini-2.5-flash,gemini-3.5-flash,gemini-3.6-flash,gemini-3-flash-preview,gemini-2.5-pro")
	rawModels := strings.Split(modelsStr, ",")
	var parsed []string
	for _, m := range rawModels {
		if m = strings.TrimSpace(m); m != "" {
			parsed = append(parsed, m)
		}
	}
	if len(parsed) == 0 {
		return []string{"gemini-2.5-flash"} // fallback absolut
	}
	return parsed
}

// RequestTimeout mengembalikan durasi timeout untuk tiap panggilan ke Gemini.
func RequestTimeout() time.Duration {
	seconds := facades.Config().GetInt("ai.request_timeout", 60)
	return time.Duration(seconds) * time.Second
}
