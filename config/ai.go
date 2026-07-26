package config

import (
	"goravel/app/facades"
)

func init() {
	config := facades.Config()
	config.Add("ai", map[string]any{
		"gemini_api_key": config.Env("GEMINI_API_KEY", ""),
		// GEMINI_API_KEYS: multi-key untuk auto-rotate + failover, format
		// comma-separated, mis: "key_akun_1,key_akun_2,key_akun_3".
		// Kalau kosong, fallback ke GEMINI_API_KEY (single key, Fase 1).
		"gemini_api_keys": config.Env("GEMINI_API_KEYS", ""),
		"gemini_models":   config.Env("GEMINI_MODELS", "gemini-2.5-flash,gemini-3.5-flash,gemini-3.6-flash,gemini-3-flash-preview,gemini-2.5-pro"),
		"request_timeout": config.Env("AI_REQUEST_TIMEOUT_SECONDS", 300),
	})
}
