package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"graft/server/internal/config"
)

func validateStartupConfiguration(cmd *cobra.Command) error {
	report, err := config.ResolveAndValidate(config.ResolveOptions{})
	if writeErr := config.WriteReport(cmd.OutOrStdout(), report, "text"); writeErr != nil {
		return fmt.Errorf("write configuration validation report: %w", writeErr)
	}
	return err
}
