package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"goravel/app/facades"
)

type M20260720093622CreateAdminRequestsTable struct{}

// Signature The unique signature for the migration.
func (r *M20260720093622CreateAdminRequestsTable) Signature() string {
	return "20260720093622_create_admin_requests_table"
}

// Up Run the migrations.
func (r *M20260720093622CreateAdminRequestsTable) Up() error {
	if !facades.Schema().HasTable("admin_requests") {
		return facades.Schema().Create("admin_requests", func(table schema.Blueprint) {
			table.ID()
			table.UnsignedBigInteger("user_id")
			table.String("type").Comment("chat, module, evaluation")
			table.Text("payload").Nullable().Comment("JSON payload")
			table.String("status").Default("pending").Comment("pending, processing, completed, failed")
			table.UnsignedBigInteger("admin_id").Nullable()
			table.Text("response").Nullable().Comment("JSON response")
			table.TimestampsTz()
		})
	}

	return nil
}

// Down Reverse the migrations.
func (r *M20260720093622CreateAdminRequestsTable) Down() error {
	return facades.Schema().DropIfExists("admin_requests")
}
