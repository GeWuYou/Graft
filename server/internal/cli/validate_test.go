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

// TestResolveBackendModuleRootFromServerDir 验证在 `server` 目录运行时会直接识别模块根目录。
func TestResolveBackendModuleRootFromServerDir(t *testing.T) {
	originalGetwd := backendGetwd
	originalReadFile := backendReadFile
	defer func() {
		backendGetwd = originalGetwd
		backendReadFile = originalReadFile
	}()

	tempDir := t.TempDir()
	serverDir := filepath.Join(tempDir, "server")
	if err := os.MkdirAll(serverDir, 0o750); err != nil {
		t.Fatalf("mkdir server dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(serverDir, "go.mod"), []byte("module graft/server\n"), 0o600); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

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

	tempDir := t.TempDir()
	serverDir := filepath.Join(tempDir, "server")
	if err := os.MkdirAll(serverDir, 0o750); err != nil {
		t.Fatalf("mkdir server dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(serverDir, "go.mod"), []byte("module graft/server\n"), 0o600); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	backendGetwd = func() (string, error) {
		return tempDir, nil
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
	originalGoTestRunner := backendGoTestRunner
	originalGoBuildRunner := backendGoBuildRunner
	originalSmokeRunner := backendSmokeRunner
	defer func() {
		backendLintRunner = originalLintRunner
		backendGoTestRunner = originalGoTestRunner
		backendGoBuildRunner = originalGoBuildRunner
		backendSmokeRunner = originalSmokeRunner
	}()

	var steps []string
	backendLintRunner = func(_ *cobra.Command, lintConfig string, testLintConfig string) error {
		steps = append(steps, "lint:"+lintConfig+":"+testLintConfig)
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

// TestRunValidateBackendBuildTestStage 验证 buildtest 阶段会先跑 go test，再构建 `./cmd/graft`。
func TestRunValidateBackendBuildTestStage(t *testing.T) {
	originalLintRunner := backendLintRunner
	originalGoTestRunner := backendGoTestRunner
	originalGoBuildRunner := backendGoBuildRunner
	defer func() {
		backendLintRunner = originalLintRunner
		backendGoTestRunner = originalGoTestRunner
		backendGoBuildRunner = originalGoBuildRunner
	}()

	var steps []string
	backendLintRunner = func(_ *cobra.Command, _ string, _ string) error {
		t.Fatal("lint runner should not be called during buildtest stage")
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
	originalGoTestRunner := backendGoTestRunner
	originalGoBuildRunner := backendGoBuildRunner
	originalSmokeRunner := backendSmokeRunner
	defer func() {
		backendLintRunner = originalLintRunner
		backendGoTestRunner = originalGoTestRunner
		backendGoBuildRunner = originalGoBuildRunner
		backendSmokeRunner = originalSmokeRunner
	}()

	var steps []string
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
