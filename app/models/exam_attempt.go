package models

import (
	"github.com/goravel/framework/database/orm"
)

// ExamAttempt menyimpan satu submisi jawaban user untuk sebuah Exam,
// beserta hasil grading AI (final_score & analysis, lihat Fase 6).
type ExamAttempt struct {
	orm.Model
	ExamID     uint    `json:"exam_id"`
	Answers    string  `json:"-"`
	FinalScore *int    `json:"final_score,omitempty"`
	Analysis   *string `json:"analysis,omitempty"`
}
