package models

import (
	"database/sql/driver"
	"encoding/json"

	"github.com/goravel/framework/database/orm"
)

// Project merepresentasikan satu "topik belajar" yang dibuat user
// (disebut "Topic" di frontend saat ini — nama field JSON sengaja
// dipertahankan agar kompatibel dengan Home.jsx / AddTopic.jsx yang sudah ada).
type Project struct {
	orm.Model
	UserID        *uint           `json:"user_id,omitempty"`
	Title         string          `json:"title"`
	SourceFileURL string          `json:"source_file_url,omitempty"`
	Methods       *ProjectMethods `gorm:"type:json;serializer:json" json:"methods,omitempty"`

	// Completed & Total: field legacy untuk kompatibilitas dengan UI progress bar
	// yang sudah ada. Ke depan (setelah Fase 2-3 AI selesai) nilai ini bisa
	// dihitung otomatis dari jumlah modules dengan status "mastered".
	Completed int `json:"completed"`
	Total     int `json:"total"`

	orm.SoftDeletes
}

// ProjectMethods merepresentasikan toggle metode belajar yang diaktifkan
// user untuk sebuah project (Feynman, Pomodoro, Spaced Repetition).
//
// Diimplementasikan sebagai custom JSON type mengikuti pola resmi Goravel
// (https://www.goravel.dev/orm/getting-started.html#json-field): dengan
// Value()/Scan(), Orm otomatis marshal/unmarshal struct ini ke/dari kolom
// JSON di DB — tidak perlu json.Marshal/Unmarshal manual berulang di controller.
type ProjectMethods struct {
	Feynman          bool `json:"feynman"`
	Pomodoro         bool `json:"pomodoro"`
	SpacedRepetition bool `json:"spaced_repetition"`
}

// Value mengimplementasikan driver.Valuer — dipanggil Orm saat menulis ke DB.
func (m *ProjectMethods) Value() (driver.Value, error) {
	return json.Marshal(m)
}

// Scan mengimplementasikan sql.Scanner — dipanggil Orm saat membaca dari DB.
func (m *ProjectMethods) Scan(value any) error {
	data, ok := value.([]byte)
	if !ok || len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, m)
}
