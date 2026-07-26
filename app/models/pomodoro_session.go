package models

import (
	"github.com/goravel/framework/database/orm"
)

type PomodoroSession struct {
	orm.Model
	UserID    uint   `json:"user_id"`
	Phase     string `json:"phase"` // e.g. "focus", "short_break", "long_break"
	Duration  int    `json:"duration"` // Duration in minutes
	User      User   `json:"-"`
}
