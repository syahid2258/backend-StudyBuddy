package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"goravel/app/facades"
)

type M20260714000003CreateModulesTable struct{}

// Signature The unique signature for the migration.
func (r *M20260714000003CreateModulesTable) Signature() string {
	return "20260714000003_create_modules_table"
}

// Up Run the migrations.
func (r *M20260714000003CreateModulesTable) Up() error {
	if facades.Schema().HasTable("modules") {
		return nil
	}

	return facades.Schema().Create("modules", func(table schema.Blueprint) {
		table.ID()
		table.UnsignedBigInteger("project_id")
		table.String("title")
		table.Integer("order").Default(0)
		table.Boolean("is_locked").Default(true)
		// status: not_started | in_progress | mastered
		table.String("status").Default("locked")
		// Kolom JSON native, dipasangkan dengan custom type ContentBlockList
		// (Value()/Scan()) di model. Nullable karena baru terisi saat modul
		// pertama kali dibuka (generate on-demand, lihat Fase 3).
		table.Json("content_blocks").Nullable()
		table.DateTimeTz("created_at").UseCurrent()
		table.DateTimeTz("updated_at").UseCurrent().UseCurrentOnUpdate()
		table.Index("project_id")
		// Kalau project dihapus, semua modul di dalamnya ikut terhapus (tidak jadi orphan data).
		table.Foreign("project_id").References("id").On("projects").CascadeOnDelete()
	})
}

// Down Reverse the migrations.
func (r *M20260714000003CreateModulesTable) Down() error {
	return facades.Schema().DropIfExists("modules")
}
