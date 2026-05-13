package cli

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"graft/server/internal/app"
	"graft/server/plugins/user"
)

func newServeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the Graft HTTP server",
		RunE:  runServe,
	}
}

func runServe(cmd *cobra.Command, args []string) error {
	runtime, err := app.NewRuntime(user.NewPlugin())
	if err != nil {
		return fmt.Errorf("create runtime: %w", err)
	}

	runCtx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if runCtx == nil {
		runCtx = context.Background()
	}

	if err := runtime.Run(runCtx); err != nil {
		return fmt.Errorf("run runtime: %w", err)
	}

	return nil
}
