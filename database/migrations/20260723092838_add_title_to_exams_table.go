package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"goravel/app/facades"
)

type M20260723092838AddTitleToExamsTable struct{}

// Signature The unique signature for the migration.
func (r *M20260723092838AddTitleToExamsTable) Signature() string {
	return "20260723092838_add_title_to_exams_table"
}

// Up Run the migrations.
func (r *M20260723092838AddTitleToExamsTable) Up() error {
	return facades.Schema().Table("exams", func(table schema.Blueprint) {
		table.String("title").Nullable()
	})
}

// Down Reverse the migrations.
func (r *M20260723092838AddTitleToExamsTable) Down() error {
	return facades.Schema().Table("exams", func(table schema.Blueprint) {
		table.DropColumn("title")
	})
}
