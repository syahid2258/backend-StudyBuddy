package models

import (
	"time"

	"github.com/goravel/framework/database/orm"
)

// Flashcard merepresentasikan satu kartu spaced repetition (mirip Anki)
// milik sebuah Project. Penjadwalan diatur oleh algoritma SM-2 sederhana
// di service layer (Fase 4), bukan oleh AI generatif.
type Flashcard struct {
	orm.Model
	ProjectID      uint       `json:"project_id"`
	FrontText      string     `json:"front_text"`
	BackText       string     `json:"back_text"`
	EaseFactor     float64    `json:"-"`
	IntervalDays   int        `json:"-"`
	NextReviewDate *time.Time `json:"next_review_date,omitempty"`
}
