package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"graft/server/internal/config"
)

func newBackendModuleRootFixture(t *testing.T) string {
	t.Helper()

	tempDir := t.TempDir()
	serverDir := filepath.Join(tempDir, "server")
	if err := os.MkdirAll(serverDir, 0o750); err != nil {
		t.Fatalf("mkdir server dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(serverDir, "go.mod"), []byte("module graft/server\n"), 0o600); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	return serverDir
}

// TestResolveBackendModuleRootFromServerDir 验证在 `server` 目录运行时会直接识别模块根目录。
func TestResolveBackendModuleRootFromServerDir(t *testing.T) {
	originalGetwd := backendGetwd
	originalReadFile := backendReadFile
	defer func() {
		backendGetwd = originalGetwd
		backendReadFile = originalReadFile
	}()

	serverDir := newBackendModuleRootFixture(t)

	backendGetwd = func() (string, error) {
		return serverDir, nil
	}
	backendReadFile = os.ReadFile

	actual, err := resolveBackendModuleRoot()
	if err != nil {
		t.Fatalf("resolve backend module root: %v", err)
	}
	if actual != serverDir {
		t.Fatalf("expected %s, got %s", serverDir, actual)
	}
}

// TestResolveBackendModuleRootFromRepoRoot 验证从仓库根目录运行时会下钻到 `server` 模块根。
func TestResolveBackendModuleRootFromRepoRoot(t *testing.T) {
	originalGetwd := backendGetwd
	originalReadFile := backendReadFile
	defer func() {
		backendGetwd = originalGetwd
		backendReadFile = originalReadFile
	}()

	serverDir := newBackendModuleRootFixture(t)

	backendGetwd = func() (string, error) {
		return filepath.Dir(serverDir), nil
	}
	backendReadFile = os.ReadFile

	actual, err := resolveBackendModuleRoot()
	if err != nil {
		t.Fatalf("resolve backend module root: %v", err)
	}
	if actual != serverDir {
		t.Fatalf("expected %s, got %s", serverDir, actual)
	}
}

// TestRunValidateBackendLintStage 验证 lint 阶段会顺序执行两套 golangci-lint 配置。
func TestRunValidateBackendLintStage(t *testing.T) {
	originalLintRunner := backendLintRunner
	originalOpenAPIRunner := backendOpenAPIRunner
	originalGoTestRunner := backendGoTestRunner
	originalGoBuildRunner := backendGoBuildRunner
	originalSmokeRunner := backendSmokeRunner
	defer func() {
		backendLintRunner = originalLintRunner
		backendOpenAPIRunner = originalOpenAPIRunner
		backendGoTestRunner = originalGoTestRunner
		backendGoBuildRunner = originalGoBuildRunner
		backendSmokeRunner = originalSmokeRunner
	}()

	var steps []string
	backendLintRunner = func(_ *cobra.Command, lintConfig string, testLintConfig string) error {
		steps = append(steps, "lint:"+lintConfig+":"+testLintConfig)
		return nil
	}
	backendOpenAPIRunner = func(_ *cobra.Command, _ string) error {
		t.Fatal("openapi runner should not be called during lint stage")
		return nil
	}
	backendGoTestRunner = func(_ *cobra.Command, _ []string) error {
		t.Fatal("go test runner should not be called during lint stage")
		return nil
	}
	backendGoBuildRunner = func(_ *cobra.Command) error {
		t.Fatal("go build runner should not be called during lint stage")
		return nil
	}
	backendSmokeRunner = func(_ *cobra.Command, _ smokeValidateOptions) error {
		t.Fatal("smoke runner should not be called during lint stage")
		return nil
	}

	err := runValidateBackend(&cobra.Command{}, backendValidateOptions{
		stage:          "lint",
		lintConfig:     defaultBackendLintConfig,
		testLintConfig: defaultBackendTestLintConfig,
	})
	if err != nil {
		t.Fatalf("run validate backend lint stage: %v", err)
	}

	expected := []string{"lint:" + defaultBackendLintConfig + ":" + defaultBackendTestLintConfig}
	if !reflect.DeepEqual(steps, expected) {
		t.Fatalf("expected %v, got %v", expected, steps)
	}
}

func TestRunValidateBackendOpenAPIStage(t *testing.T) {
	originalOpenAPIRunner := backendOpenAPIRunner
	originalLintRunner := backendLintRunner
	defer func() {
		backendOpenAPIRunner = originalOpenAPIRunner
		backendLintRunner = originalLintRunner
	}()

	var calledWith string
	backendOpenAPIRunner = func(_ *cobra.Command, spec string) error {
		calledWith = spec
		return nil
	}
	backendLintRunner = func(_ *cobra.Command, _, _ string) error {
		t.Fatal("lint runner should not be called during openapi-only stage")
		return nil
	}

	err := runValidateBackend(&cobra.Command{}, backendValidateOptions{
		stage:       defaultOpenAPIStage,
		openapiSpec: defaultOpenAPIRootSpec,
	})
	if err != nil {
		t.Fatalf("run validate backend openapi stage: %v", err)
	}
	if calledWith != defaultOpenAPIRootSpec {
		t.Fatalf("expected spec %q, got %q", defaultOpenAPIRootSpec, calledWith)
	}
}

// TestRunBackendLintUsesChangedFileScopedArgs 验证 blocking lint gate 采用 changed-file scoped 语义，
// 并且不会在该路径里运行 full-repo audit。
func TestRunBackendLintUsesChangedFileScopedArgs(t *testing.T) {
	originalLookPath := backendLookPath
	originalCommandRunner := backendCommandRunner
	originalGitOutputRunner := backendGitOutputRunner
	originalGetwd := backendGetwd
	originalReadFile := backendReadFile
	originalGetenv := backendGetenv
	defer func() {
		backendLookPath = originalLookPath
		backendCommandRunner = originalCommandRunner
		backendGitOutputRunner = originalGitOutputRunner
		backendGetwd = originalGetwd
		backendReadFile = originalReadFile
		backendGetenv = originalGetenv
	}()

	serverDir := newBackendModuleRootFixture(t)

	backendLookPath = func(_ string) (string, error) {
		return "golangci-lint", nil
	}
	backendGetwd = func() (string, error) {
		return serverDir, nil
	}
	backendReadFile = os.ReadFile
	backendGetenv = func(key string) string {
		if key == defaultLintBaseRefEnv {
			return "main"
		}
		return ""
	}

	var gitCalls []string
	backendGitOutputRunner = func(_ *cobra.Command, _ string, args ...string) (string, error) {
		gitCalls = append(gitCalls, strings.Join(args, " "))
		switch strings.Join(args, " ") {
		case "rev-parse --verify refs/remotes/origin/main":
			return "refs/remotes/origin/main", nil
		case "merge-base HEAD refs/remotes/origin/main":
			return "abc123", nil
		case "rev-parse HEAD":
			return "head123", nil
		default:
			t.Fatalf("unexpected git args: %v", args)
			return "", nil
		}
	}

	var commands [][]string
	backendCommandRunner = func(_ *cobra.Command, name string, args ...string) error {
		if name != "golangci-lint" {
			t.Fatalf("unexpected command %q", name)
		}
		commands = append(commands, append([]string{name}, args...))
		return nil
	}

	if err := runBackendLint(&cobra.Command{}, defaultBackendLintConfig, defaultBackendTestLintConfig); err != nil {
		t.Fatalf("run backend lint: %v", err)
	}

	if len(commands) != 2 {
		t.Fatalf("expected 2 golangci-lint invocations, got %d", len(commands))
	}
	for _, command := range commands {
		joined := strings.Join(command, " ")
		if !strings.Contains(joined, "--new-from-rev=abc123") {
			t.Fatalf("expected changed-file scoped merge-base arg in %q", joined)
		}
		if !strings.Contains(joined, "--whole-files") {
			t.Fatalf("expected whole-file enforcement arg in %q", joined)
		}
		if strings.Contains(joined, "./...") {
			t.Fatalf("blocking lint gate must not run full-repo audit command, got %q", joined)
		}
	}
}

// TestResolveBackendLintBaseRefPrefersExplicitEnv 验证显式 base ref 优先于 CI 环境变量和 origin/HEAD。
func TestResolveBackendLintBaseRefPrefersExplicitEnv(t *testing.T) {
	originalGetenv := backendGetenv
	defer func() {
		backendGetenv = originalGetenv
	}()

	backendGetenv = func(key string) string {
		switch key {
		case defaultLintBaseRefEnv:
			return "release/next"
		case githubBaseRefEnv:
			return "main"
		default:
			return ""
		}
	}

	baseRef, source, err := resolveBackendLintBaseRef(&cobra.Command{}, t.TempDir())
	if err != nil {
		t.Fatalf("resolve backend lint base ref: %v", err)
	}
	if baseRef != "refs/remotes/origin/release/next" {
		t.Fatalf("expected explicit env base ref, got %q", baseRef)
	}
	if source != defaultLintBaseRefEnv {
		t.Fatalf("expected explicit env source, got %q", source)
	}
}

// TestResolveBackendLintBaseRefKeepsExplicitCommitSHA 验证显式 base ref 为 commit SHA 时不会被改写成 remote ref。
func TestResolveBackendLintBaseRefKeepsExplicitCommitSHA(t *testing.T) {
	originalGetenv := backendGetenv
	defer func() {
		backendGetenv = originalGetenv
	}()

	commitSHA := "7afe9b8eb396f2b1931998d248c02361d68a3fdc"
	backendGetenv = func(key string) string {
		if key == defaultLintBaseRefEnv {
			return commitSHA
		}
		return ""
	}

	baseRef, source, err := resolveBackendLintBaseRef(&cobra.Command{}, t.TempDir())
	if err != nil {
		t.Fatalf("resolve backend lint base ref: %v", err)
	}
	if baseRef != commitSHA {
		t.Fatalf("expected explicit commit SHA base ref, got %q", baseRef)
	}
	if source != defaultLintBaseRefEnv {
		t.Fatalf("expected explicit env source, got %q", source)
	}
}

// TestResolveBackendLintBaseRefFallsBackToRemoteHead 验证缺少显式 env 时会回退到 origin/HEAD。
func TestResolveBackendLintBaseRefFallsBackToRemoteHead(t *testing.T) {
	originalGetenv := backendGetenv
	originalGitOutputRunner := backendGitOutputRunner
	defer func() {
		backendGetenv = originalGetenv
		backendGitOutputRunner = originalGitOutputRunner
	}()

	backendGetenv = func(string) string {
		return ""
	}
	backendGitOutputRunner = func(_ *cobra.Command, _ string, args ...string) (string, error) {
		if !reflect.DeepEqual(args, []string{"symbolic-ref", defaultRemoteHeadRef}) {
			t.Fatalf("unexpected git args: %v", args)
		}
		return "refs/remotes/origin/main", nil
	}

	baseRef, source, err := resolveBackendLintBaseRef(&cobra.Command{}, t.TempDir())
	if err != nil {
		t.Fatalf("resolve backend lint base ref: %v", err)
	}
	if baseRef != "refs/remotes/origin/main" {
		t.Fatalf("expected origin/HEAD fallback, got %q", baseRef)
	}
	if source != "origin/HEAD" {
		t.Fatalf("expected origin/HEAD source, got %q", source)
	}
}

// TestResolveBackendLintBaseRefFailsWithoutOriginHead 验证 origin/HEAD 缺失时会给出显式修复提示。
func TestResolveBackendLintBaseRefFailsWithoutOriginHead(t *testing.T) {
	originalGetenv := backendGetenv
	originalGitOutputRunner := backendGitOutputRunner
	defer func() {
		backendGetenv = originalGetenv
		backendGitOutputRunner = originalGitOutputRunner
	}()

	backendGetenv = func(string) string {
		return ""
	}
	backendGitOutputRunner = func(_ *cobra.Command, _ string, _ ...string) (string, error) {
		return "", errors.New("symbolic-ref failed")
	}

	_, _, err := resolveBackendLintBaseRef(&cobra.Command{}, t.TempDir())
	if err == nil {
		t.Fatal("expected base ref resolution error")
	}
	if !strings.Contains(err.Error(), "git remote set-head origin -a") {
		t.Fatalf("expected origin/HEAD remediation, got %v", err)
	}
	if !strings.Contains(err.Error(), defaultLintBaseRefEnv) {
		t.Fatalf("expected explicit env remediation, got %v", err)
	}
}

// TestResolveBackendLintMergeBaseFailsWhenBaseMissing 验证 base branch 未 fetch 时返回 fetch 提示。
func TestResolveBackendLintMergeBaseFailsWhenBaseMissing(t *testing.T) {
	originalGitOutputRunner := backendGitOutputRunner
	defer func() {
		backendGitOutputRunner = originalGitOutputRunner
	}()

	backendGitOutputRunner = func(_ *cobra.Command, _ string, args ...string) (string, error) {
		switch strings.Join(args, " ") {
		case "rev-parse --verify refs/remotes/origin/main":
			return "", errors.New("missing base")
		case "rev-parse HEAD":
			return "head123", nil
		default:
			t.Fatalf("unexpected git args: %v", args)
			return "", nil
		}
	}

	_, err := resolveBackendLintMergeBase(&cobra.Command{}, t.TempDir(), "refs/remotes/origin/main", defaultLintBaseRefEnv)
	if err == nil {
		t.Fatal("expected merge-base resolution error")
	}
	if !strings.Contains(err.Error(), "git fetch origin main") {
		t.Fatalf("expected fetch remediation, got %v", err)
	}
	if !strings.Contains(err.Error(), "HEAD \"head123\"") {
		t.Fatalf("expected HEAD reference in error, got %v", err)
	}
	if !strings.Contains(err.Error(), "source: "+defaultLintBaseRefEnv) {
		t.Fatalf("expected base ref source in error, got %v", err)
	}
}

// TestResolveBackendLintMergeBaseAcceptsCommitSHA 验证 commit SHA 可以直接作为 merge-base 输入。
func TestResolveBackendLintMergeBaseAcceptsCommitSHA(t *testing.T) {
	originalGitOutputRunner := backendGitOutputRunner
	defer func() {
		backendGitOutputRunner = originalGitOutputRunner
	}()

	commitSHA := "7afe9b8eb396f2b1931998d248c02361d68a3fdc"
	backendGitOutputRunner = func(_ *cobra.Command, _ string, args ...string) (string, error) {
		switch strings.Join(args, " ") {
		case "rev-parse --verify " + commitSHA:
			return commitSHA, nil
		case "merge-base HEAD " + commitSHA:
			return "abc123", nil
		default:
			t.Fatalf("unexpected git args: %v", args)
			return "", nil
		}
	}

	mergeBase, err := resolveBackendLintMergeBase(&cobra.Command{}, t.TempDir(), commitSHA, defaultLintBaseRefEnv)
	if err != nil {
		t.Fatalf("resolve backend lint merge-base: %v", err)
	}
	if mergeBase != "abc123" {
		t.Fatalf("expected merge-base abc123, got %q", mergeBase)
	}
}

// TestResolveBackendLintMergeBaseFailsWhenCommitMissing 验证 commit SHA 缺失时返回 revision 级提示，而不是 branch fetch 提示。
func TestResolveBackendLintMergeBaseFailsWhenCommitMissing(t *testing.T) {
	originalGitOutputRunner := backendGitOutputRunner
	defer func() {
		backendGitOutputRunner = originalGitOutputRunner
	}()

	commitSHA := "7afe9b8eb396f2b1931998d248c02361d68a3fdc"
	backendGitOutputRunner = func(_ *cobra.Command, _ string, args ...string) (string, error) {
		switch strings.Join(args, " ") {
		case "rev-parse --verify " + commitSHA:
			return "", errors.New("missing revision")
		case "rev-parse HEAD":
			return "head123", nil
		default:
			t.Fatalf("unexpected git args: %v", args)
			return "", nil
		}
	}

	_, err := resolveBackendLintMergeBase(&cobra.Command{}, t.TempDir(), commitSHA, defaultLintBaseRefEnv)
	if err == nil {
		t.Fatal("expected merge-base resolution error")
	}
	if !strings.Contains(err.Error(), "base revision") {
		t.Fatalf("expected base revision wording, got %v", err)
	}
	if strings.Contains(err.Error(), "git fetch origin") {
		t.Fatalf("expected no branch fetch remediation for commit revision, got %v", err)
	}
	if !strings.Contains(err.Error(), "HEAD \"head123\"") {
		t.Fatalf("expected HEAD reference in error, got %v", err)
	}
}

// TestRunBackendLintUsesExplicitCommitBaseRef 验证 blocking lint gate 可以直接消费 commit SHA base ref。
func TestRunBackendLintUsesExplicitCommitBaseRef(t *testing.T) {
	originalLookPath := backendLookPath
	originalCommandRunner := backendCommandRunner
	originalGitOutputRunner := backendGitOutputRunner
	originalGetwd := backendGetwd
	originalReadFile := backendReadFile
	originalGetenv := backendGetenv
	defer func() {
		backendLookPath = originalLookPath
		backendCommandRunner = originalCommandRunner
		backendGitOutputRunner = originalGitOutputRunner
		backendGetwd = originalGetwd
		backendReadFile = originalReadFile
		backendGetenv = originalGetenv
	}()

	serverDir := newBackendModuleRootFixture(t)
	commitSHA := "7afe9b8eb396f2b1931998d248c02361d68a3fdc"

	backendLookPath = func(_ string) (string, error) {
		return "golangci-lint", nil
	}
	backendGetwd = func() (string, error) {
		return serverDir, nil
	}
	backendReadFile = os.ReadFile
	backendGetenv = func(key string) string {
		if key == defaultLintBaseRefEnv {
			return commitSHA
		}
		return ""
	}

	backendGitOutputRunner = func(_ *cobra.Command, _ string, args ...string) (string, error) {
		switch strings.Join(args, " ") {
		case "rev-parse --verify " + commitSHA:
			return commitSHA, nil
		case "merge-base HEAD " + commitSHA:
			return "abc123", nil
		case "rev-parse HEAD":
			return "head123", nil
		default:
			t.Fatalf("unexpected git args: %v", args)
			return "", nil
		}
	}

	var commands [][]string
	backendCommandRunner = func(_ *cobra.Command, name string, args ...string) error {
		if name != "golangci-lint" {
			t.Fatalf("unexpected command %q", name)
		}
		commands = append(commands, append([]string{name}, args...))
		return nil
	}

	if err := runBackendLint(&cobra.Command{}, defaultBackendLintConfig, defaultBackendTestLintConfig); err != nil {
		t.Fatalf("run backend lint: %v", err)
	}

	if len(commands) != 2 {
		t.Fatalf("expected 2 golangci-lint invocations, got %d", len(commands))
	}
	for _, command := range commands {
		joined := strings.Join(command, " ")
		if !strings.Contains(joined, "--new-from-rev=abc123") {
			t.Fatalf("expected changed-file scoped merge-base arg in %q", joined)
		}
		if !strings.Contains(joined, "--whole-files") {
			t.Fatalf("expected whole-file enforcement arg in %q", joined)
		}
	}
}

// TestResolveBackendLintMergeBaseFailsWithHeadAndBaseContext 验证 merge-base 失败时返回 HEAD、base 和修复提示。
func TestResolveBackendLintMergeBaseFailsWithHeadAndBaseContext(t *testing.T) {
	originalGitOutputRunner := backendGitOutputRunner
	defer func() {
		backendGitOutputRunner = originalGitOutputRunner
	}()

	backendGitOutputRunner = func(_ *cobra.Command, _ string, args ...string) (string, error) {
		switch strings.Join(args, " ") {
		case "rev-parse --verify refs/remotes/origin/main":
			return "refs/remotes/origin/main", nil
		case "merge-base HEAD refs/remotes/origin/main":
			return "", errors.New("merge-base failed")
		case "rev-parse HEAD":
			return "head123", nil
		default:
			t.Fatalf("unexpected git args: %v", args)
			return "", nil
		}
	}

	_, err := resolveBackendLintMergeBase(&cobra.Command{}, t.TempDir(), "refs/remotes/origin/main", defaultLintBaseRefEnv)
	if err == nil {
		t.Fatal("expected merge-base failure")
	}
	if !strings.Contains(err.Error(), "HEAD \"head123\"") {
		t.Fatalf("expected HEAD context, got %v", err)
	}
	if !strings.Contains(err.Error(), "base \"refs/remotes/origin/main\"") {
		t.Fatalf("expected base ref context, got %v", err)
	}
	if !strings.Contains(err.Error(), defaultLintBaseRefEnv) {
		t.Fatalf("expected explicit env remediation, got %v", err)
	}
	if !strings.Contains(err.Error(), "source: "+defaultLintBaseRefEnv) {
		t.Fatalf("expected base ref source in error, got %v", err)
	}
}

// TestRunValidateBackendLintStageDoesNotRunAudit 验证 blocking lint stage 不会衍生 full-repo audit 流程。
func TestRunValidateBackendLintStageDoesNotRunAudit(t *testing.T) {
	originalLintRunner := backendLintRunner
	originalOpenAPIRunner := backendOpenAPIRunner
	originalGoTestRunner := backendGoTestRunner
	originalGoBuildRunner := backendGoBuildRunner
	originalSmokeRunner := backendSmokeRunner
	defer func() {
		backendLintRunner = originalLintRunner
		backendOpenAPIRunner = originalOpenAPIRunner
		backendGoTestRunner = originalGoTestRunner
		backendGoBuildRunner = originalGoBuildRunner
		backendSmokeRunner = originalSmokeRunner
	}()

	var lintCalls int
	backendLintRunner = func(_ *cobra.Command, lintConfig string, testLintConfig string) error {
		lintCalls++
		if lintConfig != defaultBackendLintConfig || testLintConfig != defaultBackendTestLintConfig {
			t.Fatalf("unexpected lint configs: %s %s", lintConfig, testLintConfig)
		}
		return nil
	}
	backendOpenAPIRunner = func(_ *cobra.Command, _ string) error {
		t.Fatal("openapi runner should not be called during lint-only blocking stage")
		return nil
	}
	backendGoTestRunner = func(_ *cobra.Command, _ []string) error {
		t.Fatal("go test runner should not be called during lint-only blocking stage")
		return nil
	}
	backendGoBuildRunner = func(_ *cobra.Command) error {
		t.Fatal("go build runner should not be called during lint-only blocking stage")
		return nil
	}
	backendSmokeRunner = func(_ *cobra.Command, _ smokeValidateOptions) error {
		t.Fatal("smoke runner should not be called during lint-only blocking stage")
		return nil
	}

	if err := runValidateBackend(&cobra.Command{}, backendValidateOptions{
		stage:          "lint",
		lintConfig:     defaultBackendLintConfig,
		testLintConfig: defaultBackendTestLintConfig,
	}); err != nil {
		t.Fatalf("run validate backend lint stage: %v", err)
	}

	if lintCalls != 1 {
		t.Fatalf("expected exactly one blocking lint stage call, got %d", lintCalls)
	}
}

// TestRunValidateBackendBuildTestStage 验证 buildtest 阶段会先跑 go test，再构建 `./cmd/graft`。
func TestRunValidateBackendBuildTestStage(t *testing.T) {
	originalLintRunner := backendLintRunner
	originalOpenAPIRunner := backendOpenAPIRunner
	originalGoTestRunner := backendGoTestRunner
	originalGoBuildRunner := backendGoBuildRunner
	defer func() {
		backendLintRunner = originalLintRunner
		backendOpenAPIRunner = originalOpenAPIRunner
		backendGoTestRunner = originalGoTestRunner
		backendGoBuildRunner = originalGoBuildRunner
	}()

	var steps []string
	backendLintRunner = func(_ *cobra.Command, _ string, _ string) error {
		t.Fatal("lint runner should not be called during buildtest stage")
		return nil
	}
	backendOpenAPIRunner = func(_ *cobra.Command, _ string) error {
		t.Fatal("openapi runner should not be called during buildtest stage")
		return nil
	}
	backendGoTestRunner = func(_ *cobra.Command, targets []string) error {
		steps = append(steps, "test:"+strings.Join(targets, ","))
		return nil
	}
	backendGoBuildRunner = func(_ *cobra.Command) error {
		steps = append(steps, "build")
		return nil
	}

	err := runValidateBackend(&cobra.Command{}, backendValidateOptions{
		stage:       "buildtest",
		testTargets: []string{"./plugins/user", "./internal/httpx"},
	})
	if err != nil {
		t.Fatalf("run validate backend buildtest stage: %v", err)
	}

	expected := []string{"test:./plugins/user,./internal/httpx", "build"}
	if !reflect.DeepEqual(steps, expected) {
		t.Fatalf("expected %v, got %v", expected, steps)
	}
}

// TestRunValidateBackendFullStageWithSmoke 验证 full 阶段会按固定顺序串联 lint、test、build 与可选 smoke。
func TestRunValidateBackendFullStageWithSmoke(t *testing.T) {
	originalLintRunner := backendLintRunner
	originalOpenAPIRunner := backendOpenAPIRunner
	originalGoTestRunner := backendGoTestRunner
	originalGoBuildRunner := backendGoBuildRunner
	originalSmokeRunner := backendSmokeRunner
	defer func() {
		backendLintRunner = originalLintRunner
		backendOpenAPIRunner = originalOpenAPIRunner
		backendGoTestRunner = originalGoTestRunner
		backendGoBuildRunner = originalGoBuildRunner
		backendSmokeRunner = originalSmokeRunner
	}()

	var steps []string
	backendOpenAPIRunner = func(_ *cobra.Command, spec string) error {
		steps = append(steps, "openapi:"+spec)
		return nil
	}
	backendLintRunner = func(_ *cobra.Command, _ string, _ string) error {
		steps = append(steps, "lint")
		return nil
	}
	backendGoTestRunner = func(_ *cobra.Command, targets []string) error {
		steps = append(steps, "test:"+strings.Join(targets, ","))
		return nil
	}
	backendGoBuildRunner = func(_ *cobra.Command) error {
		steps = append(steps, "build")
		return nil
	}
	backendSmokeRunner = func(_ *cobra.Command, opts smokeValidateOptions) error {
		steps = append(steps, "smoke:"+opts.migrationDir+":"+opts.healthPath)
		return nil
	}

	err := runValidateBackend(&cobra.Command{}, backendValidateOptions{
		stage: "full",
		smoke: true,
	})
	if err != nil {
		t.Fatalf("run validate backend full stage: %v", err)
	}

	expected := []string{
		"openapi:" + defaultOpenAPIRootSpec,
		"lint",
		"test:./...",
		"build",
		"smoke:" + defaultMigrationDir + ":" + defaultSmokeHealthPath,
	}
	if !reflect.DeepEqual(steps, expected) {
		t.Fatalf("expected %v, got %v", expected, steps)
	}
}

// TestRunValidateBackendRejectsSmokeOutsideFull 验证 `--smoke` 只能附着在完整质量链之后。
func TestRunValidateBackendRejectsSmokeOutsideFull(t *testing.T) {
	err := runValidateBackend(&cobra.Command{}, backendValidateOptions{
		stage: "lint",
		smoke: true,
	})
	if err == nil {
		t.Fatal("expected backend validation error")
	}
	if !strings.Contains(err.Error(), "`--smoke` requires `--stage full`") {
		t.Fatalf("expected smoke stage restriction, got %v", err)
	}
}

// TestRunValidateBackendRejectsUnknownStage 验证未知 stage 会返回显式错误。
func TestRunValidateBackendRejectsUnknownStage(t *testing.T) {
	err := runValidateBackend(&cobra.Command{}, backendValidateOptions{
		stage: "unknown",
	})
	if err == nil {
		t.Fatal("expected backend validation error")
	}
	if !strings.Contains(err.Error(), "unsupported backend validation stage") {
		t.Fatalf("expected stage validation error, got %v", err)
	}
}

// TestRunValidateSmokeRunsMigrateBeforeServe 验证 smoke 验证会先执行迁移，
// 再等待健康检查成功，最后主动停止运行时。
func TestRunValidateSmokeRunsMigrateBeforeServe(t *testing.T) {
	originalMigrateRunner := smokeMigrateRunner
	originalServeRunner := smokeServeRunner
	originalLoadConfig := smokeLoadConfig
	originalHealthChecker := smokeHealthChecker
	defer func() {
		smokeMigrateRunner = originalMigrateRunner
		smokeServeRunner = originalServeRunner
		smokeLoadConfig = originalLoadConfig
		smokeHealthChecker = originalHealthChecker
	}()

	var (
		steps   []string
		stepsMu sync.Mutex
	)
	appendStep := func(step string) {
		stepsMu.Lock()
		defer stepsMu.Unlock()
		steps = append(steps, step)
	}
	stepsSnapshot := func() []string {
		stepsMu.Lock()
		defer stepsMu.Unlock()
		return append([]string(nil), steps...)
	}
	serveStarted := make(chan struct{})

	smokeMigrateRunner = func(_ *cobra.Command, migrationDir string) error {
		appendStep("migrate:" + migrationDir)
		return nil
	}
	smokeLoadConfig = func() (*config.Config, error) {
		return &config.Config{
			HTTP: config.HTTPConfig{Addr: ":18080"},
		}, nil
	}
	smokeServeRunner = func(cmd *cobra.Command, _ []string) error {
		appendStep("serve-start")
		close(serveStarted)
		<-cmd.Context().Done()
		appendStep("serve-stop")
		return nil
	}
	smokeHealthChecker = func(_ context.Context, probeURL string) error {
		<-serveStarted
		appendStep("health:" + probeURL)
		return nil
	}

	err := runValidateSmoke(&cobra.Command{}, smokeValidateOptions{
		migrationDir: defaultMigrationDir,
		healthPath:   defaultSmokeHealthPath,
		timeout:      time.Second,
	})
	if err != nil {
		t.Fatalf("run validate smoke: %v", err)
	}

	expected := []string{
		"migrate:" + defaultMigrationDir,
		"serve-start",
		"health:http://127.0.0.1:18080/healthz",
		"serve-stop",
	}
	if actual := stepsSnapshot(); !reflect.DeepEqual(actual, expected) {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
}

// TestRunValidateSmokeStopsAfterMigrationFailure 验证迁移失败时不会继续启动运行时。
func TestRunValidateSmokeStopsAfterMigrationFailure(t *testing.T) {
	originalMigrateRunner := smokeMigrateRunner
	originalServeRunner := smokeServeRunner
	defer func() {
		smokeMigrateRunner = originalMigrateRunner
		smokeServeRunner = originalServeRunner
	}()

	smokeMigrateRunner = func(_ *cobra.Command, _ string) error {
		return errors.New("migrate failed")
	}
	smokeServeRunner = func(_ *cobra.Command, _ []string) error {
		t.Fatal("serve runner should not be called")
		return nil
	}

	err := runValidateSmoke(&cobra.Command{}, smokeValidateOptions{
		migrationDir: defaultMigrationDir,
		healthPath:   defaultSmokeHealthPath,
		timeout:      time.Second,
	})
	if err == nil {
		t.Fatal("expected smoke validation error")
	}
	if !strings.Contains(err.Error(), "run smoke migrations") {
		t.Fatalf("expected migration context, got %v", err)
	}
}

// TestRunValidateSmokeReturnsServeFailure 验证运行时在健康检查前退出时会立刻返回服务错误。
func TestRunValidateSmokeReturnsServeFailure(t *testing.T) {
	originalMigrateRunner := smokeMigrateRunner
	originalServeRunner := smokeServeRunner
	originalLoadConfig := smokeLoadConfig
	originalHealthChecker := smokeHealthChecker
	defer func() {
		smokeMigrateRunner = originalMigrateRunner
		smokeServeRunner = originalServeRunner
		smokeLoadConfig = originalLoadConfig
		smokeHealthChecker = originalHealthChecker
	}()

	smokeMigrateRunner = func(_ *cobra.Command, _ string) error {
		return nil
	}
	smokeLoadConfig = func() (*config.Config, error) {
		return &config.Config{
			HTTP: config.HTTPConfig{Addr: ":18080"},
		}, nil
	}
	smokeServeRunner = func(_ *cobra.Command, _ []string) error {
		return errors.New("listen failed")
	}
	smokeHealthChecker = func(ctx context.Context, _ string) error {
		<-ctx.Done()
		return ctx.Err()
	}

	err := runValidateSmoke(&cobra.Command{}, smokeValidateOptions{
		migrationDir: defaultMigrationDir,
		healthPath:   defaultSmokeHealthPath,
		timeout:      time.Second,
	})
	if err == nil {
		t.Fatal("expected smoke validation error")
	}
	if !strings.Contains(err.Error(), "run smoke server") {
		t.Fatalf("expected serve context, got %v", err)
	}
}

// TestRunValidateSmokeReturnsHealthFailure 验证健康检查失败时会停止运行时并返回探测错误。
func TestRunValidateSmokeReturnsHealthFailure(t *testing.T) {
	originalMigrateRunner := smokeMigrateRunner
	originalServeRunner := smokeServeRunner
	originalLoadConfig := smokeLoadConfig
	originalHealthChecker := smokeHealthChecker
	defer func() {
		smokeMigrateRunner = originalMigrateRunner
		smokeServeRunner = originalServeRunner
		smokeLoadConfig = originalLoadConfig
		smokeHealthChecker = originalHealthChecker
	}()

	smokeMigrateRunner = func(_ *cobra.Command, _ string) error {
		return nil
	}
	smokeLoadConfig = func() (*config.Config, error) {
		return &config.Config{
			HTTP: config.HTTPConfig{Addr: ":18080"},
		}, nil
	}
	smokeServeRunner = func(cmd *cobra.Command, _ []string) error {
		<-cmd.Context().Done()
		return nil
	}
	smokeHealthChecker = func(_ context.Context, _ string) error {
		return errors.New("health failed")
	}

	err := runValidateSmoke(&cobra.Command{}, smokeValidateOptions{
		migrationDir: defaultMigrationDir,
		healthPath:   defaultSmokeHealthPath,
		timeout:      time.Second,
	})
	if err == nil {
		t.Fatal("expected smoke validation error")
	}
	if !strings.Contains(err.Error(), "wait for smoke health check") {
		t.Fatalf("expected health-check context, got %v", err)
	}
}

// TestBuildSmokeProbeURLUsesLoopbackForWildcard 验证 wildcard 监听地址会转换为本地可探测的 loopback URL。
func TestBuildSmokeProbeURLUsesLoopbackForWildcard(t *testing.T) {
	testCases := []struct {
		name     string
		addr     string
		path     string
		expected string
	}{
		{
			name:     "empty host",
			addr:     ":8080",
			path:     "/healthz",
			expected: "http://127.0.0.1:8080/healthz",
		},
		{
			name:     "ipv4 wildcard",
			addr:     "0.0.0.0:8080",
			path:     "healthz",
			expected: "http://127.0.0.1:8080/healthz",
		},
		{
			name:     "localhost",
			addr:     "127.0.0.1:8080",
			path:     "/healthz",
			expected: "http://127.0.0.1:8080/healthz",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual, err := buildSmokeProbeURL(testCase.addr, testCase.path)
			if err != nil {
				t.Fatalf("build smoke probe url: %v", err)
			}
			if actual != testCase.expected {
				t.Fatalf("expected %s, got %s", testCase.expected, actual)
			}
		})
	}
}

// TestNewRootCommandRegistersValidateCommands 验证根命令始终注册 `validate` 子命令树。
func TestNewRootCommandRegistersValidateCommands(t *testing.T) {
	command := NewRootCommand()

	foundBackend, _, err := command.Find([]string{"validate", "backend"})
	if err != nil {
		t.Fatalf("find validate backend command: %v", err)
	}
	if foundBackend == nil || foundBackend.Name() != "backend" {
		t.Fatalf("expected backend command, got %#v", foundBackend)
	}

	foundSmoke, _, err := command.Find([]string{"validate", "smoke"})
	if err != nil {
		t.Fatalf("find validate smoke command: %v", err)
	}
	if foundSmoke == nil || foundSmoke.Name() != "smoke" {
		t.Fatalf("expected smoke command, got %#v", foundSmoke)
	}
}
