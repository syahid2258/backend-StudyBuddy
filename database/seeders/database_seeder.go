package seeders

import (
	"github.com/goravel/framework/contracts/database/seeder"

	"goravel/app/facades"
)

// DatabaseSeeder is the root seeder that calls all other seeders in order.
type DatabaseSeeder struct{}

func (s *DatabaseSeeder) Signature() string {
	return "DatabaseSeeder"
}

func (s *DatabaseSeeder) Run() error {
	return facades.Seeder().Call([]seeder.Seeder{
		&UserSeeder{},
		&ProjectSeeder{},
		&AdminUserSeeder{},
	})
}
