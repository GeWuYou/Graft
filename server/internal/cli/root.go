package cli

import "github.com/spf13/cobra"

// NewRootCommand 返回 `graft` 根命令。
//
// 约束：
//   - 根命令不接受位置参数。
//   - 不带子命令执行时只输出帮助信息。
//   - `serve`、`migrate` 与 `dev` 子命令始终注册到根命令下。
//
// 使用边界：
//   - 普通运行时启动必须保持在 `graft serve` 下显式触发。
//   - 本地开发编排通过 `graft dev` 组合显式迁移与启动流程。
func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:          "graft",
		Short:        "Graft server runtime and maintenance commands",
		Long:         "Graft uses explicit subcommands for database migration, local development orchestration, and server startup. Running `graft` without a subcommand only prints help.",
		Example:      "  graft dev\n  graft migrate up\n  graft serve",
		SilenceUsage: true,
		Args:         cobra.NoArgs,
		// 保持 `serve` 作为纯运行时入口，这样 `dev` 可以复用显式迁移步骤，
		// 同时根命令仍然只是所有 server 操作的可发现入口。
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	root.AddCommand(newDevCommand())
	root.AddCommand(newServeCommand())
	root.AddCommand(newMigrateCommand())
	return root
}
