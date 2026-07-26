package models

import (
	"github.com/goravel/framework/database/orm"
)

// User merepresentasikan akun pengguna platform.
type User struct {
	orm.Model
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"-"` // tidak pernah diserialisasi ke response JSON
	Role     string `json:"role"`
}
