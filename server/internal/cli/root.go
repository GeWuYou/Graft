package cli

import "github.com/spf13/cobra"

// NewRootCommand returns the root `graft` command.
//
// Contract:
//   - The root command accepts no positional arguments.
//   - Executing the root command without a subcommand prints help output.
//   - The `serve` and `migrate` subcommands are always registered.
//
// Usage constraints:
//   - Runtime startup must stay explicit under `graft serve`.
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
