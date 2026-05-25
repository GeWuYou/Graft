package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/spf13/cobra"

	"graft/server/internal/config"
)

const (
	defaultSmokeHealthPath = "/healthz"
	defaultSmokeTimeout    = 10 * time.Second
	defaultSmokeProbeDelay = 200 * time.Millisecond
	defaultBackendStage    = "full"
	defaultOpenAPIStage    = "openapi"

	defaultBackendLintConfig     = ".golangci.yml"
	defaultBackendTestLintConfig = ".golangci.test.yml"
	defaultGolangCILintVersion   = "v2.12.2"
	defaultBackendCacheRoot      = "graft-backend-validate"
	defaultBackendDirPerm        = 0o755
	defaultHealthProbeReadLimit  = 256
	defaultLintBaseRefEnv        = "GRAFT_LINT_BASE_REF"
	githubBaseRefEnv             = "GITHUB_BASE_REF"
	defaultRemoteName            = "origin"
	defaultRemoteHeadRef         = "refs/remotes/origin/HEAD"
	defaultOpenAPIRootSpec       = "openapi/openapi.yaml"
	shaLength40                  = 40
	shaLength64                  = 64
)

// smokeValidateOptions 封装最小运行时 smoke 验证的显式输入。
type smokeValidateOptions struct {
	migrationDir string
	healthPath   string
	timeout      time.Duration
}

// backendValidateOptions 封装后端统一质量链的显式输入。
type backendValidateOptions struct {
	stage          string
	lintConfig     string
	testLintConfig string
	testTargets    []string
	smoke          bool
	openapiSpec    string
}

var smokeMigrateRunner = func(cmd *cobra.Command, migrationDir string) error {
	return runMigrateUp(cmd, migrateUpOptions{migrationDir: migrationDir})
}

var smokeServeRunner = runServe
var smokeLoadConfig = config.Load
var smokeHealthChecker = waitForSmokeHealth
var backendLintRunner = runBackendLint
var backendGoTestRunner = runBackendGoTest
var backendGoBuildRunner = runBackendGoBuild
var backendSmokeRunner = runValidateSmoke
var backendOpenAPIRunner = runValidateOpenAPI
var backendOpenAPIFreshnessRunner = runValidateOpenAPIFreshness
var backendCommandRunner = runBackendCommand
var backendGitOutputRunner = runBackendGitOutput

var backendLookPath = exec.LookPath
var backendCommandContext = exec.CommandContext
var backendGetwd = os.Getwd
var backendReadFile = os.ReadFile
var backendMkdirAll = os.MkdirAll
var backendGetenv = os.Getenv
var backendGitRevisionPattern = regexp.MustCompile(`\A[0-9A-Fa-f]+\z`)

// newValidateCommand 创建后端显式验证命令树。
//
// 这里的命令只编排仓库内已经存在的迁移与运行时入口，不负责隐式拉起
// disposable 基础设施，避免把环境准备魔法塞进 core 或 CLI 黑盒里。
func newValidateCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "validate",
		Short: "Run explicit backend validation commands",
	}

	command.AddCommand(newValidateBackendCommand())
	command.AddCommand(newValidateOpenAPICommand())
	command.AddCommand(newValidateSmokeCommand())
	return command
}

// newValidateBackendCommand 创建后端统一质量链命令。
//
// 该命令显式收口 `server` 的 lint、test、build 与可选 smoke 顺序，避免
// agent、本地开发与 CI 分别维护第二套验证参数或隐式脚本魔法。
func newValidateBackendCommand() *cobra.Command {
	opts := backendValidateOptions{
		stage:          defaultBackendStage,
		lintConfig:     defaultBackendLintConfig,
		testLintConfig: defaultBackendTestLintConfig,
		openapiSpec:    defaultOpenAPIRootSpec,
	}

	command := &cobra.Command{
		Use:   "backend",
		Short: "Run the unified backend quality chain",
		Long: "graft validate backend is the repository-local backend quality entrypoint. " +
			"It runs OpenAPI validation first, then golangci-lint, then executes go test on the requested scope, " +
			"then builds ./cmd/graft, and optionally appends `graft validate smoke` when the slice needs a runtime proof.",
		Example: "  graft validate backend\n" +
			"  graft validate backend --stage openapi\n" +
			"  graft validate backend --test-target ./plugins/user --test-target ./internal/httpx\n" +
			"  graft validate backend --stage lint\n" +
			"  graft validate backend --smoke",
		SilenceUsage: true,
		Args:         cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runValidateBackend(cmd, opts)
		},
	}

	command.Flags().StringVar(&opts.stage, "stage", defaultBackendStage, "validation stage: openapi, lint, buildtest, or full")
	command.Flags().StringVar(&opts.lintConfig, "lint-config", defaultBackendLintConfig, "golangci-lint config for non-test code")
	command.Flags().StringVar(&opts.testLintConfig, "test-lint-config", defaultBackendTestLintConfig, "golangci-lint config for test code")
	command.Flags().StringArrayVar(&opts.testTargets, "test-target", nil, "go test package target to validate; repeatable, defaults to ./...")
	command.Flags().BoolVar(&opts.smoke, "smoke", false, "append `graft validate smoke` after lint, test, and build")
	command.Flags().StringVar(&opts.openapiSpec, "openapi-spec", defaultOpenAPIRootSpec, "root OpenAPI spec path relative to repository root")
	return command
}

func newValidateOpenAPICommand() *cobra.Command {
	var specPath string

	command := &cobra.Command{
		Use:          "openapi",
		Short:        "Validate the root OpenAPI spec",
		SilenceUsage: true,
		Args:         cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runValidateOpenAPI(cmd, specPath)
		},
	}

	command.Flags().StringVar(&specPath, "spec", defaultOpenAPIRootSpec, "root OpenAPI spec path relative to repository root")
	return command
}

// newValidateSmokeCommand 创建最小 smoke 验证子命令。
//
// 该命令显式执行一次迁移、启动运行时、探测 `/healthz`，然后主动停止服务，
// 用来把现有 disposable PostgreSQL/Redis 验证流压缩成一个仓库内入口。
func newValidateSmokeCommand() *cobra.Command {
	var opts smokeValidateOptions

	command := &cobra.Command{
		Use:   "smoke",
		Short: "Run migrations and probe the runtime health endpoint once",
		Long: "graft validate smoke is an explicit backend smoke validation command. " +
			"It runs migrations first, starts the runtime against already-prepared infrastructure, " +
			"waits for the health endpoint to become ready, and then shuts the runtime down.",
		Example:      "  graft validate smoke\n  graft validate smoke --timeout 15s",
		SilenceUsage: true,
		Args:         cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runValidateSmoke(cmd, opts)
		},
	}

	command.Flags().StringVar(&opts.migrationDir, "dir", defaultMigrationDir, "migration directory")
	command.Flags().StringVar(&opts.healthPath, "health-path", defaultSmokeHealthPath, "health probe path")
	command.Flags().DurationVar(&opts.timeout, "timeout", defaultSmokeTimeout, "maximum time to wait for the smoke health probe")
	return command
}

// runValidateBackend 执行统一的后端质量链。
//
// 顺序语义：
//   - `openapi` 阶段显式验证根 OpenAPI 文档及其 fragments。
//   - `lint` 阶段固定先跑生产代码和测试代码两套 golangci-lint 配置，显式收口
//     不同阈值，而不是靠一份模糊配置同时兼顾两类目标。
//   - `buildtest` 阶段固定执行最小直接覆盖范围的 `go test`，随后构建 `./cmd/graft`。
//   - `full` 阶段串联完整顺序；只有 `full` 才允许追加 `--smoke`，避免跳过前置门禁。
func runValidateBackend(cmd *cobra.Command, opts backendValidateOptions) error {
	stage := strings.TrimSpace(opts.stage)
	if stage == "" {
		stage = defaultBackendStage
	}
	if strings.TrimSpace(opts.openapiSpec) == "" {
		opts.openapiSpec = defaultOpenAPIRootSpec
	}
	if err := validateBackendStageOptions(stage, opts.smoke); err != nil {
		return err
	}

	switch stage {
	case defaultOpenAPIStage:
		return backendOpenAPIRunner(cmd, opts.openapiSpec)
	case "lint":
		return backendLintRunner(cmd, opts.lintConfig, opts.testLintConfig)
	case "buildtest":
		return runBackendBuildTest(cmd, opts.testTargets)
	case "full":
		return runFullBackendValidation(cmd, opts)
	default:
		return fmt.Errorf("unsupported backend validation stage %q: expected openapi, lint, buildtest, or full", stage)
	}
}

func validateBackendStageOptions(stage string, smoke bool) error {
	if !smoke || stage == "full" {
		return nil
	}

	return errors.New("`--smoke` requires `--stage full` so lint, test, and build stay in a fixed order")
}

func runFullBackendValidation(cmd *cobra.Command, opts backendValidateOptions) error {
	if err := backendOpenAPIRunner(cmd, opts.openapiSpec); err != nil {
		return err
	}
	if err := backendLintRunner(cmd, opts.lintConfig, opts.testLintConfig); err != nil {
		return err
	}
	if err := runBackendBuildTest(cmd, opts.testTargets); err != nil {
		return err
	}
	if !opts.smoke {
		return nil
	}
	if err := backendSmokeRunner(cmd, smokeValidateOptions{
		migrationDir: defaultMigrationDir,
		healthPath:   defaultSmokeHealthPath,
		timeout:      defaultSmokeTimeout,
	}); err != nil {
		return fmt.Errorf("run backend smoke validation: %w", err)
	}

	return nil
}

func runValidateOpenAPI(_ *cobra.Command, specPath string) error {
	specPath = strings.TrimSpace(specPath)
	if specPath == "" {
		specPath = defaultOpenAPIRootSpec
	}

	repoRoot, err := resolveRepositoryRoot()
	if err != nil {
		return fmt.Errorf("resolve repository root for openapi validation: %w", err)
	}

	rootSpec := filepath.Join(repoRoot, filepath.FromSlash(specPath))
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	document, err := loader.LoadFromFile(rootSpec)
	if err != nil {
		return fmt.Errorf("load openapi spec %q: %w", rootSpec, err)
	}
	if err := document.Validate(loader.Context); err != nil {
		return fmt.Errorf("validate openapi spec %q: %w", rootSpec, err)
	}

	if err := backendOpenAPIFreshnessRunner(); err != nil {
		return err
	}

	return nil
}

func runValidateOpenAPIFreshness() error {
	repoRoot, err := resolveRepositoryRoot()
	if err != nil {
		return fmt.Errorf("resolve repository root for generated freshness validation: %w", err)
	}

	scriptPath := filepath.Join(repoRoot, "scripts", "openapi_generated_freshness_check.py")
	cmd := &cobra.Command{}
	if err := backendCommandRunner(cmd, "python3", scriptPath, "--target", "backend-monitor", "--mode", "check"); err != nil {
		return fmt.Errorf("run backend generated freshness check: %w", err)
	}
	if err := backendCommandRunner(cmd, "python3", scriptPath, "--target", "backend-rbac-permissions", "--mode", "check"); err != nil {
		return fmt.Errorf("run backend generated freshness check: %w", err)
	}

	return nil
}

// runBackendBuildTest 执行 `go test -> go build ./cmd/graft` 的后端编译验证链。
func runBackendBuildTest(cmd *cobra.Command, testTargets []string) error {
	targets := append([]string(nil), testTargets...)
	if len(targets) == 0 {
		targets = []string{"./..."}
	}

	if err := backendGoTestRunner(cmd, targets); err != nil {
		return err
	}
	if err := backendGoBuildRunner(cmd); err != nil {
		return err
	}
	return nil
}

// runBackendLint 通过统一入口执行后端 lint。
//
// 这里不直接维护第二套 lint 参数，而是回到仓库统一 CLI，让本地、CI 和 agent
// 共用同一条入口和同一套配置文件约束。
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

func isBackendGitRevision(baseRef string) bool {
	trimmed := strings.TrimSpace(baseRef)
	if len(trimmed) != shaLength40 && len(trimmed) != shaLength64 {
		return false
	}
	return backendGitRevisionPattern.MatchString(trimmed)
}

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

func currentBackendGitHead(cmd *cobra.Command, workingDir string) string {
	headRef, err := backendGitOutputRunner(cmd, workingDir, "rev-parse", "HEAD")
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(headRef)
}

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

// runBackendGoTest 执行显式的 `go test` 验证。
func runBackendGoTest(cmd *cobra.Command, targets []string) error {
	args := append([]string{"test"}, targets...)
	if err := backendCommandRunner(cmd, "go", args...); err != nil {
		return fmt.Errorf("run go test on %s: %w", strings.Join(targets, " "), err)
	}
	return nil
}

// runBackendGoBuild 执行显式的 `go build ./cmd/graft` 编译验证。
func runBackendGoBuild(cmd *cobra.Command) error {
	if err := backendCommandRunner(cmd, "go", "build", "./cmd/graft"); err != nil {
		return fmt.Errorf("run go build ./cmd/graft: %w", err)
	}
	return nil
}

// findGolangCILint 解析本地可执行的 golangci-lint 路径。
//
// 仓库固定使用同一版本，缺失时直接给出带版本号的下一步提示，避免开发者和
// agent 回退到 `latest` 或一组漂移的本地安装方式。
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

// runBackendCommand 以 `server` 模块根目录执行后端显式验证子命令。
func runBackendCommand(cmd *cobra.Command, name string, args ...string) error {
	commandContext := cmd.Context()
	if commandContext == nil {
		commandContext = context.Background()
	}

	workingDir, err := resolveBackendModuleRoot()
	if err != nil {
		return fmt.Errorf("resolve working directory: %w", err)
	}

	command := backendCommandContext(commandContext, name, args...)
	command.Dir = workingDir
	command.Stdout = cmd.OutOrStdout()
	command.Stderr = cmd.ErrOrStderr()
	command.Stdin = os.Stdin
	command.Env, err = buildBackendCommandEnv()
	if err != nil {
		return fmt.Errorf("prepare backend command env: %w", err)
	}

	if err := command.Run(); err != nil {
		if commandContext.Err() != nil {
			return commandContext.Err()
		}
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}

	return nil
}

// resolveBackendModuleRoot 从当前工作目录向上定位 `server` 模块根目录。
//
// 该解析允许 `graft validate backend` 在仓库根目录或 `server` 目录下运行，
// 同时确保 lint、test、build 始终以 `server/go.mod` 作为相对路径基准。
func resolveBackendModuleRoot() (string, error) {
	current, err := backendGetwd()
	if err != nil {
		return "", fmt.Errorf("resolve current directory: %w", err)
	}

	for {
		moduleDir, matched, err := matchBackendModuleRoot(current)
		if err != nil {
			return "", err
		}
		if matched {
			return moduleDir, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return "", errors.New("cannot locate server module root from current directory")
}

func resolveRepositoryRoot() (string, error) {
	moduleRoot, err := resolveBackendModuleRoot()
	if err != nil {
		return "", fmt.Errorf("resolve backend module root for repository root: %w", err)
	}
	repoRoot := filepath.Dir(moduleRoot)
	if _, err := backendReadFile(filepath.Join(repoRoot, "AGENTS.md")); err == nil {
		return repoRoot, nil
	}

	return "", errors.New("could not locate repository root adjacent to server module")
}

// matchBackendModuleRoot 判断当前目录或其 `server` 子目录是否是 `server` 模块根。
func matchBackendModuleRoot(dir string) (string, bool, error) {
	goModPath := filepath.Join(dir, "go.mod")
	if content, err := backendReadFile(goModPath); err == nil {
		if strings.Contains(string(content), "module graft/server") {
			return dir, true, nil
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", false, fmt.Errorf("read %s: %w", goModPath, err)
	}

	serverDir := filepath.Join(dir, "server")
	serverGoModPath := filepath.Join(serverDir, "go.mod")
	if content, err := backendReadFile(serverGoModPath); err == nil {
		if strings.Contains(string(content), "module graft/server") {
			return serverDir, true, nil
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", false, fmt.Errorf("read %s: %w", serverGoModPath, err)
	}

	return "", false, nil
}

// buildBackendCommandEnv 为后端子命令准备可写缓存目录。
//
// 某些运行环境会把默认的 `$HOME/.cache` 设为只读；这里把 `go` 与
// `golangci-lint` 的缓存统一导向系统临时目录，避免统一质量链受宿主缓存策略影响。
func buildBackendCommandEnv() ([]string, error) {
	moduleRoot, err := resolveBackendModuleRoot()
	if err != nil {
		return nil, fmt.Errorf("resolve backend module root for cache env: %w", err)
	}

	cacheRoot := filepath.Join(os.TempDir(), defaultBackendCacheRoot, backendCacheNamespace(moduleRoot))
	goCacheDir := filepath.Join(cacheRoot, "go-build")
	xdgCacheDir := filepath.Join(cacheRoot, "xdg")
	golangciCacheDir := filepath.Join(cacheRoot, "golangci-lint")

	for _, dir := range []string{cacheRoot, goCacheDir, xdgCacheDir, golangciCacheDir} {
		if err := backendMkdirAll(dir, defaultBackendDirPerm); err != nil {
			return nil, fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}

	env := os.Environ()
	env = append(env,
		"GOCACHE="+goCacheDir,
		"XDG_CACHE_HOME="+xdgCacheDir,
		"GOLANGCI_LINT_CACHE="+golangciCacheDir,
	)
	return env, nil
}

func backendCacheNamespace(moduleRoot string) string {
	sum := sha256.Sum256([]byte(moduleRoot))
	return hex.EncodeToString(sum[:8])
}

// runValidateSmoke 执行最小运行时 smoke 验证闭环。
//
// 顺序语义：
//   - 先执行显式迁移，保持 schema 变更入口仍然可见。
//   - 再启动运行时并轮询健康检查，避免把成功判断退化为“进程未立刻退出”。
//   - 健康检查成功后主动取消运行时上下文，验证服务可以完成一次最小启动与关闭。
func runValidateSmoke(cmd *cobra.Command, opts smokeValidateOptions) error {
	if err := smokeMigrateRunner(cmd, opts.migrationDir); err != nil {
		return fmt.Errorf("run smoke migrations: %w", err)
	}

	cfg, err := smokeLoadConfig()
	if err != nil {
		return fmt.Errorf("load smoke config: %w", err)
	}

	probeURL, err := buildSmokeProbeURL(cfg.HTTP.Addr, opts.healthPath)
	if err != nil {
		return fmt.Errorf("build smoke probe url: %w", err)
	}

	parentCtx := cmd.Context()
	if parentCtx == nil {
		parentCtx = context.Background()
	}

	runCtx, cancelRun := context.WithCancel(parentCtx)
	defer cancelRun()

	serveCommand := &cobra.Command{}
	serveCommand.SetContext(runCtx)
	serveCommand.SetOut(cmd.OutOrStdout())
	serveCommand.SetErr(cmd.ErrOrStderr())

	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- smokeServeRunner(serveCommand, nil)
	}()

	probeCtx, cancelProbe := context.WithTimeout(runCtx, opts.timeout)
	defer cancelProbe()

	healthErrCh := make(chan error, 1)
	go func() {
		healthErrCh <- smokeHealthChecker(probeCtx, probeURL)
	}()

	if err := waitForSmokeStartup(cancelProbe, cancelRun, serveErrCh, healthErrCh); err != nil {
		return err
	}

	cancelRun()
	if err := <-serveErrCh; err != nil {
		return fmt.Errorf("shutdown smoke server: %w", err)
	}

	return nil
}

func waitForSmokeStartup(
	cancelProbe context.CancelFunc,
	cancelRun context.CancelFunc,
	serveErrCh <-chan error,
	healthErrCh <-chan error,
) error {
	select {
	case serveErr := <-serveErrCh:
		cancelProbe()
		if serveErr == nil {
			return errors.New("smoke runtime exited before health probe completed")
		}
		return fmt.Errorf("run smoke server: %w", serveErr)
	case err := <-healthErrCh:
		if err == nil {
			return nil
		}
		cancelRun()
		serveErr := <-serveErrCh
		if serveErr == nil {
			return fmt.Errorf("wait for smoke health check: %w", err)
		}
		return errors.Join(
			fmt.Errorf("wait for smoke health check: %w", err),
			fmt.Errorf("run smoke server: %w", serveErr),
		)
	}
}

// buildSmokeProbeURL 把监听地址转换为本地健康检查 URL。
//
// 为了兼容 `:8080`、`0.0.0.0:8080` 与 `127.0.0.1:8080` 这类常见监听写法，
// 这里会把 wildcard 地址归一化为 loopback 探测目标。
func buildSmokeProbeURL(listenAddr string, healthPath string) (string, error) {
	normalizedPath := strings.TrimSpace(healthPath)
	if normalizedPath == "" {
		return "", errors.New("health path is required")
	}
	if !strings.HasPrefix(normalizedPath, "/") {
		normalizedPath = "/" + normalizedPath
	}

	host, port, err := net.SplitHostPort(strings.TrimSpace(listenAddr))
	if err != nil {
		return "", fmt.Errorf("parse listen address %q: %w", listenAddr, err)
	}

	switch strings.TrimSpace(host) {
	case "", "0.0.0.0", "::":
		host = "127.0.0.1"
	}

	return "http://" + net.JoinHostPort(host, port) + normalizedPath, nil
}

// waitForSmokeHealth 轮询健康检查接口直到成功或超时。
//
// 这里把“服务真的完成最小启动”的判定收敛为一次 `200 OK` 响应，避免把
// smoke 验证简化成只看进程是否存活。
func waitForSmokeHealth(ctx context.Context, probeURL string) error {
	client := &http.Client{
		Timeout: time.Second,
	}

	ticker := time.NewTicker(defaultSmokeProbeDelay)
	defer ticker.Stop()

	var lastErr error
	for {
		err := probeSmokeHealthOnce(ctx, client, probeURL)
		if err == nil {
			return nil
		}
		lastErr = err

		select {
		case <-ctx.Done():
			if lastErr != nil {
				return fmt.Errorf("probe %s: %w", probeURL, lastErr)
			}
			return fmt.Errorf("probe %s: %w", probeURL, ctx.Err())
		case <-ticker.C:
		}
	}
}

func probeSmokeHealthOnce(ctx context.Context, client *http.Client, probeURL string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, probeURL, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(io.LimitReader(response.Body, defaultHealthProbeReadLimit))
		if readErr != nil {
			return fmt.Errorf("health probe returned %s and read body failed: %w", response.Status, readErr)
		}

		trimmedBody := strings.TrimSpace(string(body))
		if trimmedBody == "" {
			return fmt.Errorf("health probe returned %s", response.Status)
		}

		return fmt.Errorf("health probe returned %s: %s", response.Status, trimmedBody)
	}

	return nil
}
