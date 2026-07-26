package models

import (
	"github.com/goravel/framework/database/orm"
)

// ChatMessage menyimpan satu giliran percakapan "Tanya AI" milik sebuah
// Module. Disimpan per baris (bukan satu blob JSON) supaya gampang di-query
// terurut dan gampang dibangun ulang jadi histori percakapan tiap request.
type ChatMessage struct {
	orm.Model
	ModuleID uint   `json:"module_id"`
	Role     string `json:"role"` // "user" | "model"
	Content  string `json:"content"`
}
