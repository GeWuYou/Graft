package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// devOptions 封装了开发模式命令的配置选项
type devOptions struct {
	migrationDir string
}

// devMigrateRunner 是开发模式下的数据库迁移执行函数
// 它调用 runMigrateUp 函数来执行数据库迁移操作
// 参数:
//   - cmd: cobra 命令实例
//   - migrationDir: 迁移文件所在的目录路径
//
// 返回值:
//   - error: 如果迁移执行失败则返回错误，否则返回 nil
var devMigrateRunner = func(cmd *cobra.Command, migrationDir string) error {
	return runMigrateUp(cmd, migrateUpOptions{migrationDir: migrationDir})
}

// devServeRunner 是开发模式下的服务器启动函数
// 直接引用 runServe 函数来启动开发服务器
var devServeRunner = runServe

// newDevCommand 创建并返回开发模式的 cobra 命令实例
// 该命令会先执行数据库迁移，然后在迁移成功后启动开发服务器
// 这是一个用于本地开发的显式编排命令
// 返回值:
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

// runDev 执行开发模式的主要逻辑
// 按照顺序先执行数据库迁移，然后启动开发服务器
// 参数:
//   - cmd: cobra 命令实例
//   - args: 命令行参数列表
//   - opts: 开发模式的配置选项
//
// 返回值:
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
