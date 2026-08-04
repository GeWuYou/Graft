package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"graft/server/internal/config"
	"graft/server/internal/database"
	"graft/server/modules/backup"
	backupstore "graft/server/modules/backup/store"
	"graft/server/modules/task"
	taskstore "graft/server/modules/task/store"
	platformupdate "graft/server/modules/update"
)

var updateCutover = runUpdateCutover

// newUpdateCommand 创建 Update 维护命令树。切换命令不启动 HTTP server，也不执行迁移。
func newUpdateCommand() *cobra.Command {
	command := &cobra.Command{Use: "update", Short: "Run platform update maintenance commands"}
	command.AddCommand(&cobra.Command{
		Use:          "cutover-v1",
		Short:        "Cut over legacy schema-v1 update state before migration",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return updateCutover(cmd.Context())
		},
	})
	return command
}

func runUpdateCutover(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config for update cutover: %w", err)
	}
	resources, err := database.Open(cfg.Database)
	if err != nil {
		return fmt.Errorf("open database for update cutover: %w", err)
	}
	defer func() { _ = database.Close(resources) }()
	taskRepository, err := taskstore.NewSQLRepository(resources.SQL, taskstore.SQLDialectPostgres)
	if err != nil {
		return fmt.Errorf("build task service for update cutover: %w", err)
	}
	backupRepository, err := backupstore.NewSQLRepository(resources.SQL)
	if err != nil {
		return fmt.Errorf("build backup service for update cutover: %w", err)
	}
	return platformupdate.CutoverV1(ctx, platformupdate.RunnerStateRoot, resources.SQL, task.NewRuntime(taskRepository), backup.NewService(backupRepository))
}
