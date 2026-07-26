package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"goravel/app/facades"
)

type M20260714000007CreateExamAttemptsTable struct{}

// Signature The unique signature for the migration.
func (r *M20260714000007CreateExamAttemptsTable) Signature() string {
	return "20260714000007_create_exam_attempts_table"
}

// Up Run the migrations.
func (r *M20260714000007CreateExamAttemptsTable) Up() error {
	if facades.Schema().HasTable("exam_attempts") {
		return nil
	}

	return facades.Schema().Create("exam_attempts", func(table schema.Blueprint) {
		table.ID()
		table.UnsignedBigInteger("exam_id")
		// answers: [{"question_id":"Q1","answer":"..."}] disimpan sebagai JSON string
		table.LongText("answers").Nullable()
		table.Integer("final_score").Nullable()
		table.Text("analysis").Nullable()
		table.DateTimeTz("created_at").UseCurrent()
		table.DateTimeTz("updated_at").UseCurrent().UseCurrentOnUpdate()
		table.Index("exam_id")
		table.Foreign("exam_id").References("id").On("exams").CascadeOnDelete()
	})
}

// Down Reverse the migrations.
func (r *M20260714000007CreateExamAttemptsTable) Down() error {
	return facades.Schema().DropIfExists("exam_attempts")
}
