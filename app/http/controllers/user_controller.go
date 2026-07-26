package controllers

import (
	"net/http"

	httpcontract "github.com/goravel/framework/contracts/http"

	"goravel/app/facades"
	"goravel/app/models"
)

// UserController handles user-related API requests.
// Most user operations are handled by ProfileController for page rendering.
type UserController struct{}

func NewUserController() *UserController {
	return &UserController{}
}

// Index returns a list of all users (admin-use only).
func (r *UserController) Index(ctx httpcontract.Context) httpcontract.Response {
	var users []models.User
	if err := facades.Orm().Query().Get(&users); err != nil {
		return ctx.Response().Json(http.StatusInternalServerError, httpcontract.Json{
			"status":  "error",
			"message": "Gagal mengambil data pengguna.",
		})
	}

	// Strip passwords from response
	type SafeUser struct {
		ID    uint   `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	safeUsers := make([]SafeUser, len(users))
	for i, u := range users {
		safeUsers[i] = SafeUser{ID: u.ID, Name: u.Name, Email: u.Email}
	}

	return ctx.Response().Json(http.StatusOK, httpcontract.Json{
		"status": "success",
		"data":   safeUsers,
	})
}
