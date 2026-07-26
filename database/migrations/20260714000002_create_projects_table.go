package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"goravel/app/facades"
)

type M20260714000002CreateProjectsTable struct{}

// Signature The unique signature for the migration.
func (r *M20260714000002CreateProjectsTable) Signature() string {
	return "20260714000002_create_projects_table"
}

// Up Run the migrations.
func (r *M20260714000002CreateProjectsTable) Up() error {
	if facades.Schema().HasTable("projects") {
		return nil
	}

	return facades.Schema().Create("projects", func(table schema.Blueprint) {
		table.ID()
		table.UnsignedBigInteger("user_id").Nullable()
		table.String("title")
		table.String("source_file_url").Nullable()
		// Kolom JSON native, dipasangkan dengan custom type ProjectMethods
		// (Value()/Scan()) di model — Orm otomatis marshal/unmarshal, tidak
		// perlu encode/decode manual di controller.
		table.Json("methods").Nullable()
		// Kolom legacy untuk kompatibilitas dengan frontend saat ini (Home.jsx / AddTopic.jsx)
		// yang masih pakai input manual "completed dari total sub-materi".
		// Nantinya bisa dihitung otomatis dari tabel modules setelah Fase 2-3 AI selesai.
		table.Integer("completed").Default(0)
		table.Integer("total").Default(1)
		table.DateTimeTz("created_at").UseCurrent()
		table.DateTimeTz("updated_at").UseCurrent().UseCurrentOnUpdate()
		table.Index("user_id")
		// NullOnDelete (bukan Cascade) karena user_id nullable: kalau akun user
		// dihapus, project-nya tidak ikut hilang, cukup jadi "tanpa pemilik".
		table.Foreign("user_id").References("id").On("users").NullOnDelete()
	})
}

// Down Reverse the migrations.
func (r *M20260714000002CreateProjectsTable) Down() error {
	return facades.Schema().DropIfExists("projects")
}
