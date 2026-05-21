package cli

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestRunDevRunsMigrateBeforeServe 验证开发编排命令会先执行迁移，再启动服务。
func TestRunDevRunsMigrateBeforeServe(t *testing.T) {
	originalMigrateRunner := devMigrateRunner
	originalServeRunner := devServeRunner
	defer func() {
		devMigrateRunner = originalMigrateRunner
		devServeRunner = originalServeRunner
	}()

	var steps []string
	devMigrateRunner = func(_ *cobra.Command, migrationDir string) error {
		steps = append(steps, "migrate:"+migrationDir)
		return nil
	}
	devServeRunner = func(_ *cobra.Command, _ []string) error {
		steps = append(steps, "serve")
		return nil
	}

	err := runDev(&cobra.Command{}, nil, devOptions{migrationDir: defaultMigrationDir})
	if err != nil {
		t.Fatalf("run dev: %v", err)
	}

	expected := []string{"migrate:" + defaultMigrationDir, "serve"}
	if !reflect.DeepEqual(steps, expected) {
		t.Fatalf("expected %v, got %v", expected, steps)
	}
}

// TestRunDevStopsAfterMigrationFailure 验证迁移失败时不会继续启动服务。
func TestRunDevStopsAfterMigrationFailure(t *testing.T) {
	originalMigrateRunner := devMigrateRunner
	originalServeRunner := devServeRunner
	defer func() {
		devMigrateRunner = originalMigrateRunner
		devServeRunner = originalServeRunner
	}()

	devMigrateRunner = func(_ *cobra.Command, _ string) error {
		return errors.New("migrate failed")
	}
	devServeRunner = func(_ *cobra.Command, _ []string) error {
		t.Fatal("serve runner should not be called")
		return nil
	}

	err := runDev(&cobra.Command{}, nil, devOptions{migrationDir: defaultMigrationDir})
	if err == nil {
		t.Fatal("expected dev command error")
	}
	if !strings.Contains(err.Error(), "run development migrations") {
		t.Fatalf("expected migration context, got %v", err)
	}
}

// TestRunDevAirInvokesAirRunner 验证 `graft dev air` 会调用 Air 热重载执行边界。
func TestRunDevAirInvokesAirRunner(t *testing.T) {
	originalAirRunner := devAirRunner
	defer func() {
		devAirRunner = originalAirRunner
	}()

	var gotConfig string
	devAirRunner = func(_ *cobra.Command, configPath string) error {
		gotConfig = configPath
		return nil
	}

	err := runDevAir(&cobra.Command{}, devAirOptions{configPath: ".air.toml"})
	if err != nil {
		t.Fatalf("run dev air: %v", err)
	}
	if gotConfig != ".air.toml" {
		t.Fatalf("expected config path .air.toml, got %q", gotConfig)
	}
}

// TestRunDevAirWrapsRunnerError 验证 Air 执行失败时会返回带上下文的错误。
func TestRunDevAirWrapsRunnerError(t *testing.T) {
	originalAirRunner := devAirRunner
	defer func() {
		devAirRunner = originalAirRunner
	}()

	devAirRunner = func(_ *cobra.Command, _ string) error {
		return errors.New("air failed")
	}

	err := runDevAir(&cobra.Command{}, devAirOptions{configPath: ".air.toml"})
	if err == nil {
		t.Fatal("expected dev air error")
	}
	if !strings.Contains(err.Error(), "start Air live reload") {
		t.Fatalf("expected air context, got %v", err)
	}
}

// TestServerAirConfigUsesEntrypointForServe 验证仓库内置的 Air 配置使用 entrypoint 数组
// 来表达 `graft serve`，避免把可执行文件与参数拼成一个错误路径。
func TestServerAirConfigUsesEntrypointForServe(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current test file path")
	}

	configPath := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", ".air.toml"))
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read %s: %v", configPath, err)
	}

	config := string(content)
	if !strings.Contains(config, `entrypoint = ["./tmp/graft", "serve"]`) {
		t.Fatalf("expected Air config to use entrypoint array for graft serve, got:\n%s", config)
	}
	if strings.Contains(config, `bin = "./tmp/graft serve"`) {
		t.Fatalf("legacy build.bin form must not be used because Air treats it as a single binary path:\n%s", config)
	}
}

// TestNewRootCommandRegistersDevCommand 验证根命令始终注册 `dev` 子命令。
func TestNewRootCommandRegistersDevCommand(t *testing.T) {
	command := NewRootCommand()

	found, _, err := command.Find([]string{"dev"})
	if err != nil {
		t.Fatalf("find dev command: %v", err)
	}
	if found == nil || found.Name() != "dev" {
		t.Fatalf("expected dev command, got %#v", found)
	}
}

// TestNewRootCommandRegistersDevResetAdminCommand 验证 `graft dev reset-admin` 子命令可发现。
func TestNewRootCommandRegistersDevResetAdminCommand(t *testing.T) {
	command := NewRootCommand()

	found, _, err := command.Find([]string{"dev", "reset-admin"})
	if err != nil {
		t.Fatalf("find dev reset-admin command: %v", err)
	}
	if found == nil || found.Name() != "reset-admin" {
		t.Fatalf("expected reset-admin command, got %#v", found)
	}
}

// TestNewRootCommandRegistersDevAirCommand 验证 `graft dev air` 子命令可发现。
func TestNewRootCommandRegistersDevAirCommand(t *testing.T) {
	command := NewRootCommand()

	found, _, err := command.Find([]string{"dev", "air"})
	if err != nil {
		t.Fatalf("find dev air command: %v", err)
	}
	if found == nil || found.Name() != "air" {
		t.Fatalf("expected air command, got %#v", found)
	}
}
