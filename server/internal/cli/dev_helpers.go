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

// startDevManagedChild 启动受管的开发子进程并写入其 PID 文件。
// 它返回已启动的子进程、用于接收退出结果的通道，以及错误。
// 成功启动后会在后台等待子进程退出，并将等待结果发送到返回的通道中。
// @returns 子进程、退出结果通道，以及启动或 PID 文件写入失败时的错误。
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

// buildDevChildCommand 构造并配置开发子进程的命令。
// 该命令使用给定的工作目录、可执行文件和参数，并继承标准输入、输出与错误输出；
// 当 ctx 为空时，使用 context.Background()。
// 返回配置完成的子进程命令，或在准备环境失败时返回错误。
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

// cleanupDevStartedChild 终止并回收已启动的子进程。
func cleanupDevStartedChild(child *exec.Cmd) {
	if child == nil || child.Process == nil {
		return
	}
	_ = child.Process.Kill()
	_, _ = child.Process.Wait()
}

// resolveDevAirConfigPath 将 dev air 配置路径解析为相对于后端模块根目录的路径。
// configPath 可以是绝对路径或相对路径；相对路径会以模块根目录为基准进行归一化。
// @param configPath 配置文件路径。
// @return 解析后的配置路径，或在无法解析模块根目录时返回错误。
func resolveDevAirConfigPath(configPath string) (string, error) {
	moduleRoot, err := devAirModuleRootResolver()
	if err != nil {
		return "", fmt.Errorf("resolve backend module root: %w", err)
	}
	return normalizeDevAirConfigPath(moduleRoot, configPath), nil
}

// normalizeDevAirConfigPath 将配置路径规范化为绝对路径或相对于基础目录的清理后路径。
// @param baseDir 基础目录。
// @param configPath 待规范化的配置路径。
// @returns 规范化后的路径。
func normalizeDevAirConfigPath(baseDir string, configPath string) string {
	if filepath.IsAbs(configPath) {
		return filepath.Clean(configPath)
	}
	return filepath.Clean(filepath.Join(baseDir, configPath))
}

// ensureNoLiveDevSupervisor 检查指定的 supervisor PID 文件对应的进程是否仍在运行。
// 如果 PID 文件不存在，或文件中的进程已退出，则返回 nil；如果检测到另一个开发监督进程仍然存活，则返回错误。
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

// 它返回 supervisor、air、serve 和 notify 对应的路径。
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

// 以文本形式将 pid 写入 path。
func writeDevPIDFile(path string, pid int) error {
	if err := devMkdirAll(filepath.Dir(path), devPIDDirPerm); err != nil {
		return err
	}
	return devWriteFile(path, []byte(fmt.Sprintf("%d\n", pid)), devPIDFilePerm)
}

// readDevPIDFile 读取并解析 PID 文件中的进程 ID。
// 返回文件中的 PID；如果读取失败、内容无法解析或 PID 无效，则返回错误。
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

// removeDevPIDFile 删除指定路径的 PID 文件，文件不存在时忽略。
// 其他删除错误也会被忽略。
func removeDevPIDFile(path string) {
	if err := devRemove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return
	}
}

// isDevPIDAlive 判断指定 PID 对应的进程是否仍然存活。
// 
// @returns `true` 如果进程仍然存在，`false` 如果进程已结束或不存在；当无法检查进程状态时返回错误。
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

// signalDevPID 向指定 PID 发送信号。
//
// 当进程已结束或不存在时，该函数返回 nil。
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

// @returns 可执行文件路径，或解析失败时的错误。
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
