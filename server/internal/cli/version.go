package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"graft/server/internal/buildinfo"
)

var versionInfoProvider = buildinfo.Current

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

func runVersion(cmd *cobra.Command, _ []string) error {
	return writeVersionOutput(cmd.OutOrStdout(), versionInfoProvider())
}

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
