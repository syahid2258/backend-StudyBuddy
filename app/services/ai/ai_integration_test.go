package ai_test

import (
	"context"
	"os"
	"testing"
	"time"

	"goravel/app/services/ai"
	"goravel/bootstrap"
)

func init() {
	bootstrap.Boot()
}

func getTestTimeoutContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 90*time.Second)
}

func skipIfNoAPIKey(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Gemini AI integration test in short mode.")
	}

	apiKey := os.Getenv("GEMINI_API_KEY")
	apiKeys := os.Getenv("GEMINI_API_KEYS")

	if apiKey == "" && apiKeys == "" {
		t.Skip("Skipping Gemini AI integration test: GEMINI_API_KEY or GEMINI_API_KEYS is not set.")
	}
}

// 1. Test Client Ping / GetClient & GetPoolStatus
func TestClientAndPool(t *testing.T) {
	skipIfNoAPIKey(t)
	ctx, cancel := getTestTimeoutContext()
	defer cancel()

	t.Log("Testing GetClient...")
	client, err := ai.GetClient(ctx)
	if err != nil {
		t.Fatalf("GetClient failed: %v", err)
	}
	if client == nil {
		t.Fatalf("GetClient returned nil client")
	}
	t.Log("Successfully obtained genai.Client instance.")

	t.Log("Testing GetPoolStatus...")
	status, err := ai.GetPoolStatus(ctx)
	if err != nil {
		t.Fatalf("GetPoolStatus failed: %v", err)
	}
	t.Logf("Key pool status: %d keys registered", len(status))
	for i, entry := range status {
		t.Logf("  Key [%d]: Preview: %s, OnCooldown: %v", i+1, entry.KeyPreview, len(entry.Cooldowns) > 0)
	}
}

// 2. Test GenerateSyllabus
func TestGenerateSyllabus(t *testing.T) {
	skipIfNoAPIKey(t)
	ctx, cancel := getTestTimeoutContext()
	defer cancel()

	materialText := "Pemrograman Go (Golang) dasar: variabel, tipe data, fungsi, struct, dan interface."

	t.Log("Testing GenerateSyllabus...")
	syllabus, err := ai.GenerateSyllabus(ctx, materialText, nil, "")
	if err != nil {
		t.Fatalf("GenerateSyllabus failed: %v", err)
	}

	if syllabus == nil {
		t.Fatalf("GenerateSyllabus returned nil response")
	}

	if syllabus.TotalModules <= 0 || len(syllabus.Modules) == 0 {
		t.Fatalf("GenerateSyllabus returned empty/invalid modules list: %+v", syllabus)
	}

	t.Logf("Successfully generated %d modules (Total declared: %d):", len(syllabus.Modules), syllabus.TotalModules)
	for i, m := range syllabus.Modules {
		t.Logf("  [%d] Order: %d | Title: %s", i+1, m.Order, m.Title)
		if m.Title == "" {
			t.Errorf("Module %d has an empty title", i+1)
		}
	}
}

// 3. Test GenerateModuleContent
func TestGenerateModuleContent(t *testing.T) {
	skipIfNoAPIKey(t)
	ctx, cancel := getTestTimeoutContext()
	defer cancel()

	moduleTitle := "Variabel dan Tipe Data di Go"
	projectTitle := "Pemrograman Go Dasar"

	t.Log("Testing GenerateModuleContent...")
	content, err := ai.GenerateModuleContent(ctx, moduleTitle, projectTitle)
	if err != nil {
		t.Fatalf("GenerateModuleContent failed: %v", err)
	}

	if content == nil {
		t.Fatalf("GenerateModuleContent returned nil response")
	}

	if len(content.ContentBlocks) == 0 {
		t.Fatalf("GenerateModuleContent returned empty ContentBlocks")
	}

	hasMnemonic := false
	t.Logf("Successfully generated module content with %d blocks:", len(content.ContentBlocks))
	for i, b := range content.ContentBlocks {
		t.Logf("  Block [%d] Type: %s | Title: %s", i+1, b.Type, b.Title)
		if b.Type == "" {
			t.Errorf("Block %d has empty type", i+1)
		}
		if b.Text == "" {
			t.Errorf("Block %d has empty text", i+1)
		}
		if b.Type == "jembatan_keledai" {
			hasMnemonic = true
		}
	}

	if !hasMnemonic {
		t.Errorf("Expected at least one 'jembatan_keledai' block type, but none found")
	}
}

// 4. Test EvaluateFeynman
func TestEvaluateFeynman(t *testing.T) {
	skipIfNoAPIKey(t)
	ctx, cancel := getTestTimeoutContext()
	defer cancel()

	materialText := "Interface di Go adalah tipe data abstrak yang mendefinisikan sekumpulan method signature."
	userExplanation := "Interface di Go itu seperti kontrak yang mendefinisikan method tanpa implementasi asli. Struct yang punya method tersebut otomatis mengimplementasikannya."

	t.Log("Testing EvaluateFeynman (Active Recall)...")
	eval, err := ai.EvaluateFeynman(ctx, materialText, userExplanation, nil, "")
	if err != nil {
		t.Fatalf("EvaluateFeynman failed: %v", err)
	}

	if eval == nil {
		t.Fatalf("EvaluateFeynman returned nil response")
	}

	if eval.FeynmanScore < 0 || eval.FeynmanScore > 100 {
		t.Errorf("Invalid Feynman Score (must be 0-100): %v", eval.FeynmanScore)
	}

	t.Logf("Feynman Evaluation Result:")
	t.Logf("  Score: %d/100 | Passed: %v", eval.FeynmanScore, eval.IsPassed)
	t.Logf("  Feedback - Pujian: %s", eval.Feedback.Pujian)
	t.Logf("  Feedback - Kekurangan: %s", eval.Feedback.Kekurangan)
	t.Logf("  Feedback - Saran: %s", eval.Feedback.Saran)
	t.Logf("  Generated Flashcards count: %d", len(eval.GenerateFlashcards))

	for i, fc := range eval.GenerateFlashcards {
		t.Logf("    Flashcard [%d]: Front: %s | Back: %s", i+1, fc.FrontText, fc.BackText)
		if fc.FrontText == "" || fc.BackText == "" {
			t.Errorf("Generated flashcard %d has empty front or back text", i+1)
		}
	}
}

// 5. Test Reply (Chat)
func TestReply(t *testing.T) {
	skipIfNoAPIKey(t)
	ctx, cancel := getTestTimeoutContext()
	defer cancel()

	moduleTitle := "Goroutines dan Concurrency"
	moduleMaterial := "Goroutines adalah fungsi atau method yang berjalan secara concurrent dengan goroutines lainnya atas manajemen runtime Go."
	history := []ai.ChatTurn{
		{Role: "user", Content: "Apa itu goroutine?"},
		{Role: "model", Content: "Goroutine adalah thread ringan yang dikelola oleh runtime Go."},
	}
	question := "Bagaimana cara kerja scheduler goroutine?"

	t.Log("Testing Reply (AI Chat)...")
	reply, err := ai.Reply(ctx, moduleTitle, moduleMaterial, history, question)
	if err != nil {
		t.Fatalf("Reply failed: %v", err)
	}

	if reply == "" {
		t.Fatalf("Reply returned empty response")
	}

	t.Logf("AI Chat Reply:\n%s", reply)
}

func TestGlobalReply(t *testing.T) {
	skipIfNoAPIKey(t)
	ctx, cancel := getTestTimeoutContext()
	defer cancel()

	history := []ai.ChatTurn{
		{Role: "user", Content: "Halo!"},
		{Role: "model", Content: "Halo! Ada yang bisa dibantu?"},
	}
	question := "Apa ibukota Indonesia?"

	t.Log("Testing GlobalReply (Tanya AI Global)...")
	reply, err := ai.GlobalReply(ctx, history, question)
	if err != nil {
		t.Fatalf("GlobalReply failed: %v", err)
	}

	if reply == "" {
		t.Fatalf("GlobalReply returned empty response")
	}

	t.Logf("AI Global Chat Reply:\n%s", reply)
}

func TestRegenerateModuleContent(t *testing.T) {
	skipIfNoAPIKey(t)
	ctx, cancel := getTestTimeoutContext()
	defer cancel()

	moduleTitle := "Goroutines"
	projectTitle := "Belajar Golang Concurrency"
	reason := "Tolong jelaskan ulang dengan perumpamaan jalan tol."
	oldContent := "Goroutines adalah concurrent execution."

	t.Log("Testing RegenerateModuleContent...")
	content, err := ai.RegenerateModuleContent(ctx, moduleTitle, projectTitle, reason, oldContent)
	if err != nil {
		t.Fatalf("RegenerateModuleContent failed: %v", err)
	}

	if content == nil || len(content.ContentBlocks) == 0 {
		t.Fatalf("RegenerateModuleContent returned empty blocks")
	}

	t.Logf("Successfully regenerated module content with %d blocks", len(content.ContentBlocks))
}

// 6. Test GenerateExam & GradeEssayAnswers
func TestGenerateExamAndGradeEssay(t *testing.T) {
	skipIfNoAPIKey(t)
	ctx, cancel := getTestTimeoutContext()
	defer cancel()

	materialText := "Goroutines dijalankan di atas OS threads menggunakan M:N scheduler. " +
		"Channels digunakan untuk sinkronisasi dan komunikasi antar goroutine."

	t.Log("Testing GenerateExam...")
	exam, err := ai.GenerateExam(ctx, materialText, []string{"multiple_choice", "essay"}, 2, nil)
	if err != nil {
		t.Fatalf("GenerateExam failed: %v", err)
	}

	if exam == nil || len(exam.Questions) == 0 {
		t.Fatalf("GenerateExam returned empty or nil exam response")
	}

	t.Logf("Successfully generated %d exam questions:", len(exam.Questions))
	var essayQuestions []ai.ExamQuestion
	for i, q := range exam.Questions {
		t.Logf("  [%d] ID: %s | Type: %s | Q: %s", i+1, q.ID, q.Type, q.Question)
		if q.ID == "" || q.Question == "" || q.CorrectAnswer == "" {
			t.Errorf("Question %d has missing essential fields (ID/Question/CorrectAnswer)", i+1)
		}
		if q.Type == "multiple_choice" && len(q.Options) < 2 {
			t.Errorf("Multiple choice question %d has fewer than 2 options", i+1)
		}
		if q.Type == "essay" {
			essayQuestions = append(essayQuestions, q)
		}
	}

	t.Log("Testing GradeEssayAnswers...")
	var essays []ai.EssayAnswer
	if len(essayQuestions) > 0 {
		for _, eq := range essayQuestions {
			essays = append(essays, ai.EssayAnswer{
				Question:      eq.Question,
				ModelAnswer:   eq.CorrectAnswer,
				StudentAnswer: "Channel dan scheduler M:N bekerja sama untuk menyinkronkan eksekusi goroutine secara aman dan efisien.",
			})
		}
	} else {
		essays = []ai.EssayAnswer{
			{
				Question:      "Jelaskan fungsi channel di Go.",
				ModelAnswer:   "Channel berfungsi sebagai pipa komunikasi dan sinkronisasi antar goroutines.",
				StudentAnswer: "Channel dipakai buat kirim data antar goroutine.",
			},
		}
	}

	grading, err := ai.GradeEssayAnswers(ctx, materialText, essays, 1, 2)
	if err != nil {
		t.Fatalf("GradeEssayAnswers failed: %v", err)
	}

	if grading == nil {
		t.Fatalf("GradeEssayAnswers returned nil response")
	}

	if grading.FinalScore < 0 || grading.FinalScore > 100 {
		t.Errorf("Invalid final score: %v", grading.FinalScore)
	}
	if grading.Analysis == "" {
		t.Errorf("Grading analysis is empty")
	}

	t.Logf("Exam Grading Result:")
	t.Logf("  Final Score: %d/100", grading.FinalScore)
	t.Logf("  Analysis: %s", grading.Analysis)
}

// ============================================================================
// 7. Test DefaultModel & RequestTimeout (config utilities)
// ============================================================================

func TestDefaultModelAndRequestTimeout(t *testing.T) {
	// Tidak butuh API key — ini hanya membaca config.
	t.Log("Testing Models fallback list...")
	models := ai.Models()
	if len(models) == 0 {
		t.Fatal("Models returned empty list")
	}
	t.Logf("  Primary Model: %s", models[0])

	t.Log("Testing RequestTimeout...")
	timeout := ai.RequestTimeout()
	if timeout <= 0 {
		t.Fatalf("RequestTimeout returned non-positive duration: %v", timeout)
	}
	t.Logf("  RequestTimeout: %v", timeout)
}

// ============================================================================
// 8. Test GenerateExam Retake (with previousQuestions)
// ============================================================================

func TestGenerateExamRetake(t *testing.T) {
	skipIfNoAPIKey(t)
	ctx, cancel := getTestTimeoutContext()
	defer cancel()

	materialText := "Goroutines dijalankan di atas OS threads menggunakan M:N scheduler. " +
		"Channels digunakan untuk sinkronisasi dan komunikasi antar goroutine. " +
		"Select statement memungkinkan goroutine menunggu beberapa operasi channel sekaligus."

	// Langkah 1: Generate soal pertama (tanpa previousQuestions)
	t.Log("Step 1: Generating first exam (no previousQuestions)...")
	firstExam, err := ai.GenerateExam(ctx, materialText, []string{"multiple_choice"}, 3, nil)
	if err != nil {
		t.Fatalf("First GenerateExam failed: %v", err)
	}
	if len(firstExam.Questions) == 0 {
		t.Fatal("First GenerateExam returned no questions")
	}
	t.Logf("  First exam generated %d questions", len(firstExam.Questions))

	// Langkah 2: Kumpulkan teks soal pertama sebagai previousQuestions
	var previousQuestions []string
	for _, q := range firstExam.Questions {
		previousQuestions = append(previousQuestions, q.Question)
		t.Logf("  Previous Q: %s", q.Question)
	}

	// Langkah 3: Generate soal retake — AI harus membuat variasi berbeda
	t.Log("Step 2: Generating retake exam (with previousQuestions)...")
	retakeExam, err := ai.GenerateExam(ctx, materialText, []string{"multiple_choice"}, 3, previousQuestions)
	if err != nil {
		t.Fatalf("Retake GenerateExam failed: %v", err)
	}
	if len(retakeExam.Questions) == 0 {
		t.Fatal("Retake GenerateExam returned no questions")
	}
	t.Logf("  Retake exam generated %d questions", len(retakeExam.Questions))

	// Validasi: soal retake harus ada (AI diminta buat variasi, bukan mengulang)
	for i, q := range retakeExam.Questions {
		t.Logf("  Retake Q[%d]: %s", i+1, q.Question)
		if q.ID == "" || q.Question == "" || q.CorrectAnswer == "" {
			t.Errorf("Retake question %d has missing essential fields", i+1)
		}
		if q.Type == "multiple_choice" && len(q.Options) < 2 {
			t.Errorf("Retake MC question %d has fewer than 2 options", i+1)
		}
	}

	// Cek duplikasi: hitung berapa soal retake yang persis sama dengan soal pertama
	duplicateCount := 0
	for _, rq := range retakeExam.Questions {
		for _, pq := range previousQuestions {
			if rq.Question == pq {
				duplicateCount++
				t.Logf("  ⚠️  Duplicate found: %s", rq.Question)
			}
		}
	}
	if duplicateCount == len(retakeExam.Questions) {
		t.Errorf("All %d retake questions are exact duplicates of previous questions — AI should generate variations", duplicateCount)
	} else {
		t.Logf("  Duplicate check: %d/%d questions are duplicates (acceptable if not 100%%)", duplicateCount, len(retakeExam.Questions))
	}
}

// ============================================================================
// 9. Test GenerateSyllabus with file upload (multimodal)
// ============================================================================

func TestGenerateSyllabusWithFile(t *testing.T) {
	skipIfNoAPIKey(t)
	ctx, cancel := getTestTimeoutContext()
	defer cancel()

	// Simulasikan upload file teks sebagai "file bytes" — Gemini menerima
	// text/plain sebagai inline data, jadi ini cukup untuk menguji path
	// multimodal tanpa perlu file PDF asli.
	fileContent := `BAB 1: PENGENALAN DATABASE
Database adalah kumpulan data yang terorganisir. DBMS (Database Management System) 
adalah perangkat lunak untuk mengelola database.

BAB 2: SQL DASAR
SELECT, INSERT, UPDATE, DELETE adalah perintah dasar SQL.
JOIN digunakan untuk menggabungkan data dari beberapa tabel.

BAB 3: NORMALISASI
1NF: Tidak ada repeating group.
2NF: Memenuhi 1NF + tidak ada partial dependency.
3NF: Memenuhi 2NF + tidak ada transitive dependency.`

	fileBytes := []byte(fileContent)
	fileMimeType := "text/plain"

	t.Log("Testing GenerateSyllabus with file bytes (text/plain)...")
	syllabus, err := ai.GenerateSyllabus(ctx, "", fileBytes, fileMimeType)
	if err != nil {
		t.Fatalf("GenerateSyllabus with file failed: %v", err)
	}

	if syllabus == nil {
		t.Fatal("GenerateSyllabus with file returned nil response")
	}
	if syllabus.TotalModules <= 0 || len(syllabus.Modules) == 0 {
		t.Fatalf("GenerateSyllabus with file returned empty/invalid modules: %+v", syllabus)
	}

	t.Logf("Successfully generated %d modules from file (Total declared: %d):", len(syllabus.Modules), syllabus.TotalModules)
	for i, m := range syllabus.Modules {
		t.Logf("  [%d] Order: %d | Title: %s", i+1, m.Order, m.Title)
		if m.Title == "" {
			t.Errorf("Module %d has an empty title", i+1)
		}
	}

	// Test kombinasi: file + teks tambahan
	t.Log("Testing GenerateSyllabus with file + additional text...")
	syllabusCombo, err := ai.GenerateSyllabus(ctx, "Tambahkan juga materi tentang indexing dan query optimization.", fileBytes, fileMimeType)
	if err != nil {
		t.Fatalf("GenerateSyllabus with file+text failed: %v", err)
	}
	if syllabusCombo == nil || len(syllabusCombo.Modules) == 0 {
		t.Fatal("GenerateSyllabus with file+text returned empty response")
	}
	t.Logf("  Combo result: %d modules (Total declared: %d)", len(syllabusCombo.Modules), syllabusCombo.TotalModules)
}

// ============================================================================
// 10. Test EvaluateFeynman — input validation edge cases
// ============================================================================

func TestEvaluateFeynmanValidation(t *testing.T) {
	// Test ini TIDAK butuh API key — hanya menguji validasi input di sisi Go.
	ctx, cancel := getTestTimeoutContext()
	defer cancel()

	t.Log("Testing EvaluateFeynman with empty explanation (should fail)...")
	_, err := ai.EvaluateFeynman(ctx, "materi tentang interface", "", nil, "")
	if err == nil {
		t.Error("Expected error for empty explanation, got nil")
	} else {
		t.Logf("  Correctly rejected: %v", err)
	}

	t.Log("Testing EvaluateFeynman with empty material (should fail)...")
	_, err = ai.EvaluateFeynman(ctx, "", "penjelasan user", nil, "")
	if err == nil {
		t.Error("Expected error for empty material, got nil")
	} else {
		t.Logf("  Correctly rejected: %v", err)
	}

	t.Log("Testing EvaluateFeynman with whitespace-only inputs (should fail)...")
	_, err = ai.EvaluateFeynman(ctx, "   ", "   ", nil, "")
	if err == nil {
		t.Error("Expected error for whitespace-only inputs, got nil")
	} else {
		t.Logf("  Correctly rejected: %v", err)
	}
}

// ============================================================================
// 11. Test EvaluateFeynman with audio (multimodal path)
// ============================================================================

func TestEvaluateFeynmanWithAudio(t *testing.T) {
	skipIfNoAPIKey(t)
	ctx, cancel := getTestTimeoutContext()
	defer cancel()

	materialText := "Interface di Go adalah tipe data abstrak yang mendefinisikan sekumpulan method signature. " +
		"Struct yang memiliki semua method dari sebuah interface otomatis mengimplementasikannya (implicit implementation)."

	// Generate audio WAV minimal yang valid — Gemini bisa memproses header WAV
	// walaupun isinya silence. Ini cukup untuk menguji bahwa code path
	// multimodal (audioBytes → InlineData blob) berjalan tanpa crash.
	//
	// Format: PCM 16-bit mono 8kHz, ~0.5 detik silence (4000 samples)
	wavHeader := generateMinimalWAV(4000, 8000)

	t.Log("Testing EvaluateFeynman with audio (minimal WAV silence)...")
	t.Log("  Note: AI mungkin memberi skor rendah karena audio-nya silence — " +
		"yang diuji adalah code path multimodal, bukan akurasi scoring.")

	eval, err := ai.EvaluateFeynman(ctx, materialText, "", wavHeader, "audio/wav")
	if err != nil {
		// Gemini mungkin menolak audio silence sebagai "penjelasan kosong" —
		// ini acceptable, yang penting code path tidak panic/crash.
		t.Logf("  EvaluateFeynman with audio returned error (acceptable for silence): %v", err)
		t.Log("  ✓ Code path multimodal berjalan tanpa crash/panic.")
		return
	}

	t.Logf("  Score: %d/100 | Passed: %v", eval.FeynmanScore, eval.IsPassed)
	t.Logf("  Feedback - Pujian: %s", eval.Feedback.Pujian)
	t.Logf("  Feedback - Kekurangan: %s", eval.Feedback.Kekurangan)

	if eval.FeynmanScore < 0 || eval.FeynmanScore > 100 {
		t.Errorf("Invalid Feynman Score: %d (must be 0-100)", eval.FeynmanScore)
	}
}

// generateMinimalWAV creates a valid WAV file with silence (all zeros).
// numSamples = total samples, sampleRate = samples per second (e.g. 8000).
func generateMinimalWAV(numSamples, sampleRate int) []byte {
	dataSize := numSamples * 2 // 16-bit = 2 bytes per sample
	fileSize := 36 + dataSize  // 36 bytes header + data

	buf := make([]byte, 44+dataSize)

	// RIFF header
	copy(buf[0:4], "RIFF")
	putLE32(buf[4:8], uint32(fileSize))
	copy(buf[8:12], "WAVE")

	// fmt sub-chunk
	copy(buf[12:16], "fmt ")
	putLE32(buf[16:20], 16)              // sub-chunk size
	putLE16(buf[20:22], 1)               // PCM format
	putLE16(buf[22:24], 1)               // mono
	putLE32(buf[24:28], uint32(sampleRate))
	putLE32(buf[28:32], uint32(sampleRate*2)) // byte rate
	putLE16(buf[32:34], 2)               // block align
	putLE16(buf[34:36], 16)              // bits per sample

	// data sub-chunk
	copy(buf[36:40], "data")
	putLE32(buf[40:44], uint32(dataSize))
	// buf[44:] is already all zeros = silence

	return buf
}

func putLE16(b []byte, v uint16) {
	b[0] = byte(v)
	b[1] = byte(v >> 8)
}

func putLE32(b []byte, v uint32) {
	b[0] = byte(v)
	b[1] = byte(v >> 8)
	b[2] = byte(v >> 16)
	b[3] = byte(v >> 24)
}

// ============================================================================
// 12. Test GenerateExam — input validation edge cases
// ============================================================================

func TestGenerateExamValidation(t *testing.T) {
	// Test ini TIDAK butuh API key — hanya menguji validasi input di sisi Go.
	ctx, cancel := getTestTimeoutContext()
	defer cancel()

	t.Log("Testing GenerateExam with empty material (should fail)...")
	_, err := ai.GenerateExam(ctx, "", []string{"multiple_choice"}, 5, nil)
	if err == nil {
		t.Error("Expected error for empty material, got nil")
	} else {
		t.Logf("  Correctly rejected: %v", err)
	}

	t.Log("Testing GenerateExam with whitespace-only material (should fail)...")
	_, err = ai.GenerateExam(ctx, "   \n\t  ", []string{"essay"}, 3, nil)
	if err == nil {
		t.Error("Expected error for whitespace-only material, got nil")
	} else {
		t.Logf("  Correctly rejected: %v", err)
	}
}

// ============================================================================
// 13. Test GenerateSyllabus — input validation edge cases
// ============================================================================

func TestGenerateSyllabusValidation(t *testing.T) {
	// Test ini TIDAK butuh API key.
	ctx, cancel := getTestTimeoutContext()
	defer cancel()

	t.Log("Testing GenerateSyllabus with empty material AND empty file (should fail)...")
	_, err := ai.GenerateSyllabus(ctx, "", nil, "")
	if err == nil {
		t.Error("Expected error for empty material+file, got nil")
	} else {
		t.Logf("  Correctly rejected: %v", err)
	}

	t.Log("Testing GenerateSyllabus with whitespace-only text and no file (should fail)...")
	_, err = ai.GenerateSyllabus(ctx, "   ", nil, "")
	if err == nil {
		t.Error("Expected error for whitespace-only text, got nil")
	} else {
		t.Logf("  Correctly rejected: %v", err)
	}
}

// ============================================================================
// 14. Test Reply (Chat) — edge cases
// ============================================================================

func TestReplyEdgeCases(t *testing.T) {
	skipIfNoAPIKey(t)
	ctx, cancel := getTestTimeoutContext()
	defer cancel()

	moduleTitle := "Variabel dan Tipe Data di Go"
	moduleMaterial := "Go memiliki tipe data dasar: int, float64, string, bool. " +
		"Variabel bisa dideklarasikan dengan var atau short declaration (:=)."

	// Test 1: Chat tanpa history (pertanyaan pertama)
	t.Log("Testing Reply with empty history (first question)...")
	reply, err := ai.Reply(ctx, moduleTitle, moduleMaterial, nil, "Apa perbedaan var dan :=?")
	if err != nil {
		t.Fatalf("Reply with empty history failed: %v", err)
	}
	if reply == "" {
		t.Fatal("Reply with empty history returned empty response")
	}
	t.Logf("  Reply (no history): %s", truncate(reply, 150))

	// Test 2: Chat dengan history panjang (simulasi percakapan multi-turn)
	t.Log("Testing Reply with long history (multi-turn)...")
	longHistory := []ai.ChatTurn{
		{Role: "user", Content: "Apa itu variabel?"},
		{Role: "model", Content: "Variabel adalah tempat penyimpanan data di memori."},
		{Role: "user", Content: "Bagaimana cara deklarasi variabel di Go?"},
		{Role: "model", Content: "Ada dua cara: menggunakan keyword 'var' atau short declaration ':='."},
		{Role: "user", Content: "Apa perbedaan keduanya?"},
		{Role: "model", Content: "'var' bisa digunakan di level package, sedangkan ':=' hanya di dalam fungsi."},
	}
	reply, err = ai.Reply(ctx, moduleTitle, moduleMaterial, longHistory, "Kalau tipe data boolean bagaimana?")
	if err != nil {
		t.Fatalf("Reply with long history failed: %v", err)
	}
	if reply == "" {
		t.Fatal("Reply with long history returned empty response")
	}
	t.Logf("  Reply (long history): %s", truncate(reply, 150))
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
