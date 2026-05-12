package cli

import "github.com/spf13/cobra"

// NewRootCommand assembles the server command tree.
func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:          "graft",
		Short:        "Graft server runtime and maintenance commands",
		SilenceUsage: true,
		RunE:         runServe,
	}

	root.AddCommand(newServeCommand())
	root.AddCommand(newMigrateCommand())
	return root
}
