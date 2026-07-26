package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

type M20260724064237CreatePomodoroSessionsTable struct {
}

// Signature The unique signature for the migration.
func (r *M20260724064237CreatePomodoroSessionsTable) Signature() string {
	return "20260724064237_create_pomodoro_sessions_table"
}

// Up Run the migrations.
func (r *M20260724064237CreatePomodoroSessionsTable) Up() error {
	return facades.Schema().Create("pomodoro_sessions", func(table schema.Blueprint) {
		table.ID("id")
		table.UnsignedBigInteger("user_id")
		table.String("phase", 50)
		table.Integer("duration")
		table.Timestamps()
		table.SoftDeletes()

		table.Foreign("user_id").References("id").On("users")
	})
}

// Down Reverse the migrations.
func (r *M20260724064237CreatePomodoroSessionsTable) Down() error {
	return facades.Schema().DropIfExists("pomodoro_sessions")
}
