package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"goravel/app/facades"
)

type M20260720093500AddRoleToUsersTable struct{}

// Signature The unique signature for the migration.
func (r *M20260720093500AddRoleToUsersTable) Signature() string {
	return "20260720093500_add_role_to_users_table"
}

// Up Run the migrations.
func (r *M20260720093500AddRoleToUsersTable) Up() error {
		return facades.Schema().Table("users", func(table schema.Blueprint) {
			table.String("role").Default("user").Comment("user, admin")
		})
}

// Down Reverse the migrations.
func (r *M20260720093500AddRoleToUsersTable) Down() error {
	return nil
}
