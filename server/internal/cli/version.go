package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"graft/server/internal/buildinfo"
)

var versionInfoProvider = buildinfo.Current

// NewVersionCommand 构造并返回一个 Cobra 命令，用于打印服务器的构建标识。该命令不启动运行时、不连接外部服务，也不执行迁移。
func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the server build identity",
		Long: "graft version prints the canonical server build identity surface without starting the runtime, " +
			"opening external services, or applying migrations.",
		Args: cobra.NoArgs,
		RunE: runVersion,
	}
}

// runVersion 执行 version 命令，获取并输出服务器的构建标识信息。
func runVersion(cmd *cobra.Command, _ []string) error {
	return writeVersionOutput(cmd.OutOrStdout(), versionInfoProvider())
}

// writeVersionOutput 将构建版本、git 提交、构建时间和 git tree 状态写入提供的写入器，并返回任何写入错误。
func writeVersionOutput(out io.Writer, info buildinfo.Info) error {
	_, err := fmt.Fprintf(
		out,
		"version: %s\ngit_commit: %s\nbuild_time_utc: %s\ngit_tree_state: %s\n",
		info.Version,
		info.GitCommit,
		info.BuildTimeUTC,
		info.GitTreeState,
	)
	return err
}
