package seeders

import (
	"time"

	"goravel/app/facades"
	"goravel/app/models"
)

// ProjectSeeder creates demo projects for all user accounts down to modules and flashcards.
type ProjectSeeder struct{}

func (s *ProjectSeeder) Signature() string {
	return "ProjectSeeder"
}

func (s *ProjectSeeder) Run() error {
	var users []models.User
	if err := facades.Orm().Query().Get(&users); err != nil {
		return err
	}

	for _, user := range users {
		if user.Role == "admin" {
			continue // Admin expects user administration, no topics out of the box
		}

		var count int64
		count, _ = facades.Orm().Query().Model(&models.Project{}).Where("user_id", user.ID).Count()
		if count > 0 {
			continue
		}

		projects := []models.Project{
			{
				UserID:        &user.ID,
				Title:         "Pemrograman Web Modern",
				Completed:     2,
				Total:         5,
				SourceFileURL: "/storage/dummy/web-modern.pdf",
				Methods:       &models.ProjectMethods{Feynman: true, SpacedRepetition: true, Pomodoro: true},
			},
			{
				UserID:        &user.ID,
				Title:         "Kecerdasan Buatan Dasar",
				Completed:     0,
				Total:         3,
				SourceFileURL: "/storage/dummy/ai-dasar.pdf",
			},
			{
				UserID:    &user.ID,
				Title:     "Algoritma dan Struktur Data",
				Completed: 1,
				Total:     4,
				Methods:   &models.ProjectMethods{Feynman: true, SpacedRepetition: false, Pomodoro: false},
			},
		}

		for i := range projects {
			if err := facades.Orm().Query().Create(&projects[i]); err != nil {
				return err
			}

			// 1. Create modules
			modules := []models.Module{
				{ProjectID: projects[i].ID, Title: "Pengenalan dan Dasar", Order: 1, IsLocked: false, Status: "mastered"},
				{ProjectID: projects[i].ID, Title: "Konsep Inti", Order: 2, IsLocked: false, Status: "in_progress"},
				{ProjectID: projects[i].ID, Title: "Studi Kasus Ekstensif", Order: 3, IsLocked: true, Status: "locked"},
			}

			if projects[i].Total >= 4 {
				modules = append(modules, models.Module{ProjectID: projects[i].ID, Title: "Implementasi Sistem", Order: 4, IsLocked: true, Status: "locked"})
			}
			if projects[i].Total >= 5 {
				modules = append(modules, models.Module{ProjectID: projects[i].ID, Title: "Evaluasi Akhir", Order: 5, IsLocked: true, Status: "locked"})
			}

			for j := range modules {
				// Populate ContentBlocks for mastered/opened modules to simulate generative AI result check
				if modules[j].Status == "mastered" || modules[j].Status == "in_progress" {
					modules[j].ContentBlocks = &models.ContentBlockList{
						{Type: "paragraph", Title: "Ringkasan Utama", Text: "Konsep utama dari " + modules[j].Title + " membahas fondasi dasar yang membentuk pilar sistem."},
						{Type: "jembatan_keledai", Title: "Cara Mudah Ingat", Text: "Ingat saja K-U-A-T (Kuatkan, Utama, Analisis, Terapkan)."},
					}
				}
				facades.Orm().Query().Create(&modules[j])
			}

			// 2. Create Flashcards
			if projects[i].Methods != nil && projects[i].Methods.SpacedRepetition {
				now := time.Now()
				flashcards := []models.Flashcard{
					{ProjectID: projects[i].ID, FrontText: "Jelaskan definisi dari " + projects[i].Title + "!", BackText: "Sebuah cabang keilmuan yang memfokuskan pada integrasi sistem.", EaseFactor: 2.5, IntervalDays: 1, NextReviewDate: &now},
					{ProjectID: projects[i].ID, FrontText: "Sebutkan 3 komponen utama!", BackText: "1. Data 2. Analisis 3. Infrastruktur", EaseFactor: 2.1, IntervalDays: 3, NextReviewDate: &now},
					{ProjectID: projects[i].ID, FrontText: "Apa kepanjangan K-U-A-T?", BackText: "Kuatkan, Utama, Analisis, Terapkan", EaseFactor: 2.3, IntervalDays: 2, NextReviewDate: &now},
				}
				for k := range flashcards {
					facades.Orm().Query().Create(&flashcards[k])
				}
			}
		}
	}

	return nil
}
