package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

type M20260727000002AddTotalUsageToApiKeys struct{}

// Signature The unique signature for the migration.
func (r *M20260727000002AddTotalUsageToApiKeys) Signature() string {
	return "20260727000002_add_total_usage_to_api_keys"
}

// Up Run the migrations.
func (r *M20260727000002AddTotalUsageToApiKeys) Up() error {
	return facades.Schema().Table("api_keys", func(table schema.Blueprint) {
		table.Integer("total_requests").Default(0)
		table.Integer("total_tokens").Default(0)
	})
}

// Down Reverse the migrations.
func (r *M20260727000002AddTotalUsageToApiKeys) Down() error {
	return facades.Schema().Table("api_keys", func(table schema.Blueprint) {
		table.DropColumn("total_requests")
		table.DropColumn("total_tokens")
	})
}
