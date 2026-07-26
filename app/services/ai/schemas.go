package ai

// SyllabusModule merepresentasikan satu sub-materi hasil generate AI,
// sesuai skema di briefing Bagian 3 poin 5.
type SyllabusModule struct {
	Title string `json:"title"`
	Order int    `json:"order"`
}

// SyllabusResponse adalah bentuk terparsing dari JSON yang diminta ke
// Gemini lewat ResponseJsonSchema di GenerateSyllabus.
type SyllabusResponse struct {
	TotalModules int              `json:"total_modules"`
	Modules      []SyllabusModule `json:"modules"`
}

// syllabusJSONSchema adalah JSON Schema yang dikirim ke Gemini supaya
// responsnya dijamin sesuai struktur SyllabusResponse di atas.
// Lihat: https://ai.google.dev/gemini-api/docs/generate-content/structured-output
var syllabusJSONSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"total_modules": map[string]any{
			"type": "integer",
		},
		"modules": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title": map[string]any{"type": "string"},
					"order": map[string]any{"type": "integer"},
				},
				"required": []string{"title", "order"},
			},
		},
	},
	"required": []string{"total_modules", "modules"},
}

// ContentBlock merepresentasikan satu blok konten materi hasil generate AI,
// sesuai skema content_blocks di briefing (Bagian 3 poin 6). Type membedakan
// blok rangkuman biasa ("paragraph") dari blok mnemonic ("jembatan_keledai").
type ContentBlock struct {
	Type  string `json:"type"`
	Title string `json:"title,omitempty"`
	Text  string `json:"text"`
}

// ModuleContentResponse adalah bentuk terparsing dari JSON yang diminta ke
// Gemini lewat moduleContentJSONSchema di GenerateModuleContent.
type ModuleContentResponse struct {
	ContentBlocks []ContentBlock `json:"content_blocks"`
}

// moduleContentJSONSchema adalah JSON Schema untuk fitur Generate Materi (Fase 3).
var moduleContentJSONSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"content_blocks": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"type":  map[string]any{"type": "string", "enum": []string{"paragraph", "jembatan_keledai"}},
					"title": map[string]any{"type": "string"},
					"text":  map[string]any{"type": "string"},
				},
				"required": []string{"type", "text"},
			},
		},
	},
	"required": []string{"content_blocks"},
}

// FeynmanFeedback merepresentasikan feedback terstruktur hasil evaluasi
// Active Recall, sesuai skema di briefing Bagian 3 poin 8.
type FeynmanFeedback struct {
	Pujian     string `json:"pujian"`
	Kekurangan string `json:"kekurangan"`
	Saran      string `json:"saran"`
}

// GeneratedFlashcard adalah flashcard siap-pakai (bukan cuma nama istilah)
// yang dibuat AI dari bagian yang terlewat saat evaluasi Feynman. Ini sedikit
// berbeda dari contoh di briefing (yang cuma array nama istilah) — sengaja
// diperluas jadi front_text/back_text supaya langsung bisa disimpan sebagai
// flashcard yang benar-benar bisa dipakai user, tanpa perlu panggilan AI kedua.
type GeneratedFlashcard struct {
	FrontText string `json:"front_text"`
	BackText  string `json:"back_text"`
}

// FeynmanEvaluationResponse adalah bentuk terparsing dari JSON yang diminta
// ke Gemini lewat feynmanJSONSchema di EvaluateFeynman.
type FeynmanEvaluationResponse struct {
	FeynmanScore       int                  `json:"feynman_score"`
	IsPassed           bool                 `json:"is_passed"`
	Feedback           FeynmanFeedback      `json:"feedback"`
	GenerateFlashcards []GeneratedFlashcard `json:"generate_flashcards"`
}

// feynmanJSONSchema adalah JSON Schema untuk fitur Evaluasi Feynman (Fase 5) —
// bagian paling kritis karena hasilnya mengontrol unlock modul berikutnya.
var feynmanJSONSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"feynman_score": map[string]any{"type": "integer"},
		"is_passed":     map[string]any{"type": "boolean"},
		"feedback": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"pujian":     map[string]any{"type": "string"},
				"kekurangan": map[string]any{"type": "string"},
				"saran":      map[string]any{"type": "string"},
			},
			"required": []string{"pujian", "kekurangan", "saran"},
		},
		"generate_flashcards": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"front_text": map[string]any{"type": "string"},
					"back_text":  map[string]any{"type": "string"},
				},
				"required": []string{"front_text", "back_text"},
			},
		},
	},
	"required": []string{"feynman_score", "is_passed", "feedback", "generate_flashcards"},
}

// ExamQuestion merepresentasikan satu soal ujian hasil generate AI, sesuai
// skema di briefing Bagian 3 poin 9. CorrectAnswer adalah kunci jawaban —
// untuk "multiple_choice" harus persis sama dengan salah satu isi Options;
// untuk "essay" berisi poin kunci yang dipakai sebagai rubrik penilaian.
// Field ini WAJIB disembunyikan dari response ke frontend di level controller.
type ExamQuestion struct {
	ID            string   `json:"id"`
	Type          string   `json:"type"` // "multiple_choice" | "essay"
	Question      string   `json:"question"`
	Options       []string `json:"options,omitempty"`
	CorrectAnswer string   `json:"correct_answer"`
}

// ExamGenerationResponse adalah bentuk terparsing dari JSON yang diminta ke
// Gemini lewat examJSONSchema di GenerateExam.
type ExamGenerationResponse struct {
	Questions []ExamQuestion `json:"questions"`
}

// examJSONSchema adalah JSON Schema untuk fitur Generate Soal Ujian (Fase 6).
var examJSONSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"questions": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":             map[string]any{"type": "string"},
					"type":           map[string]any{"type": "string", "enum": []string{"multiple_choice", "essay"}},
					"question":       map[string]any{"type": "string"},
					"options":        map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
					"correct_answer": map[string]any{"type": "string"},
				},
				"required": []string{"id", "type", "question", "correct_answer"},
			},
		},
	},
	"required": []string{"questions"},
}

// EssayAnswer adalah pasangan soal essay + kunci jawaban + jawaban siswa,
// dipakai sebagai input grading di GradeEssayAnswers.
type EssayAnswer struct {
	Question      string
	ModelAnswer   string
	StudentAnswer string
}

// ExamGradingResponse adalah bentuk terparsing dari JSON yang diminta ke
// Gemini lewat examGradingJSONSchema di GradeEssayAnswers.
type ExamGradingResponse struct {
	FinalScore int    `json:"final_score"`
	Analysis   string `json:"analysis"`
}

// examGradingJSONSchema adalah JSON Schema untuk fitur Grading Ujian (Fase 6).
var examGradingJSONSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"final_score": map[string]any{"type": "integer"},
		"analysis":    map[string]any{"type": "string"},
	},
	"required": []string{"final_score", "analysis"},
}
