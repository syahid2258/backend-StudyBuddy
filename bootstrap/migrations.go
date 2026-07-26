package bootstrap

import (
	"github.com/goravel/framework/contracts/database/schema"
	"goravel/database/migrations"
)

func Migrations() []schema.Migration {
	return []schema.Migration{
		&migrations.M20210101000001CreateJobsTable{},
		&migrations.M20260714000001CreateUsersTable{},
		&migrations.M20260714000002CreateProjectsTable{},
		&migrations.M20260714000003CreateModulesTable{},
		&migrations.M20260714000004CreateModuleAttemptsTable{},
		&migrations.M20260714000005CreateFlashcardsTable{},
		&migrations.M20260714000006CreateExamsTable{},
		&migrations.M20260714000007CreateExamAttemptsTable{},
		&migrations.M20260714000008CreateChatMessagesTable{},
		&migrations.M20260715000001AddForeignKeys{},
		&migrations.M20260720093500AddRoleToUsersTable{},
		&migrations.M20260720093622CreateAdminRequestsTable{},
		&migrations.M20260720100513AddSubTopicIdToAdminRequestsTable{},
		&migrations.M20260721054054CreateActiveRecallsTable{},
		&migrations.M20260723092838AddTitleToExamsTable{},
		&migrations.M20260724064237CreatePomodoroSessionsTable{},
		&migrations.M20260727000001CreateApiKeysTable{},
		&migrations.M20260727000002AddTotalUsageToApiKeys{},
	}
}
