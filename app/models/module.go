package models

import (
	"database/sql/driver"
	"encoding/json"

	"github.com/goravel/framework/database/orm"
)

// Module merepresentasikan satu sub-materi dalam rute belajar (silabus)
// sebuah Project. Modul pertama dibuat unlocked, sisanya locked, dan
// terbuka berurutan setelah user lulus evaluasi Feynman modul sebelumnya.
type Module struct {
	orm.Model
	ProjectID uint   `json:"project_id"`
	Title     string `json:"title"`
	Order     int    `json:"order"`
	IsLocked  bool   `json:"is_locked"`
	// Status: not_started | in_progress | mastered
	Status string `json:"status"`
	// ContentBlocks: hasil generate AI (rangkuman + jembatan_keledai), diisi
	// on-demand saat modul pertama kali dibuka (lihat Fase 3). Nullable
	// (pointer) karena kosong sampai saat itu.
	ContentBlocks *ContentBlockList `gorm:"type:json;serializer:json" json:"content_blocks,omitempty"`
	ActiveRecall  *ActiveRecall     `json:"active_recall,omitempty"`
	Attempts      []ModuleAttempt   `json:"attempts,omitempty"`
}

// ContentBlock merepresentasikan satu blok konten materi, sesuai skema
// content_blocks di briefing (Bagian 3 poin 6). Type membedakan blok
// rangkuman biasa ("paragraph") dari blok mnemonic ("jembatan_keledai").
type ContentBlock struct {
	Type  string `json:"type"`
	Title string `json:"title,omitempty"`
	Text  string `json:"text"`
}

// ContentBlockList adalah wrapper slice supaya bisa diimplementasikan
// sebagai custom JSON type (pola resmi Goravel), dipakai langsung sebagai
// tipe field ContentBlocks di atas.
type ContentBlockList []ContentBlock

// Value mengimplementasikan driver.Valuer — dipanggil Orm saat menulis ke DB.
func (c *ContentBlockList) Value() (driver.Value, error) {
	return json.Marshal(c)
}

// Scan mengimplementasikan sql.Scanner — dipanggil Orm saat membaca dari DB.
func (c *ContentBlockList) Scan(value any) error {
	data, ok := value.([]byte)
	if !ok || len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, c)
}
