package controllers

import (
	httpcontract "github.com/goravel/framework/contracts/http"

	"goravel/app/facades"
	"goravel/app/models"
)

type HomeController struct{}

func NewHomeController() *HomeController {
	return &HomeController{}
}

// Index renders the home page with the authenticated user's topics/projects.
func (c *HomeController) Index(ctx httpcontract.Context) httpcontract.Response {
	userID := ctx.Value("auth_user_id")

	var user models.User
	if userID != nil {
		facades.Orm().Query().Find(&user, userID)
	}

	var projects []models.Project
	query := facades.Orm().Query()
	if userID != nil {
		query = query.Where("user_id", userID)
	}
	_ = query.OrderBy("created_at", "desc").Get(&projects)

	return ctx.Response().View().Make("home.tmpl", map[string]any{
		"user":     user,
		"projects": projects,
	})
}
