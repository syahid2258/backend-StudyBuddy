package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"goravel/app/facades"
)

type M20260714000008CreateChatMessagesTable struct{}

// Signature The unique signature for the migration.
func (r *M20260714000008CreateChatMessagesTable) Signature() string {
	return "20260714000008_create_chat_messages_table"
}

// Up Run the migrations.
func (r *M20260714000008CreateChatMessagesTable) Up() error {
	if facades.Schema().HasTable("chat_messages") {
		return nil
	}

	return facades.Schema().Create("chat_messages", func(table schema.Blueprint) {
		table.ID()
		table.UnsignedBigInteger("module_id")
		// role: "user" | "model" — dipakai untuk membangun ulang histori
		// percakapan tiap request (mode stateless, lihat ai/chat.go).
		table.String("role")
		table.LongText("content")
		table.DateTimeTz("created_at").UseCurrent()
		table.DateTimeTz("updated_at").UseCurrent().UseCurrentOnUpdate()
		table.Index("module_id")
		table.Foreign("module_id").References("id").On("modules").CascadeOnDelete()
	})
}

// Down Reverse the migrations.
func (r *M20260714000008CreateChatMessagesTable) Down() error {
	return facades.Schema().DropIfExists("chat_messages")
}
