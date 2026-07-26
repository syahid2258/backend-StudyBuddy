package controllers

import (
	"fmt"
	"net/http"
	"regexp"

	httpcontract "github.com/goravel/framework/contracts/http"

	"goravel/app/facades"
	"goravel/app/services"
)

type AuthController struct {
	authService *services.AuthService
}

func NewAuthController() *AuthController {
	return &AuthController{
		authService: services.NewAuthService(),
	}
}

// Login handles POST /api/login
// Verifies credentials, generates JWT, sets auth cookie.
func (c *AuthController) Login(ctx httpcontract.Context) httpcontract.Response {
	email := ctx.Request().Input("email")
	password := ctx.Request().Input("password")

	if email == "" || password == "" {
		return ctx.Response().Json(http.StatusBadRequest, httpcontract.Json{
			"status":  "error",
			"message": "Email dan password wajib diisi.",
		})
	}

	// Find user by email
	user, err := c.authService.FindByEmail(email)
	if err != nil {
		return ctx.Response().Json(http.StatusUnauthorized, httpcontract.Json{
			"status":  "error",
			"message": "Email atau password salah.",
		})
	}

	// Verify password
	if !facades.Hash().Check(password, user.Password) {
		return ctx.Response().Json(http.StatusUnauthorized, httpcontract.Json{
			"status":  "error",
			"message": "Email atau password salah.",
		})
	}

	// Generate JWT token
	token, err := facades.Auth(ctx).Login(user)
	if err != nil {
		return ctx.Response().Json(http.StatusInternalServerError, httpcontract.Json{
			"status":  "error",
			"message": "Gagal membuat sesi login. Coba lagi.",
		})
	}

	// FINDING-004 FIX: SameSite=Strict
	// FINDING-005 FIX: Secure=true
	ctx.Response().Cookie(httpcontract.Cookie{
		Name:     "auth_token",
		Value:    token,
		MaxAge:   60 * 60 * 24 * 7,
		Path:     "/",
		HttpOnly: true,
		Secure:   true, // must be true in production with HTTPS
		SameSite: "Strict",
	})

	return ctx.Response().Json(http.StatusOK, httpcontract.Json{
		"status":  "success",
		"message": "Login berhasil",
		"user": httpcontract.Json{
			"id":    user.ID,
			"name":  user.Name,
			"email": user.Email,
			"role":  user.Role,
		},
	})
}

// Register handles POST /api/register
// Creates a new user and logs them in automatically.
func (c *AuthController) Register(ctx httpcontract.Context) httpcontract.Response {
	name := ctx.Request().Input("name")
	email := ctx.Request().Input("email")
	password := ctx.Request().Input("password")

	if name == "" || email == "" || password == "" {
		return ctx.Response().Json(http.StatusBadRequest, httpcontract.Json{
			"status":  "error",
			"message": "Semua field (nama, email, password) wajib diisi.",
		})
	}

	// FINDING-015 FIX: Enforce stronger password policy
	if err := validatePasswordStrength(password); err != nil {
		return ctx.Response().Json(http.StatusBadRequest, httpcontract.Json{
			"status":  "error",
			"message": err.Error(),
		})
	}

	user, err := c.authService.CreateUser(name, email, password)
	if err != nil {
		return ctx.Response().Json(http.StatusConflict, httpcontract.Json{
			"status":  "error",
			"message": err.Error(),
		})
	}

	// Auto-login after registration
	token, err := facades.Auth(ctx).Login(user)
	if err != nil {
		return ctx.Response().Json(http.StatusInternalServerError, httpcontract.Json{
			"status":  "error",
			"message": "Akun berhasil dibuat, silakan login secara manual.",
		})
	}

	ctx.Response().Cookie(httpcontract.Cookie{
		Name:     "auth_token",
		Value:    token,
		MaxAge:   60 * 60 * 24 * 7,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: "Strict",
	})

	return ctx.Response().Json(http.StatusOK, httpcontract.Json{
		"status":  "success",
		"message": "Pendaftaran berhasil! Selamat datang, " + user.Name,
		"user": httpcontract.Json{
			"id":    user.ID,
			"name":  user.Name,
			"email": user.Email,
			"role":  user.Role,
		},
	})
}

// Logout handles POST /logout
// Invalidates the JWT and clears the auth cookie.
func (c *AuthController) Logout(ctx httpcontract.Context) httpcontract.Response {
	// Invalidate JWT
	_ = facades.Auth(ctx).Logout()

	// Clear cookie
	ctx.Response().Cookie(httpcontract.Cookie{
		Name:     "auth_token",
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		HttpOnly: true,
	})

	return ctx.Response().Json(http.StatusOK, httpcontract.Json{
		"status":  "success",
		"message": "Logout berhasil",
	})
}

// validatePasswordStrength enforces password requirements.
// FINDING-015 FIX: min 8 chars, at least one uppercase, one lowercase, one digit.
func validatePasswordStrength(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password minimal 8 karakter")
	}
	if !regexp.MustCompile(`[A-Z]`).MatchString(password) {
		return fmt.Errorf("password harus mengandung setidaknya satu huruf kapital")
	}
	if !regexp.MustCompile(`[a-z]`).MatchString(password) {
		return fmt.Errorf("password harus mengandung setidaknya satu huruf kecil")
	}
	if !regexp.MustCompile(`[0-9]`).MatchString(password) {
		return fmt.Errorf("password harus mengandung setidaknya satu angka")
	}
	return nil
}

