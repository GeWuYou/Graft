package cli

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"graft/server/internal/pluginregistry"
)

type migrateTestHooks struct {
	getwd                func() (string, error)
	lookPath             func(string) (string, error)
	commandContext       func(context.Context, string, ...string) *exec.Cmd
	stdin                io.Reader
	registryMigrationDirs func() ([]string, error)
	readDir              func(string) ([]os.DirEntry, error)
	readFile             func(string) ([]byte, error)
	writeFile            func(string, []byte, os.FileMode) error
	mkdirTemp            func(string, string) (string, error)
	removeAll            func(string) error
}

func captureMigrateTestHooks() migrateTestHooks {
	return migrateTestHooks{
		getwd:                migrateGetwd,
		lookPath:             migrateLookPath,
		commandContext:       migrateCommandContext,
		stdin:                migrateStdin,
		registryMigrationDirs: migrateRegistryMigrationDirs,
		readDir:              migrateReadDir,
		readFile:             migrateReadFile,
		writeFile:            migrateWriteFile,
		mkdirTemp:            migrateMkdirTemp,
		removeAll:            migrateRemoveAll,
	}
}

func (hooks migrateTestHooks) restore() {
	migrateGetwd = hooks.getwd
	migrateLookPath = hooks.lookPath
	migrateCommandContext = hooks.commandContext
	migrateStdin = hooks.stdin
	migrateRegistryMigrationDirs = hooks.registryMigrationDirs
	migrateReadDir = hooks.readDir
	migrateReadFile = hooks.readFile
	migrateWriteFile = hooks.writeFile
	migrateMkdirTemp = hooks.mkdirTemp
	migrateRemoveAll = hooks.removeAll
}

func setMigrateCommandTestEnv(t *testing.T) {
	t.Helper()
	t.Setenv("GRAFT_DATABASE_URL", "postgres://user:pass@localhost:5432/graft?sslmode=disable")
	t.Setenv("GRAFT_REDIS_ADDR", "127.0.0.1:6379")
	t.Setenv("GRAFT_AUTH_JWT_SECRET", "test-signing-secret")
}

func newSilentMigrateCommand() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	return cmd
}

func createMigrationFixture(t *testing.T, dirs []string, files map[string]string) {
	t.Helper()

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
}

func writeAtlasStateFiles(t *testing.T, dirs []string) {
	t.Helper()

	for _, dir := range dirs {
		if err := os.WriteFile(filepath.Join(dir, "atlas.sum"), []byte(filepath.Base(dir)), 0o600); err != nil {
			t.Fatalf("write atlas.sum in %s: %v", dir, err)
		}
	}
}

func useRealMigrateFileOps(removeAll func(string) error) {
	migrateReadDir = os.ReadDir
	migrateReadFile = os.ReadFile
	migrateWriteFile = os.WriteFile
	migrateMkdirTemp = os.MkdirTemp
	migrateRemoveAll = removeAll
}

func assertDefaultChainAtlasCommands(t *testing.T, gotArgs [][]string) string {
	t.Helper()

	if len(gotArgs) != 2 {
		t.Fatalf("expected hash + apply atlas commands, got %d", len(gotArgs))
	}
	if !reflect.DeepEqual(gotArgs[0][:2], []string{"migrate", "hash"}) {
		t.Fatalf("expected first atlas command to hash, got %v", gotArgs[0])
	}
	if !reflect.DeepEqual(gotArgs[1][:2], []string{"migrate", "apply"}) {
		t.Fatalf("expected second atlas command to apply, got %v", gotArgs[1])
	}

	synthDir := atlasDirArgument(t, gotArgs[0])
	if synthDir == "" {
		t.Fatalf("expected synthesized dir in hash args, got %v", gotArgs[0])
	}
	if applyDir := atlasDirArgument(t, gotArgs[1]); applyDir != synthDir {
		t.Fatalf("expected apply dir %s to match hash dir, got %s", synthDir, applyDir)
	}

	return synthDir
}

func assertSynthesizedMigrationFiles(t *testing.T, synthDir string, expectedNames []string) {
	t.Helper()

	entries, err := os.ReadDir(synthDir)
	if err != nil {
		t.Fatalf("read synthesized dir: %v", err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		names = append(names, entry.Name())
	}
	slices.Sort(names)

	if !reflect.DeepEqual(names, expectedNames) {
		t.Fatalf("expected synthesized files %v, got %v", expectedNames, names)
	}
}

// TestResolveMigrationDirFindsServerRelativePathFromRepoRoot 验证仓库根目录下
// 的默认迁移目录会被解析为 `server` 相对路径。
func TestResolveMigrationDirFindsServerRelativePathFromRepoRoot(t *testing.T) {
	root := t.TempDir()
	migrationDir := filepath.Join(root, "server", pluginregistry.HistoricalSharedMigrationDir)
	if err := os.MkdirAll(migrationDir, 0o750); err != nil {
		t.Fatalf("mkdir migration dir: %v", err)
	}

	resolved, err := resolveMigrationDir(root, pluginregistry.HistoricalSharedMigrationDir)
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
	migrationDir := filepath.Join(serverRoot, pluginregistry.HistoricalSharedMigrationDir)
	if err := os.MkdirAll(migrationDir, 0o750); err != nil {
		t.Fatalf("mkdir migration dir: %v", err)
	}

	resolved, err := resolveMigrationDir(serverRoot, pluginregistry.HistoricalSharedMigrationDir)
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

	_, err := resolveMigrationDir(root, pluginregistry.HistoricalSharedMigrationDir)
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
	auditDir := filepath.Join(root, "server", "plugins", "audit", "migrations")
	pluginDir := filepath.Join(root, "server", "plugins", "user", "migrations")
	for _, dir := range []string{auditDir, pluginDir} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	for _, dir := range []string{auditDir, pluginDir} {
		if err := os.WriteFile(filepath.Join(dir, "atlas.sum"), []byte(filepath.Base(dir)), 0o600); err != nil {
			t.Fatalf("write atlas.sum in %s: %v", dir, err)
		}
	}

	migrateRegistryMigrationDirs = func() ([]string, error) {
		return []string{"plugins/audit/migrations", "plugins/user/migrations"}, nil
	}
	migrateReadDir = os.ReadDir

	resolved, err := resolveMigrationDirs(root, defaultMigrationDir)
	if err != nil {
		t.Fatalf("resolve migration dirs: %v", err)
	}

	expected := []string{auditDir, pluginDir}
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
	auditDir := filepath.Join(root, "server", "plugins", "audit", "migrations")
	pluginDir := filepath.Join(root, "server", "plugins", "user", "migrations")
	for _, dir := range []string{auditDir, pluginDir} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(auditDir, "atlas.sum"), []byte("audit"), 0o600); err != nil {
		t.Fatalf("write atlas.sum: %v", err)
	}

	migrateRegistryMigrationDirs = func() ([]string, error) {
		return []string{"plugins/audit/migrations", "plugins/user/migrations"}, nil
	}
	migrateReadDir = os.ReadDir

	resolved, err := resolveMigrationDirs(root, defaultMigrationDir)
	if err != nil {
		t.Fatalf("resolve migration dirs: %v", err)
	}

	expected := []string{auditDir}
	if !reflect.DeepEqual(resolved, expected) {
		t.Fatalf("expected %v, got %v", expected, resolved)
	}
}

// TestResolveMigrationDirsSkipsMissingRegistryDirs 验证默认 registry 链路会跳过
// 尚未创建的插件迁移目录，而不是让缺失目录阻断全部迁移。
func TestResolveMigrationDirsSkipsMissingRegistryDirs(t *testing.T) {
	originalRegistryMigrationDirs := migrateRegistryMigrationDirs
	originalReadDir := migrateReadDir
	defer func() {
		migrateRegistryMigrationDirs = originalRegistryMigrationDirs
		migrateReadDir = originalReadDir
	}()

	root := t.TempDir()
	auditDir := filepath.Join(root, "server", "plugins", "audit", "migrations")
	if err := os.MkdirAll(auditDir, 0o750); err != nil {
		t.Fatalf("mkdir %s: %v", auditDir, err)
	}
	if err := os.WriteFile(filepath.Join(auditDir, "atlas.sum"), []byte("audit"), 0o600); err != nil {
		t.Fatalf("write atlas.sum: %v", err)
	}

	migrateRegistryMigrationDirs = func() ([]string, error) {
		return []string{"plugins/audit/migrations", "plugins/user/migrations"}, nil
	}
	migrateReadDir = os.ReadDir

	resolved, err := resolveMigrationDirs(root, defaultMigrationDir)
	if err != nil {
		t.Fatalf("resolve migration dirs: %v", err)
	}

	expected := []string{auditDir}
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
	auditDir := filepath.Join(root, "server", "plugins", "audit", "migrations")
	pluginDir := filepath.Join(root, "server", "plugins", "user", "migrations")
	for _, dir := range []string{auditDir, pluginDir} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	migrateRegistryMigrationDirs = func() ([]string, error) {
		return []string{"plugins/audit/migrations", "plugins/user/migrations"}, nil
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

// TestResolveMigrationDirsKeepsExplicitHistoricalSharedDir 验证显式传入历史共享目录时，
// CLI 仍会直接解析该目录，而不会回退到默认 registry 链。
func TestResolveMigrationDirsKeepsExplicitHistoricalSharedDir(t *testing.T) {
	originalRegistryMigrationDirs := migrateRegistryMigrationDirs
	defer func() {
		migrateRegistryMigrationDirs = originalRegistryMigrationDirs
	}()

	root := t.TempDir()
	historicalDir := filepath.Join(root, "server", pluginregistry.HistoricalSharedMigrationDir)
	if err := os.MkdirAll(historicalDir, 0o750); err != nil {
		t.Fatalf("mkdir %s: %v", historicalDir, err)
	}

	migrateRegistryMigrationDirs = func() ([]string, error) {
		t.Fatal("explicit historical dir should not consult registry")
		return nil, nil
	}

	resolved, err := resolveMigrationDirs(root, pluginregistry.HistoricalSharedMigrationDir)
	if err != nil {
		t.Fatalf("resolve migration dirs: %v", err)
	}

	expected := []string{historicalDir}
	if !reflect.DeepEqual(resolved, expected) {
		t.Fatalf("expected %v, got %v", expected, resolved)
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
	originalReadFile := migrateReadFile
	originalWriteFile := migrateWriteFile
	originalMkdirTemp := migrateMkdirTemp
	originalRemoveAll := migrateRemoveAll
	defer func() {
		migrateGetwd = originalGetwd
		migrateLookPath = originalLookPath
		migrateCommandContext = originalCommandContext
		migrateStdin = originalStdin
		migrateRegistryMigrationDirs = originalRegistryMigrationDirs
		migrateReadDir = originalReadDir
		migrateReadFile = originalReadFile
		migrateWriteFile = originalWriteFile
		migrateMkdirTemp = originalMkdirTemp
		migrateRemoveAll = originalRemoveAll
	}()

	root := t.TempDir()
	migrationDir := filepath.Join(root, "server", "plugins", "user", "migrations")
	if err := os.MkdirAll(migrationDir, 0o750); err != nil {
		t.Fatalf("mkdir migration dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(migrationDir, "atlas.sum"), []byte("core"), 0o600); err != nil {
		t.Fatalf("write atlas.sum: %v", err)
	}
	if err := os.WriteFile(filepath.Join(migrationDir, "202605190001_user.sql"), []byte("CREATE TABLE users (id bigint);\n"), 0o600); err != nil {
		t.Fatalf("write migration sql: %v", err)
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
		return []string{"plugins/user/migrations"}, nil
	}
	migrateReadDir = os.ReadDir
	migrateReadFile = os.ReadFile
	migrateWriteFile = os.WriteFile
	migrateMkdirTemp = os.MkdirTemp
	migrateRemoveAll = func(string) error { return nil }

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

// TestRunMigrateUpSynthesizesDefaultChain 验证默认迁移路径会把 plugin-owned
// 迁移目录合成为单一 Atlas 版本链，再对数据库执行一次 apply。
func TestRunMigrateUpSynthesizesDefaultChain(t *testing.T) {
	hooks := captureMigrateTestHooks()
	defer hooks.restore()

	root := t.TempDir()
	auditDir := filepath.Join(root, "server", "plugins", "audit", "migrations")
	rbacDir := filepath.Join(root, "server", "plugins", "rbac", "migrations")
	userDir := filepath.Join(root, "server", "plugins", "user", "migrations")
	dirs := []string{auditDir, rbacDir, userDir}
	createMigrationFixture(t, dirs, map[string]string{
		filepath.Join(userDir, "202605190001_user.sql"):   "CREATE TABLE users (id bigint);\n",
		filepath.Join(rbacDir, "202605190002_rbac.sql"):   "CREATE TABLE roles (id bigint);\n",
		filepath.Join(auditDir, "202605190003_audit.sql"): "CREATE TABLE audit_logs (id bigint);\n",
	})
	writeAtlasStateFiles(t, dirs)

	setMigrateCommandTestEnv(t)

	migrateGetwd = func() (string, error) {
		return root, nil
	}
	migrateLookPath = func(_ string) (string, error) {
		return "/usr/bin/atlas", nil
	}
	migrateRegistryMigrationDirs = func() ([]string, error) {
		return []string{
			"plugins/user/migrations",
			"plugins/rbac/migrations",
			"plugins/audit/migrations",
		}, nil
	}
	useRealMigrateFileOps(func(string) error { return nil })

	var gotArgs [][]string
	migrateCommandContext = func(_ context.Context, _ string, args ...string) *exec.Cmd {
		gotArgs = append(gotArgs, append([]string(nil), args...))
		return exec.Command("true")
	}
	migrateStdin = strings.NewReader("")

	if err := runMigrateUp(newSilentMigrateCommand(), migrateUpOptions{migrationDir: defaultMigrationDir}); err != nil {
		t.Fatalf("run migrate up: %v", err)
	}

	synthDir := assertDefaultChainAtlasCommands(t, gotArgs)
	assertSynthesizedMigrationFiles(t, synthDir, []string{
		"202605190001_user.sql",
		"202605190002_rbac.sql",
		"202605190003_audit.sql",
	})
}

func TestRunMigrateUpRejectsDuplicateMigrationFilenames(t *testing.T) {
	hooks := captureMigrateTestHooks()
	defer hooks.restore()

	root := t.TempDir()
	userDir := filepath.Join(root, "server", "plugins", "user", "migrations")
	rbacDir := filepath.Join(root, "server", "plugins", "rbac", "migrations")
	dirs := []string{userDir, rbacDir}
	createMigrationFixture(t, dirs, map[string]string{
		filepath.Join(userDir, "202605190001_shared.sql"): "SELECT 1;\n",
		filepath.Join(rbacDir, "202605190001_shared.sql"): "SELECT 1;\n",
	})
	writeAtlasStateFiles(t, dirs)

	setMigrateCommandTestEnv(t)

	migrateGetwd = func() (string, error) {
		return root, nil
	}
	migrateLookPath = func(_ string) (string, error) {
		return "/usr/bin/atlas", nil
	}
	migrateRegistryMigrationDirs = func() ([]string, error) {
		return []string{"plugins/user/migrations", "plugins/rbac/migrations"}, nil
	}
	useRealMigrateFileOps(os.RemoveAll)

	atlasInvoked := false
	migrateCommandContext = func(_ context.Context, _ string, _ ...string) *exec.Cmd {
		atlasInvoked = true
		return exec.Command("true")
	}
	migrateStdin = strings.NewReader("")

	err := runMigrateUp(newSilentMigrateCommand(), migrateUpOptions{migrationDir: defaultMigrationDir})
	if err == nil {
		t.Fatal("expected duplicate filename error")
	}
	if !strings.Contains(err.Error(), "duplicate migration filename 202605190001_shared.sql") {
		t.Fatalf("expected duplicate filename guidance, got %v", err)
	}
	if atlasInvoked {
		t.Fatal("atlas should not run when synthesized default chain is invalid")
	}
}

func atlasDirArgument(t *testing.T, args []string) string {
	t.Helper()

	for index := 0; index < len(args)-1; index++ {
		if args[index] == "--dir" {
			return strings.TrimPrefix(args[index+1], "file://")
		}
	}

	return ""
}
