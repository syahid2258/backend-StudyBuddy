package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"goravel/app/facades"
)

type M20260714000006CreateExamsTable struct{}

// Signature The unique signature for the migration.
func (r *M20260714000006CreateExamsTable) Signature() string {
	return "20260714000006_create_exams_table"
}

// Up Run the migrations.
func (r *M20260714000006CreateExamsTable) Up() error {
	if facades.Schema().HasTable("exams") {
		return nil
	}

	return facades.Schema().Create("exams", func(table schema.Blueprint) {
		table.ID()
		table.UnsignedBigInteger("project_id")
		// question_types: ["multiple_choice","essay"] disimpan sebagai JSON string
		table.Text("question_types").Nullable()
		// questions: array soal + kunci jawaban (kunci disembunyikan dari response ke frontend
		// di level controller, bukan di level DB) disimpan sebagai JSON string
		table.LongText("questions").Nullable()
		table.DateTimeTz("created_at").UseCurrent()
		table.DateTimeTz("updated_at").UseCurrent().UseCurrentOnUpdate()
		table.Index("project_id")
		table.Foreign("project_id").References("id").On("projects").CascadeOnDelete()
	})
}

// Down Reverse the migrations.
func (r *M20260714000006CreateExamsTable) Down() error {
	return facades.Schema().DropIfExists("exams")
}
