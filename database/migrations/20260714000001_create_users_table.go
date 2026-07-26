package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"goravel/app/facades"
)

type M20260714000001CreateUsersTable struct{}

// Signature The unique signature for the migration.
func (r *M20260714000001CreateUsersTable) Signature() string {
	return "20260714000001_create_users_table"
}

// Up Run the migrations.
func (r *M20260714000001CreateUsersTable) Up() error {
	if facades.Schema().HasTable("users") {
		return nil
	}

	return facades.Schema().Create("users", func(table schema.Blueprint) {
		table.ID()
		table.String("name")
		table.String("email")
		table.String("password")
		table.DateTimeTz("created_at").UseCurrent()
		table.DateTimeTz("updated_at").UseCurrent().UseCurrentOnUpdate()
		table.Unique("email")
	})
}

// Down Reverse the migrations.
func (r *M20260714000001CreateUsersTable) Down() error {
	return facades.Schema().DropIfExists("users")
}
