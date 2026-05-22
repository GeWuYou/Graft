package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

type devStopAirOptions struct {
	configPath string
}

const (
	devServeArgvMinLen = 2
	devAirArgvMinLen   = 4
)

type devProcessSnapshot struct {
	pid  int
	ppid int
	argv []string
	cwd  string
}

var devStopAirModuleRootResolver = resolveBackendModuleRoot
var devStopAirListProcesses = listDevProcesses
var devStopAirSignal = signalDevProcess

func newDevStopAirCommand() *cobra.Command {
	opts := devStopAirOptions{configPath: ".air.toml"}

	command := &cobra.Command{
		Use:   "stop-air",
		Short: "Stop the local Air live reload process for this repository",
		Long: "graft dev stop-air stops the Air live reload parent process and any current " +
			"`tmp/graft serve` child started from this repository's `server` module.",
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

func runDevStopAir(cmd *cobra.Command, opts devStopAirOptions) error {
	moduleRoot, err := devStopAirModuleRootResolver()
	if err != nil {
		return fmt.Errorf("resolve backend module root: %w", err)
	}

	servePIDs, airPIDs, err := findDevAirTargets(moduleRoot, opts.configPath)
	if err != nil {
		return fmt.Errorf("find Air processes: %w", err)
	}

	if len(servePIDs) == 0 && len(airPIDs) == 0 {
		return writeDevStopAirResult(
			cmd.OutOrStdout(),
			"no Air development process found under %s\n",
			moduleRoot,
		)
	}

	if err := stopDevProcesses(servePIDs, "serve"); err != nil {
		return err
	}
	if err := stopDevProcesses(airPIDs, "Air"); err != nil {
		return err
	}

	return writeDevStopAirResult(
		cmd.OutOrStdout(),
		"stopped Air development processes under %s: serve=%d air=%d\n",
		moduleRoot,
		len(servePIDs),
		len(airPIDs),
	)
}

func stopDevProcesses(pids []int, label string) error {
	for _, pid := range pids {
		if err := devStopAirSignal(pid, syscall.SIGTERM); err != nil {
			return fmt.Errorf("stop %s process %d: %w", label, pid, err)
		}
	}
	return nil
}

func writeDevStopAirResult(writer io.Writer, format string, args ...any) error {
	if _, err := io.WriteString(writer, fmt.Sprintf(format, args...)); err != nil {
		return fmt.Errorf("write stop-air result: %w", err)
	}
	return nil
}

func findDevAirTargets(moduleRoot string, configPath string) ([]int, []int, error) {
	if runtime.GOOS != "linux" {
		return nil, nil, fmt.Errorf("stop-air currently supports linux only, got %s", runtime.GOOS)
	}

	processes, err := devStopAirListProcesses()
	if err != nil {
		return nil, nil, err
	}

	normalizedRoot := filepath.Clean(moduleRoot)
	expectedServeBinary := filepath.Join(normalizedRoot, "tmp", "graft")
	expectedConfigPath := normalizeDevAirConfigPath(normalizedRoot, configPath)

	var servePIDs []int
	var airPIDs []int
	for _, process := range processes {
		if matchesDevServeProcess(process, expectedServeBinary) {
			servePIDs = append(servePIDs, process.pid)
		}
		if matchesDevAirProcess(process, normalizedRoot, expectedConfigPath) {
			airPIDs = append(airPIDs, process.pid)
		}
	}

	slices.Sort(servePIDs)
	slices.Sort(airPIDs)
	return slices.Compact(servePIDs), slices.Compact(airPIDs), nil
}

func matchesDevServeProcess(process devProcessSnapshot, expectedServeBinary string) bool {
	if len(process.argv) < devServeArgvMinLen {
		return false
	}
	return filepath.Clean(process.argv[0]) == expectedServeBinary && process.argv[1] == "serve"
}

func matchesDevAirProcess(process devProcessSnapshot, moduleRoot string, expectedConfigPath string) bool {
	if len(process.argv) < devAirArgvMinLen {
		return false
	}
	if filepath.Clean(process.cwd) != moduleRoot {
		return false
	}
	if process.argv[0] != "go" || process.argv[1] != "tool" || process.argv[2] != "air" {
		return false
	}

	for index := 3; index < len(process.argv)-1; index++ {
		if process.argv[index] != "-c" {
			continue
		}
		actualConfigPath := normalizeDevAirConfigPath(process.cwd, process.argv[index+1])
		return actualConfigPath == expectedConfigPath
	}

	return false
}

func normalizeDevAirConfigPath(baseDir string, configPath string) string {
	if filepath.IsAbs(configPath) {
		return filepath.Clean(configPath)
	}
	return filepath.Clean(filepath.Join(baseDir, configPath))
}

func listDevProcesses() ([]devProcessSnapshot, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, fmt.Errorf("read /proc: %w", err)
	}

	processes := make([]devProcessSnapshot, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		process, ok, err := readDevProcessSnapshot(pid)
		if err != nil {
			if shouldIgnoreDevProcessReadError(err) {
				continue
			}
			return nil, err
		}
		if ok {
			processes = append(processes, process)
		}
	}

	return processes, nil
}

func readDevProcessSnapshot(pid int) (devProcessSnapshot, bool, error) {
	cmdlineBytes, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "cmdline"))
	if err != nil {
		return devProcessSnapshot{}, false, err
	}
	if len(cmdlineBytes) == 0 {
		return devProcessSnapshot{}, false, nil
	}

	statusFile, err := os.Open(filepath.Join("/proc", strconv.Itoa(pid), "status"))
	if err != nil {
		return devProcessSnapshot{}, false, err
	}
	defer func() {
		_ = statusFile.Close()
	}()

	ppid, err := readDevProcessPPID(statusFile)
	if err != nil {
		return devProcessSnapshot{}, false, fmt.Errorf("read /proc/%d/status: %w", pid, err)
	}

	cwd, err := os.Readlink(filepath.Join("/proc", strconv.Itoa(pid), "cwd"))
	if err != nil {
		return devProcessSnapshot{}, false, err
	}

	return devProcessSnapshot{
		pid:  pid,
		ppid: ppid,
		argv: splitProcCmdline(cmdlineBytes),
		cwd:  cwd,
	}, true, nil
}

func readDevProcessPPID(reader io.Reader) (int, error) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "PPid:") {
			continue
		}
		ppidText := strings.TrimSpace(strings.TrimPrefix(line, "PPid:"))
		ppid, err := strconv.Atoi(ppidText)
		if err != nil {
			return 0, fmt.Errorf("parse PPid %q: %w", ppidText, err)
		}
		return ppid, nil
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	return 0, errors.New("missing PPid field")
}

func splitProcCmdline(cmdlineBytes []byte) []string {
	parts := strings.Split(string(cmdlineBytes), "\x00")
	argv := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		argv = append(argv, part)
	}
	return argv
}

func shouldIgnoreDevProcessReadError(err error) bool {
	return errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ESRCH)
}

func signalDevProcess(pid int, signal syscall.Signal) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("find process: %w", err)
	}

	if err := process.Signal(signal); err != nil {
		if errors.Is(err, os.ErrProcessDone) || errors.Is(err, syscall.ESRCH) {
			return nil
		}
		return err
	}

	return nil
}
