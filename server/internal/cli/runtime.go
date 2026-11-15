package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"graft/server/internal/config"
	"graft/server/internal/database"
	taskstore "graft/server/modules/task/store"
)

var runtimeModePreflight = readRuntimeModePreflight

// newRuntimeCommand 提供只读执行模式切换预检，实际绑定由 server 持有运行时独占权后推进。
func newRuntimeCommand() *cobra.Command {
	command := &cobra.Command{Use: "runtime", Short: "Inspect runtime execution ownership"}
	modeCommand := &cobra.Command{Use: "mode", Short: "Inspect an execution mode change"}
	var mode, format string
	preflight := &cobra.Command{Use: "preflight", Args: cobra.NoArgs, Short: "Check whether unsettled work permits changing execution mode", RunE: func(cmd *cobra.Command, _ []string) error {
		if mode != config.ExecutionModeLocal && mode != config.ExecutionModeAgent {
			return errors.New("--mode must be local or agent")
		}
		if format != "json" && format != "text" {
			return errors.New("--format must be json or text")
		}
		report, err := runtimeModePreflight(cmd.Context(), mode)
		if err != nil {
			return err
		}
		if format == "json" {
			if err := json.NewEncoder(cmd.OutOrStdout()).Encode(report); err != nil {
				return err
			}
		} else {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "execution mode %s -> %s ready=%t tasks=%d submissions=%d leases=%d (%s)\n", report.CurrentMode, report.RequestedMode, report.Ready, report.BlockingTasks, report.BlockingSubmissions, report.BlockingLeases, report.DiagnosticCode)
		}
		if !report.Ready {
			return errors.New("execution mode switch is blocked by unsettled work")
		}
		return nil
	}}
	preflight.Flags().StringVar(&mode, "mode", "", "requested mode: local or agent")
	preflight.Flags().StringVar(&format, "format", "text", "output format: text or json")
	_ = preflight.MarkFlagRequired("mode")
	modeCommand.AddCommand(preflight)
	command.AddCommand(modeCommand)
	return command
}

func readRuntimeModePreflight(ctx context.Context, mode string) (taskstore.ExecutionModePreflight, error) {
	cfg, err := config.Load()
	if err != nil {
		return taskstore.ExecutionModePreflight{}, err
	}
	resources, err := database.Open(cfg.Database)
	if err != nil {
		return taskstore.ExecutionModePreflight{}, err
	}
	defer func() { _ = database.Close(resources) }()
	repository, err := taskstore.NewSQLRepository(resources.SQL, taskstore.SQLDialectPostgres)
	if err != nil {
		return taskstore.ExecutionModePreflight{}, err
	}
	return repository.PreflightExecutionMode(ctx, mode)
}
