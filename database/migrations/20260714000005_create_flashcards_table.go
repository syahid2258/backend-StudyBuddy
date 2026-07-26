package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"goravel/app/facades"
)

type M20260714000005CreateFlashcardsTable struct{}

// Signature The unique signature for the migration.
func (r *M20260714000005CreateFlashcardsTable) Signature() string {
	return "20260714000005_create_flashcards_table"
}

// Up Run the migrations.
func (r *M20260714000005CreateFlashcardsTable) Up() error {
	if facades.Schema().HasTable("flashcards") {
		return nil
	}

	return facades.Schema().Create("flashcards", func(table schema.Blueprint) {
		table.ID()
		table.UnsignedBigInteger("project_id")
		table.Text("front_text")
		table.Text("back_text")
		// Parameter algoritma spaced repetition (SM-2 sederhana, lihat Fase 4)
		table.Float("ease_factor").Default(2.5)
		table.Integer("interval_days").Default(1)
		table.DateTimeTz("next_review_date").Nullable()
		table.DateTimeTz("created_at").UseCurrent()
		table.DateTimeTz("updated_at").UseCurrent().UseCurrentOnUpdate()
		table.Index("project_id")
		table.Index("next_review_date")
		table.Foreign("project_id").References("id").On("projects").CascadeOnDelete()
	})
}

// Down Reverse the migrations.
func (r *M20260714000005CreateFlashcardsTable) Down() error {
	return facades.Schema().DropIfExists("flashcards")
}
