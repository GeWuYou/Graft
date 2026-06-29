package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// runBackendLint 通过统一入口执行后端 lint。
//
// 这里不直接维护第二套 lint 参数，而是回到仓库统一 CLI，让本地、CI 和 agent
// runBackendLint 使用同一组变更范围参数分别执行生产和测试两套 golangci-lint 配置。
// 其中任一配置执行失败时，都会返回带配置名上下文的错误。
func runBackendLint(cmd *cobra.Command, lintConfig string, testLintConfig string) error {
	lintPath, err := findGolangCILint()
	if err != nil {
		return err
	}

	lintArgs, err := buildBackendLintGateArgs(cmd)
	if err != nil {
		return err
	}

	if err := backendCommandRunner(cmd, lintPath, append([]string{"run", "--config", lintConfig}, lintArgs...)...); err != nil {
		return fmt.Errorf("run production golangci-lint config %q: %w", lintConfig, err)
	}
	if err := backendCommandRunner(cmd, lintPath, append([]string{"run", "--config", testLintConfig}, lintArgs...)...); err != nil {
		return fmt.Errorf("run test golangci-lint config %q: %w", testLintConfig, err)
	}
	return nil
}

// buildBackendLintGateArgs 生成后端 lint 的变更范围门控参数。
// 它会解析当前 HEAD、确定 base ref，并计算 merge-base 作为扫描起点。
// @return 以 `--new-from-rev=<mergeBase>` 和 `--whole-files` 组成的参数列表；或在解析失败时返回错误。
func buildBackendLintGateArgs(cmd *cobra.Command) ([]string, error) {
	workingDir, err := resolveBackendModuleRoot()
	if err != nil {
		return nil, fmt.Errorf("resolve backend lint working directory: %w", err)
	}

	headRef := currentBackendGitHead(cmd, workingDir)
	baseRef, baseRefSource, err := resolveBackendLintBaseRef(cmd, workingDir)
	if err != nil {
		return nil, err
	}

	mergeBase, err := resolveBackendLintMergeBase(cmd, workingDir, baseRef, baseRefSource)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(mergeBase) == "" {
		return nil, fmt.Errorf(
			"resolve backend lint merge-base for HEAD %q and base %q (source: %s): empty merge-base result",
			headRef,
			baseRef,
			baseRefSource,
		)
	}

	return []string{
		"--new-from-rev=" + mergeBase,
		"--whole-files",
	}, nil
}

// resolveBackendLintBaseRef 解析后端 lint 使用的基准引用及其来源。
// 优先读取 `defaultLintBaseRefEnv`，其次读取 `githubBaseRefEnv`；否则从 `origin/HEAD` 解析默认远程分支。
// @returns 规范化后的基准引用、其来源标识，以及错误。
func resolveBackendLintBaseRef(cmd *cobra.Command, workingDir string) (string, string, error) {
	if baseRef := strings.TrimSpace(backendGetenv(defaultLintBaseRefEnv)); baseRef != "" {
		return normalizeBackendLintBaseRef(baseRef), defaultLintBaseRefEnv, nil
	}
	if baseRef := strings.TrimSpace(backendGetenv(githubBaseRefEnv)); baseRef != "" {
		return normalizeBackendLintBaseRef(baseRef), githubBaseRefEnv, nil
	}

	remoteHead, err := backendGitOutputRunner(cmd, workingDir, "symbolic-ref", defaultRemoteHeadRef)
	if err != nil {
		return "", "", fmt.Errorf(
			"resolve backend lint base branch: %w; origin/HEAD is not available, run `git remote set-head %s -a` or set %s",
			err,
			defaultRemoteName,
			defaultLintBaseRefEnv,
		)
	}

	return strings.TrimSpace(remoteHead), "origin/HEAD", nil
}

// normalizeBackendLintBaseRef 将 baseRef 规范化为可用于 Git 的引用形式。
// 它会保留已是提交哈希或完整引用的值，并为简写分支名补全远端前缀。
func normalizeBackendLintBaseRef(baseRef string) string {
	trimmed := strings.TrimSpace(baseRef)
	switch {
	case isBackendGitRevision(trimmed):
		return trimmed
	case strings.HasPrefix(trimmed, "refs/remotes/"):
		return trimmed
	case strings.HasPrefix(trimmed, "refs/"):
		return trimmed
	case strings.Contains(trimmed, "/"):
		if strings.HasPrefix(trimmed, defaultRemoteName+"/") {
			return "refs/remotes/" + trimmed
		}
		return "refs/remotes/" + defaultRemoteName + "/" + trimmed
	default:
		return "refs/remotes/" + defaultRemoteName + "/" + trimmed
	}
}

// resolveBackendLintMergeBase 计算 `HEAD` 与指定基准引用的 merge-base。
// 当基准引用无法在本地解析或无法计算 merge-base 时，返回包含 `baseRefSource`、当前 `HEAD` 以及相应修复建议的错误。
func resolveBackendLintMergeBase(cmd *cobra.Command, workingDir string, baseRef string, baseRefSource string) (string, error) {
	if _, err := backendGitOutputRunner(cmd, workingDir, "rev-parse", "--verify", baseRef); err != nil {
		headRef := currentBackendGitHead(cmd, workingDir)
		if isBackendGitRevision(baseRef) {
			return "", fmt.Errorf(
				"backend lint base revision %q (source: %s) is not available locally for HEAD %q: %w; update %s to a reachable commit or ref",
				baseRef,
				baseRefSource,
				headRef,
				err,
				defaultLintBaseRefEnv,
			)
		}
		return "", fmt.Errorf(
			"backend lint base branch %q (source: %s) is not available locally for HEAD %q: %w; run `git fetch %s %s`",
			baseRef,
			baseRefSource,
			headRef,
			err,
			defaultRemoteName,
			backendLintFetchTarget(baseRef),
		)
	}

	mergeBase, err := backendGitOutputRunner(cmd, workingDir, "merge-base", "HEAD", baseRef)
	if err != nil {
		headRef := currentBackendGitHead(cmd, workingDir)
		if isBackendGitRevision(baseRef) {
			return "", fmt.Errorf(
				"resolve backend lint merge-base for HEAD %q and base %q (source: %s): %w; verify branch ancestry or set %s to a different reachable commit or ref",
				headRef,
				baseRef,
				baseRefSource,
				err,
				defaultLintBaseRefEnv,
			)
		}
		return "", fmt.Errorf(
			"resolve backend lint merge-base for HEAD %q and base %q (source: %s): %w; run `git fetch %s %s`, verify branch ancestry, or set %s",
			headRef,
			baseRef,
			baseRefSource,
			err,
			defaultRemoteName,
			backendLintFetchTarget(baseRef),
			defaultLintBaseRefEnv,
		)
	}

	return strings.TrimSpace(mergeBase), nil
}

// 它要求去除首尾空白后的长度为 40 或 64，且匹配预定义的 revision 模式。
func isBackendGitRevision(baseRef string) bool {
	trimmed := strings.TrimSpace(baseRef)
	if len(trimmed) != shaLength40 && len(trimmed) != shaLength64 {
		return false
	}
	return backendGitRevisionPattern.MatchString(trimmed)
}

// backendLintFetchTarget 将 baseRef 规范化为适合 `git fetch` 的目标。
// 它会去除常见的远端引用前缀；如果去除后为空，则返回原始值。
func backendLintFetchTarget(baseRef string) string {
	trimmed := strings.TrimSpace(baseRef)
	trimmed = strings.TrimPrefix(trimmed, "refs/remotes/"+defaultRemoteName+"/")
	trimmed = strings.TrimPrefix(trimmed, "refs/heads/")
	trimmed = strings.TrimPrefix(trimmed, defaultRemoteName+"/")
	if trimmed == "" {
		return baseRef
	}

	return trimmed
}

// currentBackendGitHead 获取工作目录中的当前 Git HEAD 引用。
// 解析失败时返回 "unknown"。
func currentBackendGitHead(cmd *cobra.Command, workingDir string) string {
	headRef, err := backendGitOutputRunner(cmd, workingDir, "rev-parse", "HEAD")
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(headRef)
}

// runBackendGitOutput 在指定工作目录中执行 `git` 命令并返回标准输出。
// 其中 `workingDir` 是命令执行目录。命令失败时返回包含 `git` 参数和底层错误的包装错误。
func runBackendGitOutput(cmd *cobra.Command, workingDir string, args ...string) (string, error) {
	commandContext := cmd.Context()
	if commandContext == nil {
		commandContext = context.Background()
	}

	command := backendCommandContext(commandContext, "git", args...)
	command.Dir = workingDir
	command.Stderr = cmd.ErrOrStderr()
	command.Stdin = os.Stdin
	command.Env = os.Environ()

	output, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}

	return strings.TrimSpace(string(output)), nil
}

// findGolangCILint 解析本地可执行的 golangci-lint 路径。
//
// 仓库固定使用同一版本，缺失时直接给出带版本号的下一步提示，避免开发者和
// findGolangCILint 查找 `golangci-lint` 可执行文件。
// 成功时返回可执行文件路径；如果未找到，则返回包含建议安装固定版本的错误。
func findGolangCILint() (string, error) {
	lintPath, err := backendLookPath("golangci-lint")
	if err == nil {
		return lintPath, nil
	}

	return "", fmt.Errorf(
		"golangci-lint %s is required for `graft validate backend`; install the pinned version before rerunning: %w",
		defaultGolangCILintVersion,
		err,
	)
}
