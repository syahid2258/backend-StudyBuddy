package services

import (
	"errors"

	"goravel/app/facades"
	"goravel/app/models"
)

// AuthService handles user authentication logic.
type AuthService struct{}

func NewAuthService() *AuthService {
	return &AuthService{}
}

// FindByEmail retrieves a user by email address.
func (s *AuthService) FindByEmail(email string) (*models.User, error) {
	var user models.User
	err := facades.Orm().Query().Where("email", email).First(&user)
	if err != nil {
		return nil, err
	}
	if user.ID == 0 {
		return nil, errors.New("user not found")
	}
	return &user, nil
}

// CreateUser creates a new user with a hashed password.
func (s *AuthService) CreateUser(name, email, password string) (*models.User, error) {
	// Check if email already registered
	var existing models.User
	facades.Orm().Query().Where("email", email).First(&existing)
	if existing.ID != 0 {
		return nil, errors.New("email sudah terdaftar")
	}

	hashed, err := facades.Hash().Make(password)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Name:     name,
		Email:    email,
		Password: hashed,
	}

	if err := facades.Orm().Query().Create(user); err != nil {
		return nil, err
	}

	return user, nil
}
