// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"
)

var devStopAirSignal = signalDevPID

func newDevStopAirCommand() *cobra.Command {
	opts := devStopAirOptions{configPath: ".air.toml"}

	command := &cobra.Command{
		Use:   "stop-air",
		Short: "Stop the development supervisor and Air process for this repository",
		Long: "graft dev stop-air stops the development supervisor and any tracked Air / serve child " +
			"processes using PID files under `server/tmp`.",
		Example:      "  graft dev stop-air\n  graft dev stop-air --config .air.toml",
		SilenceUsage: true,
		Args:         cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDevStopAir(cmd, opts)
		},
	}

	command.Flags().StringVar(&opts.configPath, "config", opts.configPath, "Air config file path")
	return command
}

func runDevStopAir(cmd *cobra.Command, _ devStopAirOptions) error {
	pidPaths, err := resolveDevPIDPaths()
	if err != nil {
		return fmt.Errorf("resolve dev pid paths: %w", err)
	}

	supervisorCount, err := stopDevPIDFile(pidPaths.supervisor, syscall.SIGTERM, "supervisor")
	if err != nil {
		return err
	}
	airCount, err := stopDevPIDFile(pidPaths.air, syscall.SIGTERM, "Air")
	if err != nil {
		return err
	}
	serveCount, err := stopDevPIDFile(pidPaths.serve, syscall.SIGTERM, "serve")
	if err != nil {
		return err
	}
	removeDevPIDFile(pidPaths.notify)

	if supervisorCount == 0 && airCount == 0 && serveCount == 0 {
		return writeDevStopAirResult(cmd.OutOrStdout(), "no development process found under %s\n", filepath.Dir(pidPaths.supervisor))
	}

	return writeDevStopAirResult(
		cmd.OutOrStdout(),
		"stopped development processes under %s: supervisor=%d air=%d serve=%d\n",
		filepath.Dir(pidPaths.supervisor),
		supervisorCount,
		airCount,
		serveCount,
	)
}

func stopDevPIDFile(path string, sig syscall.Signal, label string) (int, error) {
	pid, err := readDevPIDFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		removeDevPIDFile(path)
		return 0, fmt.Errorf("read %s pid file %s: %w", label, path, err)
	}

	if err := devStopAirSignal(pid, sig); err != nil {
		return 0, fmt.Errorf("stop %s process %d: %w", label, pid, err)
	}

	removeDevPIDFile(path)
	return 1, nil
}

func writeDevStopAirResult(writer io.Writer, format string, args ...any) error {
	if _, err := io.WriteString(writer, fmt.Sprintf(format, args...)); err != nil {
		return fmt.Errorf("write stop-air result: %w", err)
	}
	return nil
}
