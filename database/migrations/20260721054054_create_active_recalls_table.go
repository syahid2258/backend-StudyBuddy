package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"goravel/app/facades"
)

type M20260721054054CreateActiveRecallsTable struct{}

// Signature The unique signature for the migration.
func (r *M20260721054054CreateActiveRecallsTable) Signature() string {
	return "20260721054054_create_active_recalls_table"
}

// Up Run the migrations.
func (r *M20260721054054CreateActiveRecallsTable) Up() error {
	if !facades.Schema().HasTable("active_recalls") {
		return facades.Schema().Create("active_recalls", func(table schema.Blueprint) {
			table.ID()
			table.UnsignedBigInteger("module_id")
			table.Text("question")
			table.Text("answer").Nullable()
			table.Integer("score").Nullable()
			table.Text("feedback").Nullable()
			table.Json("evaluations").Nullable()
			table.TimestampsTz()
			
			table.Foreign("module_id").References("id").On("modules").CascadeOnDelete()
		})
	}

	return nil
}

// Down Reverse the migrations.
func (r *M20260721054054CreateActiveRecallsTable) Down() error {
	return facades.Schema().DropIfExists("active_recalls")
}
