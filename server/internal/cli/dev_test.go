package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/spf13/cobra"
)

func TestRunDevNotifyWritesSupervisorMarker(t *testing.T) {
	root := t.TempDir()
	serverRoot := filepath.Join(root, "server")
	tmpDir := filepath.Join(serverRoot, "tmp")
	if err := os.MkdirAll(tmpDir, 0o750); err != nil {
		t.Fatalf("mkdir tmp: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, devSupervisorPIDName), []byte("42\n"), 0o600); err != nil {
		t.Fatalf("write supervisor pid: %v", err)
	}

	originalResolver := devAirModuleRootResolver
	originalAliveChecker := devPIDAliveChecker
	defer func() {
		devAirModuleRootResolver = originalResolver
		devPIDAliveChecker = originalAliveChecker
	}()

	devAirModuleRootResolver = func() (string, error) {
		return serverRoot, nil
	}

	devPIDAliveChecker = func(pid int) (bool, error) {
		if pid != 42 {
			t.Fatalf("expected pid 42, got %d", pid)
		}
		return true, nil
	}

	err := runDevNotify(&cobra.Command{}, devNotifyOptions{})
	if err != nil {
		t.Fatalf("run dev notify: %v", err)
	}
	pid, err := readDevPIDFile(filepath.Join(tmpDir, devNotifyPIDName))
	if err != nil {
		t.Fatalf("read notify marker pid: %v", err)
	}
	if pid != 42 {
		t.Fatalf("expected notify marker for pid 42, got %d", pid)
	}
}

func TestConsumeBuildNotificationConsumesOwnMarker(t *testing.T) {
	notifyPath := filepath.Join(t.TempDir(), devNotifyPIDName)
	foreignPID := os.Getpid() + 100000
	if err := os.WriteFile(notifyPath, []byte(fmt.Sprintf("%d\n", foreignPID)), 0o600); err != nil {
		t.Fatalf("write foreign marker: %v", err)
	}

	supervisor := &devSupervisor{notifyPID: notifyPath}
	ready, err := supervisor.consumeBuildNotification()
	if err != nil {
		t.Fatalf("consume foreign marker: %v", err)
	}
	if ready {
		t.Fatal("expected foreign marker to be ignored")
	}
	if _, err := os.Stat(notifyPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected foreign marker removed, got err=%v", err)
	}

	if err := os.WriteFile(notifyPath, []byte("bad-pid\n"), 0o600); err != nil {
		t.Fatalf("write malformed marker: %v", err)
	}
	ready, err = supervisor.consumeBuildNotification()
	if err == nil {
		t.Fatal("expected malformed marker error")
	}
	if ready {
		t.Fatal("malformed marker must not be ready")
	}
	if _, err := os.Stat(notifyPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected malformed marker removed, got err=%v", err)
	}

	if err := writeDevPIDFile(notifyPath, os.Getpid()); err != nil {
		t.Fatalf("write own marker: %v", err)
	}
	ready, err = supervisor.consumeBuildNotification()
	if err != nil {
		t.Fatalf("consume own marker: %v", err)
	}
	if !ready {
		t.Fatal("expected own marker to be ready")
	}
	if _, err := os.Stat(notifyPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected own marker removed, got err=%v", err)
	}
}

func TestRunDevSupervisorWithAirStartFailureStopsServe(t *testing.T) {
	root := t.TempDir()
	serverRoot := filepath.Join(root, "server")
	if err := os.MkdirAll(filepath.Join(serverRoot, "tmp"), 0o750); err != nil {
		t.Fatalf("mkdir tmp: %v", err)
	}

	originalResolver := devAirModuleRootResolver
	originalAliveChecker := devPIDAliveChecker
	originalMigrationResolver := devMigrationDirResolver
	originalMigrateRunner := devMigrateRunner
	originalCommandContext := devCommandContext
	originalCommandEnv := devCommandEnv
	originalAirLookPath := devAirLookPath
	originalSignalNotifyContext := devSignalNotifyContext
	defer func() {
		devAirModuleRootResolver = originalResolver
		devPIDAliveChecker = originalAliveChecker
		devMigrationDirResolver = originalMigrationResolver
		devMigrateRunner = originalMigrateRunner
		devCommandContext = originalCommandContext
		devCommandEnv = originalCommandEnv
		devAirLookPath = originalAirLookPath
		devSignalNotifyContext = originalSignalNotifyContext
	}()

	devAirModuleRootResolver = func() (string, error) {
		return serverRoot, nil
	}
	devPIDAliveChecker = func(_ int) (bool, error) {
		return false, nil
	}
	devMigrationDirResolver = func(_ string, _ string) ([]string, error) {
		return nil, nil
	}
	devMigrateRunner = func(_ *cobra.Command, _ string) error {
		return nil
	}
	devCommandContext = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "sleep", "30")
	}
	devCommandEnv = func() ([]string, error) {
		return os.Environ(), nil
	}
	devAirLookPath = func(_ string) (string, error) {
		return "", errors.New("go unavailable")
	}
	devSignalNotifyContext = func(parent context.Context, _ ...os.Signal) (context.Context, context.CancelFunc) {
		return context.WithCancel(parent)
	}

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	err := runDevSupervisorWithAirConfig(cmd, devOptions{}, true, ".air.toml")
	if err == nil {
		t.Fatal("expected Air startup error")
	}
	if !strings.Contains(err.Error(), "find go for Air") {
		t.Fatalf("expected Air lookup error, got %v", err)
	}
	for _, name := range []string{devSupervisorPIDName, devServePIDName, devNotifyPIDName} {
		if _, err := os.Stat(filepath.Join(serverRoot, "tmp", name)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected %s cleaned up, got err=%v", name, err)
		}
	}
}

func TestEnsureNoLiveDevSupervisorRejectsAlivePID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dev-supervisor.pid")
	if err := os.WriteFile(path, []byte("77\n"), 0o600); err != nil {
		t.Fatalf("write pid file: %v", err)
	}

	originalAliveChecker := devPIDAliveChecker
	defer func() {
		devPIDAliveChecker = originalAliveChecker
	}()

	devPIDAliveChecker = func(pid int) (bool, error) {
		if pid != 77 {
			t.Fatalf("expected pid 77, got %d", pid)
		}
		return true, nil
	}

	err := ensureNoLiveDevSupervisor(path)
	if err == nil {
		t.Fatal("expected live supervisor error")
	}
	if !strings.Contains(err.Error(), "graft dev stop-air") {
		t.Fatalf("expected stop-air guidance, got %v", err)
	}
}

func TestEnsureNoLiveDevSupervisorRemovesStalePID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dev-supervisor.pid")
	if err := os.WriteFile(path, []byte("78\n"), 0o600); err != nil {
		t.Fatalf("write pid file: %v", err)
	}

	originalAliveChecker := devPIDAliveChecker
	defer func() {
		devPIDAliveChecker = originalAliveChecker
	}()

	devPIDAliveChecker = func(pid int) (bool, error) {
		if pid != 78 {
			t.Fatalf("expected pid 78, got %d", pid)
		}
		return false, nil
	}

	err := ensureNoLiveDevSupervisor(path)
	if err != nil {
		t.Fatalf("ensure no live supervisor: %v", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected stale pid file removed, got err=%v", err)
	}
}

func TestServerAirConfigUsesNotifyEntrypoint(t *testing.T) {
	content, err := os.ReadFile("../../.air.toml")
	if err != nil {
		t.Fatalf("read ../../.air.toml: %v", err)
	}

	config := string(content)
	if !strings.Contains(config, `entrypoint = ["./tmp/graft", "dev", "notify"]`) {
		t.Fatalf("expected Air config to notify the dev supervisor, got:\n%s", config)
	}
	if strings.Contains(config, `entrypoint = ["./tmp/graft", "serve"]`) {
		t.Fatalf("Air config must not launch serve directly anymore:\n%s", config)
	}
	if !strings.Contains(config, `entrypoint = ['tmp\graft.exe', "dev", "notify"]`) {
		t.Fatalf("expected Windows Air config to notify the dev supervisor, got:\n%s", config)
	}
}

func TestSignalDevPIDIgnoresMissingProcess(t *testing.T) {
	originalFinder := devProcessFinder
	defer func() {
		devProcessFinder = originalFinder
	}()

	devProcessFinder = func(_ int) (*os.Process, error) {
		return nil, errors.New("lookup failed")
	}

	if err := signalDevPID(1, syscall.SIGTERM); err == nil {
		t.Fatal("expected lookup error")
	}
}

func TestNewRootCommandRegistersDevNotifyCommand(t *testing.T) {
	command := NewRootCommand()

	found, _, err := command.Find([]string{"dev", "notify"})
	if err != nil {
		t.Fatalf("find dev notify command: %v", err)
	}
	if found == nil || found.Name() != "notify" {
		t.Fatalf("expected notify command, got %#v", found)
	}
}
