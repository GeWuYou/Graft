package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/spf13/cobra"
)

func TestRunDevStopAirStopsTrackedPIDFiles(t *testing.T) {
	root := t.TempDir()
	serverRoot := filepath.Join(root, "server")
	tmpDir := filepath.Join(serverRoot, "tmp")
	if err := os.MkdirAll(tmpDir, 0o750); err != nil {
		t.Fatalf("mkdir tmp: %v", err)
	}

	originalResolver := devAirModuleRootResolver
	originalSignal := devStopAirSignal
	defer func() {
		devAirModuleRootResolver = originalResolver
		devStopAirSignal = originalSignal
	}()

	devAirModuleRootResolver = func() (string, error) {
		return serverRoot, nil
	}

	for _, item := range []struct {
		name string
		pid  int
	}{
		{name: devSupervisorPIDName, pid: 11},
		{name: devAirPIDName, pid: 12},
		{name: devServePIDName, pid: 13},
	} {
		if err := os.WriteFile(filepath.Join(tmpDir, item.name), []byte(fmt.Sprintf("%d\n", item.pid)), 0o600); err != nil {
			t.Fatalf("write %s: %v", item.name, err)
		}
	}
	if err := os.WriteFile(filepath.Join(tmpDir, devNotifyPIDName), []byte("11\n"), 0o600); err != nil {
		t.Fatalf("write notify marker: %v", err)
	}

	var got []string
	devStopAirSignal = func(pid int, signal syscall.Signal) error {
		got = append(got, fmt.Sprintf("%d:%d", pid, signal))
		return nil
	}

	var stdout bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&stdout)

	if err := runDevStopAir(cmd, devStopAirOptions{}); err != nil {
		t.Fatalf("run dev stop-air: %v", err)
	}

	expected := []string{"11:15", "12:15", "13:15"}
	if strings.Join(got, ",") != strings.Join(expected, ",") {
		t.Fatalf("expected %v, got %v", expected, got)
	}
	if !strings.Contains(stdout.String(), "supervisor=1 air=1 serve=1") {
		t.Fatalf("expected stop output, got %q", stdout.String())
	}
	for _, name := range []string{devSupervisorPIDName, devAirPIDName, devServePIDName, devNotifyPIDName} {
		if _, err := os.Stat(filepath.Join(tmpDir, name)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected %s removed, got err=%v", name, err)
		}
	}
}

func TestRunDevStopAirWritesNoopWhenNoPIDFiles(t *testing.T) {
	root := t.TempDir()
	serverRoot := filepath.Join(root, "server")
	if err := os.MkdirAll(filepath.Join(serverRoot, "tmp"), 0o750); err != nil {
		t.Fatalf("mkdir tmp: %v", err)
	}

	originalResolver := devAirModuleRootResolver
	defer func() {
		devAirModuleRootResolver = originalResolver
	}()
	devAirModuleRootResolver = func() (string, error) {
		return serverRoot, nil
	}

	var stdout bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&stdout)

	if err := runDevStopAir(cmd, devStopAirOptions{}); err != nil {
		t.Fatalf("run dev stop-air: %v", err)
	}
	if !strings.Contains(stdout.String(), "no development process found") {
		t.Fatalf("expected noop output, got %q", stdout.String())
	}
}

func TestRunDevStopAirWrapsSignalError(t *testing.T) {
	root := t.TempDir()
	serverRoot := filepath.Join(root, "server")
	tmpDir := filepath.Join(serverRoot, "tmp")
	if err := os.MkdirAll(tmpDir, 0o750); err != nil {
		t.Fatalf("mkdir tmp: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, devServePIDName), []byte("31\n"), 0o600); err != nil {
		t.Fatalf("write serve pid: %v", err)
	}

	originalResolver := devAirModuleRootResolver
	originalSignal := devStopAirSignal
	defer func() {
		devAirModuleRootResolver = originalResolver
		devStopAirSignal = originalSignal
	}()

	devAirModuleRootResolver = func() (string, error) {
		return serverRoot, nil
	}
	devStopAirSignal = func(_ int, _ syscall.Signal) error {
		return errors.New("permission denied")
	}

	err := runDevStopAir(&cobra.Command{}, devStopAirOptions{})
	if err == nil {
		t.Fatal("expected stop-air error")
	}
	if !strings.Contains(err.Error(), "stop serve process 31") {
		t.Fatalf("expected wrapped error, got %v", err)
	}
}
