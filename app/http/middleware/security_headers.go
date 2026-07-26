package middleware

import (
	httpcontract "github.com/goravel/framework/contracts/http"
)

// SecurityHeaders adds critical HTTP security headers to every response.
// Closes FINDING-003 (Missing HTTP Security Headers).
type SecurityHeaders struct{}

func NewSecurityHeaders() *SecurityHeaders {
	return &SecurityHeaders{}
}

func (m *SecurityHeaders) Signature() string {
	return "security_headers"
}

func (m *SecurityHeaders) Handle(ctx httpcontract.Context) {
	// Prevent MIME type sniffing
	ctx.Response().Header("X-Content-Type-Options", "nosniff")

	// Clickjacking protection (FINDING-003)
	ctx.Response().Header("X-Frame-Options", "DENY")

	// Legacy XSS protection for older browsers
	ctx.Response().Header("X-XSS-Protection", "1; mode=block")

	// Referrer Policy — don't leak URL info to third parties
	ctx.Response().Header("Referrer-Policy", "strict-origin-when-cross-origin")

	// Permissions Policy — disable unnecessary browser features
	ctx.Response().Header("Permissions-Policy", "geolocation=(), microphone=(), camera=(), payment=()")

	// HSTS — force HTTPS (will be effective in production with HTTPS)
	ctx.Response().Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")

	// Content Security Policy — restrict sources of executable content
	ctx.Response().Header("Content-Security-Policy",
		"default-src 'self'; "+
			"script-src 'self' 'unsafe-inline' cdn.jsdelivr.net cdnjs.cloudflare.com; "+
			"style-src 'self' 'unsafe-inline' cdn.jsdelivr.net fonts.googleapis.com cdnjs.cloudflare.com; "+
			"font-src 'self' fonts.gstatic.com cdn.jsdelivr.net; "+
			"img-src 'self' data: blob:; "+
			"connect-src 'self' wss: ws:; "+
			"frame-ancestors 'none';",
	)

	// Cache control for sensitive pages
	path := ctx.Request().Path()
	if len(path) >= 4 && path[:4] == "/api" {
		ctx.Response().Header("Cache-Control", "no-store, no-cache, must-revalidate, private")
		ctx.Response().Header("Pragma", "no-cache")
	}

	ctx.Request().Next()
}
