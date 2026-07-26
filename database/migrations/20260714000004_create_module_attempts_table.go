package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"goravel/app/facades"
)

type M20260714000004CreateModuleAttemptsTable struct{}

// Signature The unique signature for the migration.
func (r *M20260714000004CreateModuleAttemptsTable) Signature() string {
	return "20260714000004_create_module_attempts_table"
}

// Up Run the migrations.
func (r *M20260714000004CreateModuleAttemptsTable) Up() error {
	if facades.Schema().HasTable("module_attempts") {
		return nil
	}

	return facades.Schema().Create("module_attempts", func(table schema.Blueprint) {
		table.ID()
		table.UnsignedBigInteger("module_id")
		table.LongText("user_explanation").Nullable()
		table.Integer("feynman_score").Nullable()
		// feedback: {"pujian":"...","kekurangan":"...","saran":"..."} disimpan sebagai JSON string
		table.LongText("feedback").Nullable()
		table.DateTimeTz("created_at").UseCurrent()
		table.DateTimeTz("updated_at").UseCurrent().UseCurrentOnUpdate()
		table.Index("module_id")
		table.Foreign("module_id").References("id").On("modules").CascadeOnDelete()
	})
}

// Down Reverse the migrations.
func (r *M20260714000004CreateModuleAttemptsTable) Down() error {
	return facades.Schema().DropIfExists("module_attempts")
}
