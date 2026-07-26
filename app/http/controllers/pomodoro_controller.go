package controllers

import (
	"goravel/app/facades"
	"goravel/app/models"

	"github.com/goravel/framework/contracts/http"
)

type PomodoroController struct{}

func NewPomodoroController() *PomodoroController {
	return &PomodoroController{}
}

// LogSession mencatat sesi pomodoro yang telah selesai
func (c *PomodoroController) LogSession(ctx http.Context) http.Response {
	userID, ok := ctx.Value("auth_user_id").(uint)
	if !ok || userID == 0 {
		return ctx.Response().Json(http.StatusUnauthorized, http.Json{"error": "Unauthorized"})
	}

	phase := ctx.Request().Input("phase")
	duration := ctx.Request().InputInt("duration")

	if phase == "" || duration == 0 {
		return ctx.Response().Json(http.StatusBadRequest, http.Json{"error": "Phase and duration are required"})
	}

	session := models.PomodoroSession{
		UserID:   userID,
		Phase:    phase,
		Duration: duration,
	}

	if err := facades.Orm().Query().Create(&session); err != nil {
		return ctx.Response().Json(http.StatusInternalServerError, http.Json{"error": "Failed to log session"})
	}

	return ctx.Response().Json(http.StatusOK, http.Json{
		"message": "Pomodoro session logged successfully",
		"data":    session,
	})
}
