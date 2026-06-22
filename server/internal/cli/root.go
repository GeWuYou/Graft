package cli

import "github.com/spf13/cobra"

// NewRootCommand 返回 `graft` 根命令。
//
// 约束：
//   - 根命令不接受位置参数。
//   - 不带子命令执行时只输出帮助信息。
//   - `version`、`serve`、`migrate`、`dev` 与 `validate` 子命令始终注册到根命令下。
//
// 使用边界：
//   - `graft version` 只读取 build identity，不触发运行时启动、迁移或外部依赖连接。
//   - 普通运行时启动必须保持在 `graft serve` 下显式触发。
//   - 本地开发编排通过 `graft dev` 组合显式迁移与启动流程，并通过 `graft dev air` 提供热重载入口。
//   - 后端完成态质量链通过 `graft validate backend` 显式触发。
//   - OpenAPI 契约校验通过 `graft validate openapi` 或 `graft validate backend --stage openapi` 显式触发。
//   - release-grade 二进制契约校验通过 `graft validate release` 显式触发。
// NewRootCommand 构造并返回 graft CLI 的根命令。该命令注册 version、dev、serve、migrate 和 validate 子命令。如果不提供子命令，则仅打印帮助信息。
func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:          "graft",
		Short:        "Graft server runtime and maintenance commands",
		Long:         "Graft uses explicit subcommands for build identity inspection, database migration, local development orchestration, backend quality validation, runtime smoke validation, and server startup. Running `graft` without a subcommand only prints help.",
		Example:      "  graft version\n  graft dev\n  graft dev air\n  graft migrate up\n  graft validate openapi\n  graft validate release\n  graft validate backend\n  graft validate smoke\n  graft serve",
		SilenceUsage: true,
		Args:         cobra.NoArgs,
		// 保持 `serve` 作为纯运行时入口，这样 `dev` 可以复用显式迁移步骤，
		// 同时根命令仍然只是所有 server 操作的可发现入口。
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	root.AddCommand(newVersionCommand())
	root.AddCommand(newDevCommand())
	root.AddCommand(newServeCommand())
	root.AddCommand(newMigrateCommand())
	root.AddCommand(newValidateCommand())
	return root
}
