package seeders

import (
	"goravel/app/facades"
	"goravel/app/models"
)

// UserSeeder creates default user accounts.
// Accounts:
//   - Admin: admin@studybuddy.id / password123
//   - User:  user@studybuddy.id  / password123
type UserSeeder struct{}

func (s *UserSeeder) Signature() string {
	return "UserSeeder"
}

func (s *UserSeeder) Run() error {
	users := []struct {
		Name     string
		Email    string
		Password string
		Role     string
	}{
		{"Admin StudyBuddy", "admin@studybuddy.id", "password123", "admin"},
		{"Faisal", "faisal@studybuddy.id", "password123", "user"},
		{"Nadia", "nadia@studybuddy.id", "password123", "user"},
		{"Bima", "bima@studybuddy.id", "password123", "user"},
		{"Rizky", "rizky@studybuddy.id", "password123", "user"},
		{"Demo User", "user@studybuddy.id", "password123", "user"},
	}

	for _, u := range users {
		// Skip if already exists
		var existing models.User
		facades.Orm().Query().Where("email", u.Email).First(&existing)
		if existing.ID != 0 {
			continue
		}

		hashed, err := facades.Hash().Make(u.Password)
		if err != nil {
			return err
		}

		user := &models.User{
			Name:     u.Name,
			Email:    u.Email,
			Password: hashed,
			Role:     u.Role,
		}

		if err := facades.Orm().Query().Create(user); err != nil {
			return err
		}
	}

	return nil
}
