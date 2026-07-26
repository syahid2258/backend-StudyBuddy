package seeders

import (
	"goravel/app/models"
	"github.com/goravel/framework/facades"
)

type AdminUserSeeder struct {
}

// Signature The name and signature of the seeder.
func (s *AdminUserSeeder) Signature() string {
	return "AdminUserSeeder"
}

// Run executes the seeder logic.
func (s *AdminUserSeeder) Run() error {
	hash := facades.Hash()
	adminPassword, _ := hash.Make("admin123")
	userPassword, _ := hash.Make("user123")

	adminUser := models.User{
		Name:     "Admin AI Wizard",
		Email:    "admin@studybuddy.com",
		Password: adminPassword,
		Role:     "admin",
	}

	user1 := models.User{
		Name:     "Dummy User 1",
		Email:    "user1@test.com",
		Password: userPassword,
		Role:     "user",
	}

	user2 := models.User{
		Name:     "Dummy User 2",
		Email:    "user2@test.com",
		Password: userPassword,
		Role:     "user",
	}

	facades.Orm().Query().Create(&adminUser)
	facades.Orm().Query().Create(&user1)
	facades.Orm().Query().Create(&user2)

	return nil
}
