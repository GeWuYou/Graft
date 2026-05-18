package cli

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestResolveMigrationDirFindsServerRelativePathFromRepoRoot 验证仓库根目录下
// 的默认迁移目录会被解析为 `server` 相对路径。
func TestResolveMigrationDirFindsServerRelativePathFromRepoRoot(t *testing.T) {
	root := t.TempDir()
	migrationDir := filepath.Join(root, "server", defaultMigrationDir)
	if err := os.MkdirAll(migrationDir, 0o750); err != nil {
		t.Fatalf("mkdir migration dir: %v", err)
	}

	resolved, err := resolveMigrationDir(root, defaultMigrationDir)
	if err != nil {
		t.Fatalf("resolve migration dir: %v", err)
	}

	if resolved != migrationDir {
		t.Fatalf("expected %s, got %s", migrationDir, resolved)
	}
}

// TestResolveMigrationDirFindsPathFromServerModuleRoot 验证迁移目录解析器也支持
// 以 `server` 模块根目录作为当前工作目录。
func TestResolveMigrationDirFindsPathFromServerModuleRoot(t *testing.T) {
	root := t.TempDir()
	serverRoot := filepath.Join(root, "server")
	migrationDir := filepath.Join(serverRoot, defaultMigrationDir)
	if err := os.MkdirAll(migrationDir, 0o750); err != nil {
		t.Fatalf("mkdir migration dir: %v", err)
	}

	resolved, err := resolveMigrationDir(serverRoot, defaultMigrationDir)
	if err != nil {
		t.Fatalf("resolve migration dir: %v", err)
	}

	if resolved != migrationDir {
		t.Fatalf("expected %s, got %s", migrationDir, resolved)
	}
}

// TestResolveMigrationDirRejectsMissingPath 验证当两个受支持的迁移目录都不
// 存在时，解析器会返回错误。
func TestResolveMigrationDirRejectsMissingPath(t *testing.T) {
	root := t.TempDir()

	_, err := resolveMigrationDir(root, defaultMigrationDir)
	if err == nil {
		t.Fatal("expected missing migration dir error")
	}
}

// TestResolveMigrationDirsUsesCompileTimeRegistry 验证默认迁移目录会先回到
// compile-time registry 读取目录集合。
func TestResolveMigrationDirsUsesCompileTimeRegistry(t *testing.T) {
	originalRegistryMigrationDirs := migrateRegistryMigrationDirs
	defer func() {
		migrateRegistryMigrationDirs = originalRegistryMigrationDirs
	}()

	root := t.TempDir()
	coreDir := filepath.Join(root, "server", defaultMigrationDir)
	pluginDir := filepath.Join(root, "server", "plugins", "user", "migrations")
	for _, dir := range []string{coreDir, pluginDir} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	migrateRegistryMigrationDirs = func() ([]string, error) {
		return []string{defaultMigrationDir, "plugins/user/migrations"}, nil
	}

	resolved, err := resolveMigrationDirs(root, defaultMigrationDir)
	if err != nil {
		t.Fatalf("resolve migration dirs: %v", err)
	}

	expected := []string{coreDir, pluginDir}
	if !reflect.DeepEqual(resolved, expected) {
		t.Fatalf("expected %v, got %v", expected, resolved)
	}
}

// TestFindAtlasCLIReportsDevGuidance 验证缺少 Atlas 时会返回可执行的开发提示。
func TestFindAtlasCLIReportsDevGuidance(t *testing.T) {
	originalLookPath := migrateLookPath
	defer func() {
		migrateLookPath = originalLookPath
	}()

	migrateLookPath = func(_ string) (string, error) {
		return "", errors.New("executable file not found")
	}

	_, err := findAtlasCLI()
	if err == nil {
		t.Fatal("expected atlas lookup error")
	}

	message := err.Error()
	if !strings.Contains(message, "graft dev") {
		t.Fatalf("expected dev guidance, got %q", message)
	}
	if !strings.Contains(message, "graft serve") {
		t.Fatalf("expected serve fallback guidance, got %q", message)
	}
}

// TestRunMigrateUpFallsBackToBackgroundContext 验证未通过 Execute 链路设置
// Cobra 上下文时，迁移命令仍会使用后台上下文而不是触发 nil-context 风险。
func TestRunMigrateUpFallsBackToBackgroundContext(t *testing.T) {
	originalGetwd := migrateGetwd
	originalLookPath := migrateLookPath
	originalCommandContext := migrateCommandContext
	originalStdin := migrateStdin
	originalRegistryMigrationDirs := migrateRegistryMigrationDirs
	defer func() {
		migrateGetwd = originalGetwd
		migrateLookPath = originalLookPath
		migrateCommandContext = originalCommandContext
		migrateStdin = originalStdin
		migrateRegistryMigrationDirs = originalRegistryMigrationDirs
	}()

	root := t.TempDir()
	migrationDir := filepath.Join(root, "server", defaultMigrationDir)
	if err := os.MkdirAll(migrationDir, 0o750); err != nil {
		t.Fatalf("mkdir migration dir: %v", err)
	}

	t.Setenv("GRAFT_DATABASE_URL", "postgres://user:pass@localhost:5432/graft?sslmode=disable")
	t.Setenv("GRAFT_REDIS_ADDR", "127.0.0.1:6379")
	t.Setenv("GRAFT_AUTH_JWT_SECRET", "test-signing-secret")

	migrateGetwd = func() (string, error) {
		return root, nil
	}
	migrateLookPath = func(_ string) (string, error) {
		return "/usr/bin/atlas", nil
	}
	migrateRegistryMigrationDirs = func() ([]string, error) {
		return []string{defaultMigrationDir}, nil
	}

	capturedCtx := context.Context(nil)
	migrateCommandContext = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		capturedCtx = ctx
		return exec.Command("true")
	}
	migrateStdin = strings.NewReader("")

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	if err := runMigrateUp(cmd, migrateUpOptions{migrationDir: defaultMigrationDir}); err != nil {
		t.Fatalf("run migrate up: %v", err)
	}

	if capturedCtx == nil {
		t.Fatal("expected migrate command to receive fallback context")
	}
}

// TestRunMigrateUpAppliesRegistryDirsSequentially 验证默认迁移路径会按 registry
// 声明顺序逐个执行 Atlas apply。
func TestRunMigrateUpAppliesRegistryDirsSequentially(t *testing.T) {
	originalGetwd := migrateGetwd
	originalLookPath := migrateLookPath
	originalCommandContext := migrateCommandContext
	originalStdin := migrateStdin
	originalRegistryMigrationDirs := migrateRegistryMigrationDirs
	defer func() {
		migrateGetwd = originalGetwd
		migrateLookPath = originalLookPath
		migrateCommandContext = originalCommandContext
		migrateStdin = originalStdin
		migrateRegistryMigrationDirs = originalRegistryMigrationDirs
	}()

	root := t.TempDir()
	coreDir := filepath.Join(root, "server", defaultMigrationDir)
	pluginDir := filepath.Join(root, "server", "plugins", "user", "migrations")
	for _, dir := range []string{coreDir, pluginDir} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	t.Setenv("GRAFT_DATABASE_URL", "postgres://user:pass@localhost:5432/graft?sslmode=disable")
	t.Setenv("GRAFT_REDIS_ADDR", "127.0.0.1:6379")
	t.Setenv("GRAFT_AUTH_JWT_SECRET", "test-signing-secret")

	migrateGetwd = func() (string, error) {
		return root, nil
	}
	migrateLookPath = func(_ string) (string, error) {
		return "/usr/bin/atlas", nil
	}
	migrateRegistryMigrationDirs = func() ([]string, error) {
		return []string{defaultMigrationDir, "plugins/user/migrations"}, nil
	}

	var gotDirs []string
	migrateCommandContext = func(_ context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.Command("true")
	}
	migrateCommandContext = func(_ context.Context, _ string, args ...string) *exec.Cmd {
		for index := 0; index < len(args)-1; index++ {
			if args[index] == "--dir" {
				gotDirs = append(gotDirs, strings.TrimPrefix(args[index+1], "file://"))
			}
		}
		return exec.Command("true")
	}
	migrateStdin = strings.NewReader("")

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	if err := runMigrateUp(cmd, migrateUpOptions{migrationDir: defaultMigrationDir}); err != nil {
		t.Fatalf("run migrate up: %v", err)
	}

	expected := []string{filepath.ToSlash(coreDir), filepath.ToSlash(pluginDir)}
	if !reflect.DeepEqual(gotDirs, expected) {
		t.Fatalf("expected atlas dirs %v, got %v", expected, gotDirs)
	}
}
