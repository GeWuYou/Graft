// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

const (
	devPIDFilePerm       = 0o600
	devPIDDirPerm        = 0o755
	devStopTimeout       = 5 * time.Second
	devStatePollTimeout  = 200 * time.Millisecond
	devSupervisorPIDName = "dev-supervisor.pid"
	devAirPIDName        = "dev-air.pid"
	devServePIDName      = "dev-serve.pid"
	devNotifyPIDName     = "dev-notify.pid"
)

type devOptions struct {
	migrationDir string
}

type devAirOptions struct {
	configPath string
}

type devStopAirOptions struct {
	configPath string
}

type devNotifyOptions struct{}

type devSupervisor struct {
	moduleRoot      string
	migrationDir    string
	supervisorPID   string
	airPID          string
	servePID        string
	notifyPID       string
	appliedSnapshot string
	serveCmd        *exec.Cmd
	serveExit       chan error
	airCmd          *exec.Cmd
	airExit         chan error
}

type devPIDPaths struct {
	supervisor string
	air        string
	serve      string
	notify     string
}

var devMigrateRunner = func(cmd *cobra.Command, migrationDir string) error {
	return runMigrateUp(cmd, migrateUpOptions{migrationDir: migrationDir})
}

var devAirModuleRootResolver = resolveBackendModuleRoot
var devAirLookPath = exec.LookPath
var devCommandContext = backendCommandContext
var devCommandEnv = buildBackendCommandEnv
var devExecutablePath = os.Executable
var devMkdirAll = os.MkdirAll
var devReadFile = os.ReadFile
var devWriteFile = os.WriteFile
var devRemove = os.Remove
var devStat = os.Stat
var devReadDir = os.ReadDir
var devSignalNotifyContext = signal.NotifyContext
var devProcessFinder = os.FindProcess
var devPIDSignaler = signalDevPID
var devPIDAliveChecker = isDevPIDAlive
var devMigrationDirResolver = func(moduleRoot string, migrationDir string) ([]string, error) {
	return resolveMigrationDirs(moduleRoot, migrationDir)
}
var devAfter = time.After

func newDevCommand() *cobra.Command {
	var opts devOptions

	command := &cobra.Command{
		Use:   "dev",
		Short: "Run the Graft development supervisor",
		Long: "graft dev runs the local development supervisor. " +
			"It keeps `graft serve` as a child process, applies explicit migrations when live migration files change, " +
			"and preserves the current server when a migration fails.",
		Example:      "  graft dev\n  graft dev air\n  graft dev notify",
		SilenceUsage: true,
		Args:         cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDevSupervisor(cmd, opts, false)
		},
	}

	command.Flags().StringVar(&opts.migrationDir, "dir", defaultMigrationDir, "migration directory")
	command.AddCommand(newDevAirCommand())
	command.AddCommand(newDevNotifyCommand())
	command.AddCommand(newDevStopAirCommand())
	command.AddCommand(newDevResetAdminCommand())
	return command
}

func newDevAirCommand() *cobra.Command {
	opts := devAirOptions{configPath: ".air.toml"}

	command := &cobra.Command{
		Use:   "air",
		Short: "Run the development supervisor with Air build notifications",
		Long: "graft dev air starts the local development supervisor and an Air child process. " +
			"Air only rebuilds the server binary and triggers `graft dev notify`; the supervisor decides whether to " +
			"restart the server directly or apply migrations first.",
		Example:      "  graft dev air\n  graft dev air --config .air.toml",
		SilenceUsage: true,
		Args:         cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDevAir(cmd, opts)
		},
	}

	command.Flags().StringVar(&opts.configPath, "config", opts.configPath, "Air config file path")
	return command
}

func newDevNotifyCommand() *cobra.Command {
	opts := devNotifyOptions{}

	return &cobra.Command{
		Use:          "notify",
		Short:        "Notify the development supervisor that a fresh build is ready",
		SilenceUsage: true,
		Args:         cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDevNotify(cmd, opts)
		},
	}
}

func runDevAir(cmd *cobra.Command, opts devAirOptions) error {
	configPath, err := resolveDevAirConfigPath(opts.configPath)
	if err != nil {
		return fmt.Errorf("resolve Air config path: %w", err)
	}

	devOpts := devOptions{migrationDir: defaultMigrationDir}
	return runDevSupervisorWithAir(cmd, devOpts, configPath)
}

func runDevNotify(_ *cobra.Command, _ devNotifyOptions) error {
	pidPaths, err := resolveDevPIDPaths()
	if err != nil {
		return fmt.Errorf("resolve dev pid paths: %w", err)
	}

	pid, err := readDevPIDFile(pidPaths.supervisor)
	if err != nil {
		return fmt.Errorf("read supervisor pid: %w", err)
	}

	alive, err := devPIDAliveChecker(pid)
	if err != nil {
		return fmt.Errorf("check supervisor pid %d: %w", pid, err)
	}
	if !alive {
		return fmt.Errorf("supervisor pid %d is not running", pid)
	}

	if err := writeDevPIDFile(pidPaths.notify, pid); err != nil {
		return fmt.Errorf("write dev notify marker: %w", err)
	}

	return nil
}

func runDevSupervisor(cmd *cobra.Command, opts devOptions, withAir bool) error {
	return runDevSupervisorWithAirConfig(cmd, opts, withAir, "")
}

func runDevSupervisorWithAir(cmd *cobra.Command, opts devOptions, configPath string) error {
	return runDevSupervisorWithAirConfig(cmd, opts, true, configPath)
}

func runDevSupervisorWithAirConfig(cmd *cobra.Command, opts devOptions, withAir bool, configPath string) error {
	moduleRoot, err := devAirModuleRootResolver()
	if err != nil {
		return fmt.Errorf("resolve backend module root: %w", err)
	}

	pidPaths, err := resolveDevPIDPaths()
	if err != nil {
		return fmt.Errorf("resolve dev pid paths: %w", err)
	}

	if err := ensureNoLiveDevSupervisor(pidPaths.supervisor); err != nil {
		return err
	}

	supervisor := &devSupervisor{
		moduleRoot:    moduleRoot,
		migrationDir:  opts.migrationDir,
		supervisorPID: pidPaths.supervisor,
		airPID:        pidPaths.air,
		servePID:      pidPaths.serve,
		notifyPID:     pidPaths.notify,
	}

	if err := devMkdirAll(filepath.Dir(pidPaths.supervisor), devPIDDirPerm); err != nil {
		return fmt.Errorf("mkdir dev pid dir: %w", err)
	}
	removeDevPIDFile(pidPaths.notify)
	if err := writeDevPIDFile(pidPaths.supervisor, os.Getpid()); err != nil {
		return fmt.Errorf("write supervisor pid: %w", err)
	}
	defer removeDevPIDFile(pidPaths.supervisor)
	defer removeDevPIDFile(pidPaths.notify)

	runCtx, stop := devSignalNotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := supervisor.reconcile(cmd, true); err != nil {
		return err
	}

	if withAir {
		if err := supervisor.startAir(runCtx, cmd, configPath); err != nil {
			if shutdownErr := supervisor.shutdown(cmd); shutdownErr != nil {
				return fmt.Errorf("%w; cleanup failed: %v", err, shutdownErr)
			}
			return err
		}
	}

	return supervisor.loop(runCtx, cmd)
}

func (s *devSupervisor) loop(ctx context.Context, cmd *cobra.Command) error {
	for {
		select {
		case <-ctx.Done():
			return s.shutdown(cmd)
		case <-devAfter(devStatePollTimeout):
			ready, err := s.consumeBuildNotification()
			if err != nil {
				return err
			}
			if !ready {
				continue
			}
			s.log(cmd, "build complete")
			if err := s.reconcile(cmd, false); err != nil {
				return err
			}
		case err := <-s.serveExitChannel():
			s.handleServeExit(cmd, err)
		case err := <-s.airExitChannel():
			s.log(cmd, "air exited: %v", err)
			s.airCmd = nil
			removeDevPIDFile(s.airPID)
			return s.shutdown(cmd)
		}
	}
}

func (s *devSupervisor) reconcile(cmd *cobra.Command, forceMigrate bool) error {
	s.log(cmd, "checking migrations...")

	snapshot, err := s.liveMigrationSnapshot()
	if err != nil {
		return fmt.Errorf("snapshot live migrations: %w", err)
	}

	needsMigration := forceMigrate || snapshot != s.appliedSnapshot
	if !needsMigration {
		s.log(cmd, "no pending migrations")
		return s.restartServe(cmd)
	}

	if err := devMigrateRunner(cmd, s.migrationDir); err != nil {
		if s.serveCmd != nil {
			s.log(cmd, "migration failed, keeping existing server: %v", err)
			return nil
		}
		return fmt.Errorf("run development migrations: %w", err)
	}

	s.appliedSnapshot = snapshot
	s.log(cmd, "migration success")
	return s.restartServe(cmd)
}

func (s *devSupervisor) restartServe(cmd *cobra.Command) error {
	s.log(cmd, "restarting server")

	if err := s.stopServe(cmd); err != nil {
		return err
	}

	serveBinary, err := resolveDevServeBinary(s.moduleRoot)
	if err != nil {
		return fmt.Errorf("resolve serve binary: %w", err)
	}

	commandContext := cmd.Context()
	if commandContext == nil {
		commandContext = context.Background()
	}

	child := devCommandContext(commandContext, serveBinary, "serve")
	child.Dir = s.moduleRoot
	child.Stdout = cmd.OutOrStdout()
	child.Stderr = cmd.ErrOrStderr()
	child.Stdin = os.Stdin
	child.Env, err = devCommandEnv()
	if err != nil {
		return fmt.Errorf("prepare serve env: %w", err)
	}

	if err := child.Start(); err != nil {
		return fmt.Errorf("start development server: %w", err)
	}

	s.serveCmd = child
	s.serveExit = make(chan error, 1)
	if err := writeDevPIDFile(s.servePID, child.Process.Pid); err != nil {
		_ = child.Process.Kill()
		_, _ = child.Process.Wait()
		return fmt.Errorf("write serve pid: %w", err)
	}

	go func() {
		s.serveExit <- child.Wait()
	}()

	s.log(cmd, "server started pid=%d", child.Process.Pid)
	return nil
}

func (s *devSupervisor) stopServe(cmd *cobra.Command) error {
	if s.serveCmd == nil || s.serveCmd.Process == nil {
		removeDevPIDFile(s.servePID)
		return nil
	}

	pid := s.serveCmd.Process.Pid
	if err := devPIDSignaler(pid, syscall.SIGTERM); err != nil {
		return fmt.Errorf("stop serve pid %d: %w", pid, err)
	}

	if err := s.waitServeStop(pid); err != nil {
		return err
	}

	s.serveCmd = nil
	s.serveExit = nil
	removeDevPIDFile(s.servePID)
	if cmd != nil {
		s.log(cmd, "server stopped pid=%d", pid)
	}
	return nil
}

func (s *devSupervisor) waitServeStop(pid int) error {
	select {
	case <-s.serveExitChannel():
		return nil
	case <-devAfter(devStopTimeout):
	}

	if err := s.serveCmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return fmt.Errorf("kill serve pid %d: %w", pid, err)
	}

	select {
	case <-s.serveExitChannel():
	case <-devAfter(devStatePollTimeout):
	}

	return nil
}

func (s *devSupervisor) startAir(ctx context.Context, cmd *cobra.Command, configPath string) error {
	airPath, err := devAirLookPath("go")
	if err != nil {
		return fmt.Errorf("find go for Air: %w", err)
	}

	child := devCommandContext(ctx, airPath, "tool", "air", "-c", configPath)
	child.Dir = s.moduleRoot
	child.Stdout = cmd.OutOrStdout()
	child.Stderr = cmd.ErrOrStderr()
	child.Stdin = os.Stdin
	child.Env, err = devCommandEnv()
	if err != nil {
		return fmt.Errorf("prepare air env: %w", err)
	}

	if err := child.Start(); err != nil {
		return fmt.Errorf("start Air live reload: %w", err)
	}

	s.airCmd = child
	s.airExit = make(chan error, 1)
	if err := writeDevPIDFile(s.airPID, child.Process.Pid); err != nil {
		_ = child.Process.Kill()
		_, _ = child.Process.Wait()
		return fmt.Errorf("write Air pid: %w", err)
	}

	go func() {
		s.airExit <- child.Wait()
	}()

	s.log(cmd, "air started pid=%d", child.Process.Pid)
	return nil
}

func (s *devSupervisor) shutdown(cmd *cobra.Command) error {
	if err := s.stopServe(cmd); err != nil {
		return err
	}

	if s.airCmd != nil && s.airCmd.Process != nil {
		_ = devPIDSignaler(s.airCmd.Process.Pid, syscall.SIGTERM)
		removeDevPIDFile(s.airPID)
	}

	return nil
}

func (s *devSupervisor) consumeBuildNotification() (bool, error) {
	pid, err := readDevPIDFile(s.notifyPID)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		removeDevPIDFile(s.notifyPID)
		return false, fmt.Errorf("read dev notify marker: %w", err)
	}

	removeDevPIDFile(s.notifyPID)
	return pid == os.Getpid(), nil
}

func (s *devSupervisor) liveMigrationSnapshot() (string, error) {
	dirs, err := devMigrationDirResolver(s.moduleRoot, s.migrationDir)
	if err != nil {
		return "", err
	}

	hash := sha256.New()
	slices.Sort(dirs)
	for _, dir := range dirs {
		entries, err := devReadDir(dir)
		if err != nil {
			return "", fmt.Errorf("read migration dir %s: %w", dir, err)
		}

		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
				continue
			}

			path := filepath.Join(dir, entry.Name())
			content, err := devReadFile(path)
			if err != nil {
				return "", fmt.Errorf("read migration file %s: %w", path, err)
			}

			_, _ = io.WriteString(hash, filepath.ToSlash(path))
			_, _ = hash.Write(content)
		}
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (s *devSupervisor) handleServeExit(cmd *cobra.Command, err error) {
	if s.serveCmd == nil {
		return
	}

	pid := s.serveCmd.Process.Pid
	s.serveCmd = nil
	s.serveExit = nil
	removeDevPIDFile(s.servePID)

	if err != nil && !errors.Is(err, os.ErrProcessDone) {
		s.log(cmd, "server exited pid=%d err=%v", pid, err)
		return
	}

	s.log(cmd, "server exited pid=%d", pid)
}

func (s *devSupervisor) serveExitChannel() <-chan error {
	if s.serveExit == nil {
		return nil
	}
	return s.serveExit
}

func (s *devSupervisor) airExitChannel() <-chan error {
	if s.airExit == nil {
		return nil
	}
	return s.airExit
}

func (s *devSupervisor) log(cmd *cobra.Command, format string, args ...any) {
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "[DEV] %s\n", fmt.Sprintf(format, args...))
}

func resolveDevAirConfigPath(configPath string) (string, error) {
	moduleRoot, err := devAirModuleRootResolver()
	if err != nil {
		return "", fmt.Errorf("resolve backend module root: %w", err)
	}
	return normalizeDevAirConfigPath(moduleRoot, configPath), nil
}

func normalizeDevAirConfigPath(baseDir string, configPath string) string {
	if filepath.IsAbs(configPath) {
		return filepath.Clean(configPath)
	}
	return filepath.Clean(filepath.Join(baseDir, configPath))
}

func ensureNoLiveDevSupervisor(supervisorPID string) error {
	pid, err := readDevPIDFile(supervisorPID)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read supervisor pid: %w", err)
	}

	alive, err := devPIDAliveChecker(pid)
	if err != nil {
		return fmt.Errorf("check supervisor pid %d: %w", pid, err)
	}
	if !alive {
		removeDevPIDFile(supervisorPID)
		return nil
	}

	return fmt.Errorf("another development supervisor is already running (pid=%d); stop it with `graft dev stop-air` before starting a new one", pid)
}

func resolveDevPIDPaths() (devPIDPaths, error) {
	moduleRoot, err := devAirModuleRootResolver()
	if err != nil {
		return devPIDPaths{}, fmt.Errorf("resolve backend module root: %w", err)
	}

	tmpDir := filepath.Join(moduleRoot, "tmp")
	return devPIDPaths{
		supervisor: filepath.Join(tmpDir, devSupervisorPIDName),
		air:        filepath.Join(tmpDir, devAirPIDName),
		serve:      filepath.Join(tmpDir, devServePIDName),
		notify:     filepath.Join(tmpDir, devNotifyPIDName),
	}, nil
}

func writeDevPIDFile(path string, pid int) error {
	if err := devMkdirAll(filepath.Dir(path), devPIDDirPerm); err != nil {
		return err
	}
	return devWriteFile(path, []byte(fmt.Sprintf("%d\n", pid)), devPIDFilePerm)
}

func readDevPIDFile(path string) (int, error) {
	content, err := devReadFile(path)
	if err != nil {
		return 0, err
	}

	var pid int
	if _, err := fmt.Sscanf(strings.TrimSpace(string(content)), "%d", &pid); err != nil {
		return 0, fmt.Errorf("parse pid file %s: %w", path, err)
	}
	if pid <= 0 {
		return 0, fmt.Errorf("invalid pid %d in %s", pid, path)
	}
	return pid, nil
}

func removeDevPIDFile(path string) {
	if err := devRemove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return
	}
}

func isDevPIDAlive(pid int) (bool, error) {
	process, err := devProcessFinder(pid)
	if err != nil {
		return false, err
	}
	if err := process.Signal(syscall.Signal(0)); err != nil {
		if errors.Is(err, os.ErrProcessDone) || errors.Is(err, syscall.ESRCH) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func signalDevPID(pid int, sig syscall.Signal) error {
	process, err := devProcessFinder(pid)
	if err != nil {
		return fmt.Errorf("find process: %w", err)
	}
	if err := process.Signal(sig); err != nil {
		if errors.Is(err, os.ErrProcessDone) || errors.Is(err, syscall.ESRCH) {
			return nil
		}
		return err
	}
	return nil
}

func resolveDevServeBinary(moduleRoot string) (string, error) {
	candidate := filepath.Join(moduleRoot, "tmp", "graft")
	if runtime.GOOS == "windows" {
		candidate += ".exe"
	}
	if _, err := devStat(candidate); err == nil {
		return candidate, nil
	}
	if _, err := devStat(candidate); err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("stat %s: %w", candidate, err)
	}

	current, err := devExecutablePath()
	if err != nil {
		return "", fmt.Errorf("resolve current executable: %w", err)
	}
	return current, nil
}
