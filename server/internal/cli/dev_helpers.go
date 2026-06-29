package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

type devManagedChildSpec struct {
	moduleRoot string
	binary     string
	args       []string
	pidPath    string
	label      string
}

func startDevManagedChild(
	ctx context.Context,
	cmd *cobra.Command,
	spec devManagedChildSpec,
) (*exec.Cmd, chan error, error) {
	child, err := buildDevChildCommand(ctx, cmd, spec.moduleRoot, spec.binary, spec.args)
	if err != nil {
		return nil, nil, err
	}
	if err := child.Start(); err != nil {
		return nil, nil, err
	}
	if err := writeDevPIDFile(spec.pidPath, child.Process.Pid); err != nil {
		cleanupDevStartedChild(child)
		return nil, nil, fmt.Errorf("write %s pid: %w", spec.label, err)
	}

	exitCh := make(chan error, 1)
	go func() {
		exitCh <- child.Wait()
	}()

	return child, exitCh, nil
}

func buildDevChildCommand(
	ctx context.Context,
	cmd *cobra.Command,
	moduleRoot string,
	binary string,
	args []string,
) (*exec.Cmd, error) {
	commandContext := ctx
	if commandContext == nil {
		commandContext = context.Background()
	}

	child := devCommandContext(commandContext, binary, args...)
	child.Dir = moduleRoot
	child.Stdout = cmd.OutOrStdout()
	child.Stderr = cmd.ErrOrStderr()
	child.Stdin = os.Stdin

	env, err := devCommandEnv()
	if err != nil {
		return nil, fmt.Errorf("prepare child env: %w", err)
	}
	child.Env = env
	return child, nil
}

func cleanupDevStartedChild(child *exec.Cmd) {
	if child == nil || child.Process == nil {
		return
	}
	_ = child.Process.Kill()
	_, _ = child.Process.Wait()
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
