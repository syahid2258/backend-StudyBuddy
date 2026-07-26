package models

import (
	"github.com/goravel/framework/database/orm"
)

// Exam merepresentasikan satu set soal ujian akhir yang di-generate AI
// untuk sebuah Project (Fase 6). QuestionTypes dan Questions disimpan
// sebagai string JSON — kunci jawaban untuk soal pilihan ganda sengaja
// disembunyikan dari response ke frontend di level controller, bukan
// di level model/DB.
type Exam struct {
	orm.Model
	ProjectID     uint   `json:"project_id"`
	Title         string `json:"title"`
	QuestionTypes string `json:"-"`
	Questions     string `json:"-"`
}
