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
	originalReadDir := migrateReadDir
	defer func() {
		migrateRegistryMigrationDirs = originalRegistryMigrationDirs
		migrateReadDir = originalReadDir
	}()

	root := t.TempDir()
	coreDir := filepath.Join(root, "server", defaultMigrationDir)
	pluginDir := filepath.Join(root, "server", "plugins", "user", "migrations")
	for _, dir := range []string{coreDir, pluginDir} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	for _, dir := range []string{coreDir, pluginDir} {
		if err := os.WriteFile(filepath.Join(dir, "atlas.sum"), []byte(filepath.Base(dir)), 0o600); err != nil {
			t.Fatalf("write atlas.sum in %s: %v", dir, err)
		}
	}

	migrateRegistryMigrationDirs = func() ([]string, error) {
		return []string{defaultMigrationDir, "plugins/user/migrations"}, nil
	}
	migrateReadDir = os.ReadDir

	resolved, err := resolveMigrationDirs(root, defaultMigrationDir)
	if err != nil {
		t.Fatalf("resolve migration dirs: %v", err)
	}

	expected := []string{coreDir, pluginDir}
	if !reflect.DeepEqual(resolved, expected) {
		t.Fatalf("expected %v, got %v", expected, resolved)
	}
}

// TestResolveMigrationDirsSkipsRegistryDirsWithoutAtlasState 验证默认迁移目录会跳过
// 尚未形成 Atlas 状态的插件自有目录，避免空目录参与默认 apply 链路。
func TestResolveMigrationDirsSkipsRegistryDirsWithoutAtlasState(t *testing.T) {
	originalRegistryMigrationDirs := migrateRegistryMigrationDirs
	originalReadDir := migrateReadDir
	defer func() {
		migrateRegistryMigrationDirs = originalRegistryMigrationDirs
		migrateReadDir = originalReadDir
	}()

	root := t.TempDir()
	coreDir := filepath.Join(root, "server", defaultMigrationDir)
	pluginDir := filepath.Join(root, "server", "plugins", "user", "migrations")
	for _, dir := range []string{coreDir, pluginDir} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(coreDir, "atlas.sum"), []byte("core"), 0o600); err != nil {
		t.Fatalf("write atlas.sum: %v", err)
	}

	migrateRegistryMigrationDirs = func() ([]string, error) {
		return []string{defaultMigrationDir, "plugins/user/migrations"}, nil
	}
	migrateReadDir = os.ReadDir

	resolved, err := resolveMigrationDirs(root, defaultMigrationDir)
	if err != nil {
		t.Fatalf("resolve migration dirs: %v", err)
	}

	expected := []string{coreDir}
	if !reflect.DeepEqual(resolved, expected) {
		t.Fatalf("expected %v, got %v", expected, resolved)
	}
}

// TestResolveMigrationDirsRejectsRegistryWithoutAtlasState 验证默认 registry 目录
// 若全部缺少 Atlas 状态，会显式报错而不是静默跳过迁移。
func TestResolveMigrationDirsRejectsRegistryWithoutAtlasState(t *testing.T) {
	originalRegistryMigrationDirs := migrateRegistryMigrationDirs
	originalReadDir := migrateReadDir
	defer func() {
		migrateRegistryMigrationDirs = originalRegistryMigrationDirs
		migrateReadDir = originalReadDir
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
	migrateReadDir = os.ReadDir

	_, err := resolveMigrationDirs(root, defaultMigrationDir)
	if err == nil {
		t.Fatal("expected empty atlas-state registry error")
	}
	if !strings.Contains(err.Error(), "no migration directories with atlas state found") {
		t.Fatalf("expected atlas-state guidance, got %v", err)
	}
}

// TestResolveMigrationDirsKeepsExplicitDirWithoutAtlasState 验证显式传入的迁移目录
// 仍按用户要求参与执行，而不是被默认链路的 Atlas 状态过滤逻辑跳过。
func TestResolveMigrationDirsKeepsExplicitDirWithoutAtlasState(t *testing.T) {
	root := t.TempDir()
	pluginDir := filepath.Join(root, "server", "plugins", "user", "migrations")
	if err := os.MkdirAll(pluginDir, 0o750); err != nil {
		t.Fatalf("mkdir %s: %v", pluginDir, err)
	}

	resolved, err := resolveMigrationDirs(root, "plugins/user/migrations")
	if err != nil {
		t.Fatalf("resolve migration dirs: %v", err)
	}

	expected := []string{pluginDir}
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
	originalReadDir := migrateReadDir
	defer func() {
		migrateGetwd = originalGetwd
		migrateLookPath = originalLookPath
		migrateCommandContext = originalCommandContext
		migrateStdin = originalStdin
		migrateRegistryMigrationDirs = originalRegistryMigrationDirs
		migrateReadDir = originalReadDir
	}()

	root := t.TempDir()
	migrationDir := filepath.Join(root, "server", defaultMigrationDir)
	if err := os.MkdirAll(migrationDir, 0o750); err != nil {
		t.Fatalf("mkdir migration dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(migrationDir, "atlas.sum"), []byte("core"), 0o600); err != nil {
		t.Fatalf("write atlas.sum: %v", err)
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
	migrateReadDir = os.ReadDir

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
	originalReadDir := migrateReadDir
	defer func() {
		migrateGetwd = originalGetwd
		migrateLookPath = originalLookPath
		migrateCommandContext = originalCommandContext
		migrateStdin = originalStdin
		migrateRegistryMigrationDirs = originalRegistryMigrationDirs
		migrateReadDir = originalReadDir
	}()

	root := t.TempDir()
	coreDir := filepath.Join(root, "server", defaultMigrationDir)
	pluginDir := filepath.Join(root, "server", "plugins", "user", "migrations")
	for _, dir := range []string{coreDir, pluginDir} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	for _, dir := range []string{coreDir, pluginDir} {
		if err := os.WriteFile(filepath.Join(dir, "atlas.sum"), []byte(filepath.Base(dir)), 0o600); err != nil {
			t.Fatalf("write atlas.sum in %s: %v", dir, err)
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
	migrateReadDir = os.ReadDir

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
