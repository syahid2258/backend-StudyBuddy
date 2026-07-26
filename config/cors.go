package config

import (
	"goravel/app/facades"
)

func init() {
	config := facades.Config()

	// Allowed origins — add your production domain here.
	// FINDING-001 FIX: Restrict CORS to known origins, never use wildcard.
	allowedOrigins := []string{
		"https://localhost:5173",
		"http://localhost:5173",
		"http://localhost:3000",
	}
	productionOrigin, _ := config.Env("APP_ALLOWED_ORIGIN", "").(string)
	if productionOrigin != "" {
		allowedOrigins = append(allowedOrigins, productionOrigin)
	}

	config.Add("cors", map[string]any{
		"paths":                []string{"/api/*"},
		"allowed_methods":      []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		"allowed_origins":      allowedOrigins,
		"allowed_headers":      []string{"Content-Type", "Authorization", "X-Requested-With"},
		"exposed_headers":      []string{},
		"max_age":              3600,
		"supports_credentials": true,
	})
}
