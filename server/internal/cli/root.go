package cli

import "github.com/spf13/cobra"

// NewRootCommand assembles the server command tree.
func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:          "graft",
		Short:        "Graft server runtime and maintenance commands",
		SilenceUsage: true,
		Args:         cobra.NoArgs,
		// Keep runtime startup explicit under `graft serve` so the root command
		// can safely act as a discoverable entrypoint for all server operations.
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	root.AddCommand(newServeCommand())
	root.AddCommand(newMigrateCommand())
	return root
}
