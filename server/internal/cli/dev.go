package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// devOptions 封装开发模式命令的显式配置选项。
type devOptions struct {
	migrationDir string
}

// devMigrateRunner 保留开发模式下的迁移执行边界，便于测试替换。
//
// 参数：
//   - cmd: cobra 命令实例
//   - migrationDir: 迁移文件所在的目录路径
//
// 返回值：
//   - error: 如果迁移执行失败则返回错误，否则返回 nil
var devMigrateRunner = func(cmd *cobra.Command, migrationDir string) error {
	return runMigrateUp(cmd, migrateUpOptions{migrationDir: migrationDir})
}

// devServeRunner 保留开发模式下的服务启动边界，直接复用 `runServe`。
var devServeRunner = runServe

// newDevCommand 创建本地开发显式编排命令。
//
// 该命令会先执行数据库迁移，再在迁移成功后启动开发服务器。
//
// 返回值：
//   - *cobra.Command: 配置好的开发模式命令实例
func newDevCommand() *cobra.Command {
	var opts devOptions

	command := &cobra.Command{
		Use:   "dev",
		Short: "Run migrations and start the Graft server for local development",
		Long: "graft dev is an explicit local development orchestration command. " +
			"It runs the migration CLI first and starts the server only after migrations succeed.",
		Example:      "  graft dev",
		SilenceUsage: true,
		Args:         cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDev(cmd, args, opts)
		},
	}

	command.Flags().StringVar(&opts.migrationDir, "dir", defaultMigrationDir, "migration directory")
	return command
}

// runDev 按显式顺序执行开发模式的迁移与启动流程。
//
// 参数：
//   - cmd: cobra 命令实例
//   - args: 命令行参数列表
//   - opts: 开发模式的配置选项
//
// 返回值：
//   - error: 如果迁移或服务器启动失败则返回相应的错误，成功则返回 nil
func runDev(cmd *cobra.Command, args []string, opts devOptions) error {
	if err := devMigrateRunner(cmd, opts.migrationDir); err != nil {
		return fmt.Errorf("run development migrations: %w", err)
	}

	if err := devServeRunner(cmd, args); err != nil {
		return fmt.Errorf("start development server: %w", err)
	}

	return nil
}
