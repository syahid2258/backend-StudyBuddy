package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"

	"goravel/app/facades"
)

type M20260720100513AddSubTopicIdToAdminRequestsTable struct{}

// Signature The unique signature for the migration.
func (r *M20260720100513AddSubTopicIdToAdminRequestsTable) Signature() string {
	return "20260720100513_add_sub_topic_id_to_admin_requests_table"
}

// Up Run the migrations.
func (r *M20260720100513AddSubTopicIdToAdminRequestsTable) Up() error {
	return facades.Schema().Table("admin_requests", func(table schema.Blueprint) {
		table.UnsignedBigInteger("sub_topic_id").Nullable().After("user_id")
	})
}

// Down Reverse the migrations.
func (r *M20260720100513AddSubTopicIdToAdminRequestsTable) Down() error {
	return facades.Schema().Table("admin_requests", func(table schema.Blueprint) {
		table.DropColumn("sub_topic_id")
	})
}
