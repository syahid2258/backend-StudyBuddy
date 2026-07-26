package controllers

import (
	"fmt"
	"net/mail"
	"net/http"
	"regexp"

	httpcontract "github.com/goravel/framework/contracts/http"

	"goravel/app/facades"
	"goravel/app/models"
)

type ProfileController struct{}

func NewProfileController() *ProfileController {
	return &ProfileController{}
}

// Show renders the profile page with the authenticated user's data.
func (c *ProfileController) Show(ctx httpcontract.Context) httpcontract.Response {
	userID := ctx.Value("auth_user_id")

	var user models.User
	if userID != nil {
		facades.Orm().Query().Find(&user, userID)
	}

	var projectCount int64
	projectCount, _ = facades.Orm().Query().Model(&models.Project{}).Where("user_id", userID).Count()

	return ctx.Response().View().Make("profile.tmpl", map[string]any{
		"user":         user,
		"projectCount": projectCount,
	})
}

// GetProfile handles GET /api/profile/me — returns current user as JSON.
func (c *ProfileController) GetProfile(ctx httpcontract.Context) httpcontract.Response {
	userID := ctx.Value("auth_user_id")

	var user models.User
	if userID != nil {
		if err := facades.Orm().Query().Find(&user, userID); err != nil {
			return ctx.Response().Json(http.StatusInternalServerError, httpcontract.Json{
				"status":  "error",
				"message": "Gagal mengambil data profil.",
			})
		}
	}

	if user.ID == 0 {
		return ctx.Response().Json(http.StatusNotFound, httpcontract.Json{
			"status":  "error",
			"message": "Pengguna tidak ditemukan.",
		})
	}

	return ctx.Response().Json(http.StatusOK, httpcontract.Json{
		"status": "success",
		"user": httpcontract.Json{
			"id":         user.ID,
			"name":       user.Name,
			"email":      user.Email,
			"role":       user.Role,
			"created_at": user.CreatedAt,
		},
	})
}

// isValidEmail validates email format using stdlib net/mail.
func isValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil && regexp.MustCompile(`^[^@]+@[^@]+\.[^@]+$`).MatchString(email)
}

// UpdateProfile handles PUT /api/profile/me — updates name and/or email.
// FINDING-011 FIX: Added email format validation and uniqueness check.
func (c *ProfileController) UpdateProfile(ctx httpcontract.Context) httpcontract.Response {
	userID := ctx.Value("auth_user_id")

	var user models.User
	if userID == nil {
		return ctx.Response().Json(http.StatusUnauthorized, httpcontract.Json{
			"status":  "error",
			"message": "Tidak terautentikasi.",
		})
	}

	if err := facades.Orm().Query().Find(&user, userID); err != nil || user.ID == 0 {
		return ctx.Response().Json(http.StatusNotFound, httpcontract.Json{
			"status":  "error",
			"message": "Pengguna tidak ditemukan.",
		})
	}

	if name := ctx.Request().Input("name"); name != "" {
		if len(name) < 2 || len(name) > 100 {
			return ctx.Response().Json(http.StatusBadRequest, httpcontract.Json{
				"status":  "error",
				"message": "Nama harus antara 2-100 karakter.",
			})
		}
		user.Name = name
	}

	if email := ctx.Request().Input("email"); email != "" {
		// Validate format
		if !isValidEmail(email) {
			return ctx.Response().Json(http.StatusBadRequest, httpcontract.Json{
				"status":  "error",
				"message": "Format email tidak valid.",
			})
		}
		// Uniqueness check — prevent email takeover
		var existing models.User
		_ = facades.Orm().Query().Where("email", email).First(&existing)
		if existing.ID != 0 && existing.ID != user.ID {
			return ctx.Response().Json(http.StatusConflict, httpcontract.Json{
				"status":  "error",
				"message": "Email sudah digunakan oleh akun lain.",
			})
		}
		user.Email = email
	}

	if err := facades.Orm().Query().Save(&user); err != nil {
		// FINDING-014 FIX: Never return raw error to client
		facades.Log().Errorf("UpdateProfile error for user %d: %v", user.ID, err)
		return ctx.Response().Json(http.StatusInternalServerError, httpcontract.Json{
			"status":  "error",
			"message": fmt.Sprintf("Gagal memperbarui profil."),
		})
	}

	return ctx.Response().Json(http.StatusOK, httpcontract.Json{
		"status": "success",
		"user": httpcontract.Json{
			"id":    user.ID,
			"name":  user.Name,
			"email": user.Email,
		},
	})
}
