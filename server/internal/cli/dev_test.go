package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/spf13/cobra"
)

func TestRunDevNotifySignalsSupervisorPID(t *testing.T) {
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
	originalSignaler := devPIDSignaler
	defer func() {
		devAirModuleRootResolver = originalResolver
		devPIDSignaler = originalSignaler
	}()

	devAirModuleRootResolver = func() (string, error) {
		return serverRoot, nil
	}

	signaled := false
	devPIDSignaler = func(pid int, _ syscall.Signal) error {
		if pid != 42 {
			t.Fatalf("expected pid 42, got %d", pid)
		}
		signaled = true
		return nil
	}

	err := runDevNotify(&cobra.Command{}, devNotifyOptions{})
	if err != nil {
		t.Fatalf("run dev notify: %v", err)
	}
	if !signaled {
		t.Fatal("expected notify to signal the supervisor pid")
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
