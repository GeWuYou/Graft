package cli

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"syscall"
	"testing"

	"github.com/spf13/cobra"
)

func TestFindDevAirTargetsMatchesOnlyCurrentModule(t *testing.T) {
	originalListProcesses := devStopAirListProcesses
	defer func() {
		devStopAirListProcesses = originalListProcesses
	}()

	devStopAirListProcesses = func() ([]devProcessSnapshot, error) {
		return []devProcessSnapshot{
			{
				pid:  101,
				argv: []string{"/repo/server/tmp/graft", "serve"},
				cwd:  "/repo/server",
			},
			{
				pid:  102,
				argv: []string{"go", "tool", "air", "-c", ".air.toml"},
				cwd:  "/repo/server",
			},
			{
				pid:  201,
				argv: []string{"/other/server/tmp/graft", "serve"},
				cwd:  "/other/server",
			},
			{
				pid:  202,
				argv: []string{"go", "tool", "air", "-c", ".air.toml"},
				cwd:  "/other/server",
			},
		}, nil
	}

	servePIDs, airPIDs, err := findDevAirTargets("/repo/server", ".air.toml")
	if runtimeGOOS() != "linux" {
		if err == nil {
			t.Fatal("expected non-linux error")
		}
		return
	}
	if err != nil {
		t.Fatalf("find dev air targets: %v", err)
	}
	if !reflect.DeepEqual(servePIDs, []int{101}) {
		t.Fatalf("expected serve pid [101], got %v", servePIDs)
	}
	if !reflect.DeepEqual(airPIDs, []int{102}) {
		t.Fatalf("expected air pid [102], got %v", airPIDs)
	}
}

func TestRunDevStopAirSignalsServeBeforeAir(t *testing.T) {
	originalResolver := devStopAirModuleRootResolver
	originalListProcesses := devStopAirListProcesses
	originalSignal := devStopAirSignal
	defer func() {
		devStopAirModuleRootResolver = originalResolver
		devStopAirListProcesses = originalListProcesses
		devStopAirSignal = originalSignal
	}()

	devStopAirModuleRootResolver = func() (string, error) {
		return "/repo/server", nil
	}
	devStopAirListProcesses = func() ([]devProcessSnapshot, error) {
		return []devProcessSnapshot{
			{pid: 21, argv: []string{"/repo/server/tmp/graft", "serve"}, cwd: "/repo/server"},
			{pid: 22, argv: []string{"go", "tool", "air", "-c", ".air.toml"}, cwd: "/repo/server"},
		}, nil
	}

	var gotSignals []string
	devStopAirSignal = func(pid int, signal syscall.Signal) error {
		gotSignals = append(gotSignals, strconvItoa(pid)+":"+strconvItoa(int(signal)))
		return nil
	}

	var stdout bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&stdout)

	err := runDevStopAir(cmd, devStopAirOptions{configPath: ".air.toml"})
	if runtimeGOOS() != "linux" {
		if err == nil {
			t.Fatal("expected non-linux error")
		}
		return
	}
	if err != nil {
		t.Fatalf("run dev stop-air: %v", err)
	}

	expectedSignals := []string{"21:15", "22:15"}
	if !reflect.DeepEqual(gotSignals, expectedSignals) {
		t.Fatalf("expected signals %v, got %v", expectedSignals, gotSignals)
	}
	if !strings.Contains(stdout.String(), "serve=1 air=1") {
		t.Fatalf("expected stop-air output, got %q", stdout.String())
	}
}

func TestRunDevStopAirWritesNoopWhenNothingFound(t *testing.T) {
	originalResolver := devStopAirModuleRootResolver
	originalListProcesses := devStopAirListProcesses
	defer func() {
		devStopAirModuleRootResolver = originalResolver
		devStopAirListProcesses = originalListProcesses
	}()

	devStopAirModuleRootResolver = func() (string, error) {
		return "/repo/server", nil
	}
	devStopAirListProcesses = func() ([]devProcessSnapshot, error) {
		return nil, nil
	}

	var stdout bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&stdout)

	err := runDevStopAir(cmd, devStopAirOptions{configPath: ".air.toml"})
	if runtimeGOOS() != "linux" {
		if err == nil {
			t.Fatal("expected non-linux error")
		}
		return
	}
	if err != nil {
		t.Fatalf("run dev stop-air: %v", err)
	}
	if !strings.Contains(stdout.String(), "no Air development process found") {
		t.Fatalf("expected noop output, got %q", stdout.String())
	}
}

func TestRunDevStopAirWrapsSignalError(t *testing.T) {
	originalResolver := devStopAirModuleRootResolver
	originalListProcesses := devStopAirListProcesses
	originalSignal := devStopAirSignal
	defer func() {
		devStopAirModuleRootResolver = originalResolver
		devStopAirListProcesses = originalListProcesses
		devStopAirSignal = originalSignal
	}()

	devStopAirModuleRootResolver = func() (string, error) {
		return "/repo/server", nil
	}
	devStopAirListProcesses = func() ([]devProcessSnapshot, error) {
		return []devProcessSnapshot{
			{pid: 31, argv: []string{"/repo/server/tmp/graft", "serve"}, cwd: "/repo/server"},
		}, nil
	}
	devStopAirSignal = func(_ int, _ syscall.Signal) error {
		return errors.New("permission denied")
	}

	err := runDevStopAir(&cobra.Command{}, devStopAirOptions{configPath: ".air.toml"})
	if runtimeGOOS() != "linux" {
		if err == nil {
			t.Fatal("expected non-linux error")
		}
		return
	}
	if err == nil {
		t.Fatal("expected stop-air error")
	}
	if !strings.Contains(err.Error(), "stop serve process 31") {
		t.Fatalf("expected wrapped signal error, got %v", err)
	}
}

func TestReadDevProcessPPIDParsesStatus(t *testing.T) {
	ppid, err := readDevProcessPPID(strings.NewReader("Name:\tgo\nPPid:\t42\n"))
	if err != nil {
		t.Fatalf("read ppid: %v", err)
	}
	if ppid != 42 {
		t.Fatalf("expected ppid 42, got %d", ppid)
	}
}

func TestSplitProcCmdlineDropsTrailingSeparator(t *testing.T) {
	argv := splitProcCmdline([]byte("go\x00tool\x00air\x00-c\x00.air.toml\x00"))
	expected := []string{"go", "tool", "air", "-c", ".air.toml"}
	if !reflect.DeepEqual(argv, expected) {
		t.Fatalf("expected argv %v, got %v", expected, argv)
	}
}

func runtimeGOOS() string {
	return runtime.GOOS
}

func strconvItoa(value int) string {
	return fmt.Sprintf("%d", value)
}
