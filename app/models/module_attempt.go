package models

import (
	"github.com/goravel/framework/database/orm"
)

// ModuleAttempt menyimpan satu percobaan Active Recall / Feynman Technique
// milik user untuk sebuah Module, beserta hasil evaluasi AI (Fase 5).
type ModuleAttempt struct {
	orm.Model
	ModuleID         uint   `json:"module_id"`
	UserExplanation  string `json:"user_explanation"`
	FeynmanScore     *int   `json:"feynman_score,omitempty"`
	// sebagai string JSON.
	Feedback string `json:"feedback"`
}
