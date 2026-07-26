package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"goravel/app/facades"
)

type M20260715000001AddForeignKeys struct{}

func (r *M20260715000001AddForeignKeys) Signature() string {
	return "20260715000001_add_foreign_keys"
}

// Up adds foreign key constraints and soft delete columns where missing.
func (r *M20260715000001AddForeignKeys) Up() error {
	// Add soft delete to projects if not present
	if facades.Schema().HasTable("projects") && !facades.Schema().HasColumn("projects", "deleted_at") {
		if err := facades.Schema().Table("projects", func(table schema.Blueprint) {
			table.SoftDeletesTz()
		}); err != nil {
			return err
		}
	}

	// Add soft delete to modules if not present
	if facades.Schema().HasTable("modules") && !facades.Schema().HasColumn("modules", "deleted_at") {
		if err := facades.Schema().Table("modules", func(table schema.Blueprint) {
			table.SoftDeletesTz()
		}); err != nil {
			return err
		}
	}

	// Add foreign key: projects.user_id → users.id
	if facades.Schema().HasTable("projects") && facades.Schema().HasTable("users") {
		if facades.Schema().HasColumn("projects", "user_id") {
			_ = facades.Schema().Table("projects", func(table schema.Blueprint) {
				table.Foreign("user_id").References("id").On("users")
			})
		}
	}

	// Add foreign key: modules.project_id → projects.id
	if facades.Schema().HasTable("modules") && facades.Schema().HasTable("projects") {
		if facades.Schema().HasColumn("modules", "project_id") {
			_ = facades.Schema().Table("modules", func(table schema.Blueprint) {
				table.Foreign("project_id").References("id").On("projects")
			})
		}
	}

	// Add foreign key: flashcards.project_id → projects.id
	if facades.Schema().HasTable("flashcards") && facades.Schema().HasTable("projects") {
		if facades.Schema().HasColumn("flashcards", "project_id") {
			_ = facades.Schema().Table("flashcards", func(table schema.Blueprint) {
				table.Foreign("project_id").References("id").On("projects")
			})
		}
	}

	// Add foreign key: module_attempts.module_id → modules.id
	if facades.Schema().HasTable("module_attempts") && facades.Schema().HasTable("modules") {
		if facades.Schema().HasColumn("module_attempts", "module_id") {
			_ = facades.Schema().Table("module_attempts", func(table schema.Blueprint) {
				table.Foreign("module_id").References("id").On("modules")
			})
		}
	}

	// Add foreign key: exams.project_id → projects.id
	if facades.Schema().HasTable("exams") && facades.Schema().HasTable("projects") {
		if facades.Schema().HasColumn("exams", "project_id") {
			_ = facades.Schema().Table("exams", func(table schema.Blueprint) {
				table.Foreign("project_id").References("id").On("projects")
			})
		}
	}

	return nil
}

// Down removes the soft delete columns.
func (r *M20260715000001AddForeignKeys) Down() error {
	if facades.Schema().HasTable("projects") && facades.Schema().HasColumn("projects", "deleted_at") {
		_ = facades.Schema().Table("projects", func(table schema.Blueprint) {
			table.DropColumn("deleted_at")
		})
	}
	if facades.Schema().HasTable("modules") && facades.Schema().HasColumn("modules", "deleted_at") {
		_ = facades.Schema().Table("modules", func(table schema.Blueprint) {
			table.DropColumn("deleted_at")
		})
	}
	return nil
}
