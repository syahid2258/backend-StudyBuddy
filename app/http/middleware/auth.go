package middleware

import (
	"net/http"
	"strings"

	httpcontract "github.com/goravel/framework/contracts/http"

	"goravel/app/facades"
	"goravel/app/models"
)

// Auth is a Goravel-compatible middleware struct.
// It implements the http.Middleware interface: Handle(Context) and Signature() string.
type Auth struct{}

func NewAuth() *Auth {
	return &Auth{}
}

// Signature returns a unique name for this middleware.
func (a *Auth) Signature() string {
	return "auth"
}

// Handle checks for a valid JWT token in the "auth_token" HttpOnly cookie.
// On success: stores the authenticated user in context and proceeds.
// On failure: redirects to /login.
func (a *Auth) Handle(ctx httpcontract.Context) {
	token := ctx.Request().Cookie("auth_token")
	if token == "" {
		if strings.HasPrefix(ctx.Request().Path(), "/api/") {
			ctx.Response().Json(http.StatusUnauthorized, httpcontract.Json{"error": "Unauthorized"}).Abort()
		} else {
			ctx.Response().Redirect(http.StatusFound, "/login").Abort()
		}
		return
	}

	// Parse & validate the JWT token
	guard := facades.Auth(ctx)
	payload, err := guard.Parse(token)
	if err != nil || payload == nil {
		// Token invalid or expired — clear cookie and redirect
		ctx.Response().Cookie(httpcontract.Cookie{
			Name:     "auth_token",
			Value:    "",
			MaxAge:   -1,
			Path:     "/",
			HttpOnly: true,
		})
		if strings.HasPrefix(ctx.Request().Path(), "/api/") {
			ctx.Response().Json(http.StatusUnauthorized, httpcontract.Json{"error": "Unauthorized"}).Abort()
		} else {
			ctx.Response().Redirect(http.StatusFound, "/login").Abort()
		}
		return
	}

	// Load user from DB and store in context
	var user models.User
	if dbErr := facades.Orm().Query().Find(&user, payload.Key); dbErr != nil || user.ID == 0 {
		if strings.HasPrefix(ctx.Request().Path(), "/api/") {
			ctx.Response().Json(http.StatusUnauthorized, httpcontract.Json{"error": "Unauthorized"}).Abort()
		} else {
			ctx.Response().Redirect(http.StatusFound, "/login").Abort()
		}
		return
	}

	// Store user in context for downstream controllers
	ctx.WithValue("auth_user_id", user.ID)
	ctx.WithValue("auth_user", user)

	// Continue to next handler
	ctx.Request().Next()
}
