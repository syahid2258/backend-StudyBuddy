package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

type M20260727000001CreateApiKeysTable struct {
}

// Signature The unique signature for the migration.
func (r *M20260727000001CreateApiKeysTable) Signature() string {
	return "20260727000001_create_api_keys_table"
}

// Up Run the migrations.
func (r *M20260727000001CreateApiKeysTable) Up() error {
	return facades.Schema().Create("api_keys", func(table schema.Blueprint) {
		table.ID()
		table.String("name")
		table.String("key")
		table.Boolean("is_active").Default(true)
		table.Boolean("is_valid").Default(true)
		table.DateTimeTz("created_at").UseCurrent()
		table.DateTimeTz("updated_at").UseCurrent().UseCurrentOnUpdate()
		table.Unique("key")
	})
}

// Down Reverse the migrations.
func (r *M20260727000001CreateApiKeysTable) Down() error {
	return facades.Schema().DropIfExists("api_keys")
}
