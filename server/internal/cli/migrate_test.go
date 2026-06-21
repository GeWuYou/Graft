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

	"graft/server/internal/moduleregistry"
)

type migrateTestHooks struct {
	getwd                 func() (string, error)
	lookPath              func(string) (string, error)
	commandContext        func(context.Context, string, ...string) *exec.Cmd
	stdin                 io.Reader
	registryMigrationDirs func() ([]string, error)
	readDir               func(string) ([]os.DirEntry, error)
	readFile              func(string) ([]byte, error)
	writeFile             func(string, []byte, os.FileMode) error
	mkdirTemp             func(string, string) (string, error)
	removeAll             func(string) error
}

func captureMigrateTestHooks() migrateTestHooks {
	return migrateTestHooks{
		getwd:                 migrateGetwd,
		lookPath:              migrateLookPath,
		commandContext:        migrateCommandContext,
		stdin:                 migrateStdin,
		registryMigrationDirs: migrateRegistryMigrationDirs,
		readDir:               migrateReadDir,
		readFile:              migrateReadFile,
		writeFile:             migrateWriteFile,
		mkdirTemp:             migrateMkdirTemp,
		removeAll:             migrateRemoveAll,
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
// 的模块迁移目录会被解析为 `server` 相对路径。
func TestResolveMigrationDirFindsServerRelativePathFromRepoRoot(t *testing.T) {
	root := t.TempDir()
	migrationDir := filepath.Join(root, "server", "modules", "user", "migrations")
	if err := os.MkdirAll(migrationDir, 0o750); err != nil {
		t.Fatalf("mkdir migration dir: %v", err)
	}

	resolved, err := resolveMigrationDir(root, "modules/user/migrations")
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
	migrationDir := filepath.Join(serverRoot, "modules", "user", "migrations")
	if err := os.MkdirAll(migrationDir, 0o750); err != nil {
		t.Fatalf("mkdir migration dir: %v", err)
	}

	resolved, err := resolveMigrationDir(serverRoot, "modules/user/migrations")
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

	_, err := resolveMigrationDir(root, "modules/user/migrations")
	if err == nil {
		t.Fatal("expected missing migration dir error")
	}
}

// TestResolveMigrationDirsUsesCompileTimeRegistry 验证默认迁移目录会先回到
// compile-time registry 读取 live owner-aligned 目录集合。
func TestResolveMigrationDirsUsesCompileTimeRegistry(t *testing.T) {
	originalRegistryMigrationDirs := migrateRegistryMigrationDirs
	originalReadDir := migrateReadDir
	defer func() {
		migrateRegistryMigrationDirs = originalRegistryMigrationDirs
		migrateReadDir = originalReadDir
	}()

	root := t.TempDir()
	coreDir := filepath.Join(root, "server", "internal", "httpx", "migrations")
	auditDir := filepath.Join(root, "server", "modules", "audit", "migrations")
	moduleDir := filepath.Join(root, "server", "modules", "user", "migrations")
	for _, dir := range []string{coreDir, auditDir, moduleDir} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	for _, dir := range []string{coreDir, auditDir, moduleDir} {
		if err := os.WriteFile(filepath.Join(dir, "atlas.sum"), []byte(filepath.Base(dir)), 0o600); err != nil {
			t.Fatalf("write atlas.sum in %s: %v", dir, err)
		}
	}

	migrateRegistryMigrationDirs = func() ([]string, error) {
		return []string{"internal/httpx/migrations", "modules/audit/migrations", "modules/user/migrations"}, nil
	}
	migrateReadDir = os.ReadDir

	resolved, err := resolveMigrationDirs(root, defaultMigrationDir)
	if err != nil {
		t.Fatalf("resolve migration dirs: %v", err)
	}

	expected := []string{coreDir, auditDir, moduleDir}
	if !reflect.DeepEqual(resolved, expected) {
		t.Fatalf("expected %v, got %v", expected, resolved)
	}
}

// TestResolveMigrationDirsSkipsRegistryDirsWithoutAtlasState 验证默认迁移目录会跳过
// 尚未形成 Atlas 状态的模块自有目录，避免空目录参与默认 apply 链路。
func TestResolveMigrationDirsSkipsRegistryDirsWithoutAtlasState(t *testing.T) {
	originalRegistryMigrationDirs := migrateRegistryMigrationDirs
	originalReadDir := migrateReadDir
	defer func() {
		migrateRegistryMigrationDirs = originalRegistryMigrationDirs
		migrateReadDir = originalReadDir
	}()

	root := t.TempDir()
	coreDir := filepath.Join(root, "server", "internal", "httpx", "migrations")
	auditDir := filepath.Join(root, "server", "modules", "audit", "migrations")
	moduleDir := filepath.Join(root, "server", "modules", "user", "migrations")
	for _, dir := range []string{coreDir, auditDir, moduleDir} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(coreDir, "atlas.sum"), []byte("httpx"), 0o600); err != nil {
		t.Fatalf("write atlas.sum: %v", err)
	}
	if err := os.WriteFile(filepath.Join(auditDir, "atlas.sum"), []byte("audit"), 0o600); err != nil {
		t.Fatalf("write atlas.sum: %v", err)
	}

	migrateRegistryMigrationDirs = func() ([]string, error) {
		return []string{"internal/httpx/migrations", "modules/audit/migrations", "modules/user/migrations"}, nil
	}
	migrateReadDir = os.ReadDir

	resolved, err := resolveMigrationDirs(root, defaultMigrationDir)
	if err != nil {
		t.Fatalf("resolve migration dirs: %v", err)
	}

	expected := []string{coreDir, auditDir}
	if !reflect.DeepEqual(resolved, expected) {
		t.Fatalf("expected %v, got %v", expected, resolved)
	}
}

// TestResolveMigrationDirsSkipsMissingRegistryDirs 验证默认 registry 链路会跳过
// 尚未创建的模块迁移目录，而不是让缺失目录阻断全部迁移。
func TestResolveMigrationDirsSkipsMissingRegistryDirs(t *testing.T) {
	originalRegistryMigrationDirs := migrateRegistryMigrationDirs
	originalReadDir := migrateReadDir
	defer func() {
		migrateRegistryMigrationDirs = originalRegistryMigrationDirs
		migrateReadDir = originalReadDir
	}()

	root := t.TempDir()
	coreDir := filepath.Join(root, "server", "internal", "httpx", "migrations")
	auditDir := filepath.Join(root, "server", "modules", "audit", "migrations")
	for _, dir := range []string{coreDir, auditDir} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(coreDir, "atlas.sum"), []byte("httpx"), 0o600); err != nil {
		t.Fatalf("write atlas.sum: %v", err)
	}
	if err := os.WriteFile(filepath.Join(auditDir, "atlas.sum"), []byte("audit"), 0o600); err != nil {
		t.Fatalf("write atlas.sum: %v", err)
	}

	migrateRegistryMigrationDirs = func() ([]string, error) {
		return []string{"internal/httpx/migrations", "modules/audit/migrations", "modules/user/migrations"}, nil
	}
	migrateReadDir = os.ReadDir

	resolved, err := resolveMigrationDirs(root, defaultMigrationDir)
	if err != nil {
		t.Fatalf("resolve migration dirs: %v", err)
	}

	expected := []string{coreDir, auditDir}
	if !reflect.DeepEqual(resolved, expected) {
		t.Fatalf("expected %v, got %v", expected, resolved)
	}
}

// TestDefaultMigrationRegistrySQLDirsHaveAtlasState guards against registering
// live migration SQL that the default chain silently skips because atlas.sum is
// missing.
func TestDefaultMigrationRegistrySQLDirsHaveAtlasState(t *testing.T) {
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working dir: %v", err)
	}

	dirs, err := moduleregistry.MigrationDirs()
	if err != nil {
		t.Fatalf("load migration dirs: %v", err)
	}

	for _, dir := range dirs {
		assertSQLMigrationDirHasAtlasState(t, workingDir, dir)
	}
}

func assertSQLMigrationDirHasAtlasState(t *testing.T, workingDir string, dir string) {
	t.Helper()

	absDir, err := resolveMigrationDir(workingDir, dir)
	if errors.Is(err, os.ErrNotExist) {
		return
	}
	if err != nil {
		t.Fatalf("resolve migration dir %s: %v", dir, err)
	}

	hasSQL, hasAtlasState := migrationDirState(t, absDir)
	if hasSQL && !hasAtlasState {
		t.Fatalf("migration dir %s has SQL files but no atlas.sum", dir)
	}
}

func migrationDirState(t *testing.T, absDir string) (bool, bool) {
	t.Helper()

	entries, err := os.ReadDir(absDir)
	if err != nil {
		t.Fatalf("read migration dir %s: %v", absDir, err)
	}

	hasSQL := false
	hasAtlasState := false
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		hasSQL = hasSQL || filepath.Ext(entry.Name()) == ".sql"
		hasAtlasState = hasAtlasState || entry.Name() == "atlas.sum"
	}

	return hasSQL, hasAtlasState
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
	coreDir := filepath.Join(root, "server", "internal", "httpx", "migrations")
	auditDir := filepath.Join(root, "server", "modules", "audit", "migrations")
	moduleDir := filepath.Join(root, "server", "modules", "user", "migrations")
	for _, dir := range []string{coreDir, auditDir, moduleDir} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	migrateRegistryMigrationDirs = func() ([]string, error) {
		return []string{"internal/httpx/migrations", "modules/audit/migrations", "modules/user/migrations"}, nil
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

// TestResolveMigrationDirsKeepsExplicitLiveDir 验证显式传入 live 迁移目录时，
// CLI 仍会直接解析该目录，而不会回退到默认 registry 链。
func TestResolveMigrationDirsKeepsExplicitLiveDir(t *testing.T) {
	originalRegistryMigrationDirs := migrateRegistryMigrationDirs
	defer func() {
		migrateRegistryMigrationDirs = originalRegistryMigrationDirs
	}()

	root := t.TempDir()
	liveDir := filepath.Join(root, "server", "modules", "user", "migrations")
	if err := os.MkdirAll(liveDir, 0o750); err != nil {
		t.Fatalf("mkdir %s: %v", liveDir, err)
	}

	migrateRegistryMigrationDirs = func() ([]string, error) {
		t.Fatal("explicit live dir should not consult registry")
		return nil, nil
	}

	resolved, err := resolveMigrationDirs(root, "modules/user/migrations")
	if err != nil {
		t.Fatalf("resolve migration dirs: %v", err)
	}

	expected := []string{liveDir}
	if !reflect.DeepEqual(resolved, expected) {
		t.Fatalf("expected %v, got %v", expected, resolved)
	}
}

// TestResolveMigrationDirsKeepsExplicitDirWithoutAtlasState 验证显式传入的迁移目录
// 仍按用户要求参与执行，而不是被默认链路的 Atlas 状态过滤逻辑跳过。
func TestResolveMigrationDirsKeepsExplicitDirWithoutAtlasState(t *testing.T) {
	root := t.TempDir()
	moduleDir := filepath.Join(root, "server", "modules", "user", "migrations")
	if err := os.MkdirAll(moduleDir, 0o750); err != nil {
		t.Fatalf("mkdir %s: %v", moduleDir, err)
	}

	resolved, err := resolveMigrationDirs(root, "modules/user/migrations")
	if err != nil {
		t.Fatalf("resolve migration dirs: %v", err)
	}

	expected := []string{moduleDir}
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
	migrationDir := filepath.Join(root, "server", "modules", "user", "migrations")
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
		return []string{"modules/user/migrations"}, nil
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

// TestRunMigrateUpSynthesizesDefaultChain 验证默认迁移路径会把 live owner-aligned
// 迁移目录合成为单一 Atlas 版本链，再对数据库执行一次 apply。
func TestRunMigrateUpSynthesizesDefaultChain(t *testing.T) {
	hooks := captureMigrateTestHooks()
	defer hooks.restore()

	root := t.TempDir()
	coreDir := filepath.Join(root, "server", "internal", "httpx", "migrations")
	auditDir := filepath.Join(root, "server", "modules", "audit", "migrations")
	rbacDir := filepath.Join(root, "server", "modules", "rbac", "migrations")
	userDir := filepath.Join(root, "server", "modules", "user", "migrations")
	dirs := []string{coreDir, auditDir, rbacDir, userDir}
	createMigrationFixture(t, dirs, map[string]string{
		filepath.Join(coreDir, "202605300001_access_log.sql"): "CREATE TABLE access_logs (id bigint);\n",
		filepath.Join(userDir, "202605190001_user.sql"):       "CREATE TABLE users (id bigint);\n",
		filepath.Join(rbacDir, "202605190002_rbac.sql"):       "CREATE TABLE roles (id bigint);\n",
		filepath.Join(auditDir, "202605190003_audit.sql"):     "CREATE TABLE audit_logs (id bigint);\n",
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
			"internal/httpx/migrations",
			"modules/user/migrations",
			"modules/rbac/migrations",
			"modules/audit/migrations",
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
		"202605300001_access_log.sql",
	})
}

func TestRunMigrateUpIncludesAtlasHashStderr(t *testing.T) {
	hooks := captureMigrateTestHooks()
	defer hooks.restore()

	root := t.TempDir()
	userDir := filepath.Join(root, "server", "modules", "user", "migrations")
	dirs := []string{userDir}
	createMigrationFixture(t, dirs, map[string]string{
		filepath.Join(userDir, "202605190001_user.sql"): "CREATE TABLE users (id bigint);\n",
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
		return []string{"modules/user/migrations"}, nil
	}
	useRealMigrateFileOps(os.RemoveAll)

	atlasInvocations := 0
	migrateCommandContext = func(_ context.Context, _ string, args ...string) *exec.Cmd {
		atlasInvocations++
		if len(args) >= 2 && args[0] == "migrate" && args[1] == "hash" {
			return exec.Command("sh", "-c", "printf 'atlas hash failed: malformed sql\\n' >&2; exit 1")
		}

		return exec.Command("true")
	}
	migrateStdin = strings.NewReader("")

	err := runMigrateUp(newSilentMigrateCommand(), migrateUpOptions{migrationDir: defaultMigrationDir})
	if err == nil {
		t.Fatal("expected atlas hash error")
	}
	if !strings.Contains(err.Error(), "hash synthesized default migration dir") {
		t.Fatalf("expected synthesized dir context, got %v", err)
	}
	if !strings.Contains(err.Error(), "atlas hash failed: malformed sql") {
		t.Fatalf("expected atlas stderr in error, got %v", err)
	}
	if atlasInvocations != 1 {
		t.Fatalf("expected hash failure to stop before apply, got %d atlas invocations", atlasInvocations)
	}
}

func TestRunMigrateUpRejectsDuplicateMigrationFilenames(t *testing.T) {
	assertDuplicateSynthesizedDefaultChainError(
		t,
		map[string]string{
			"user": "202605190001_shared.sql",
			"rbac": "202605190001_shared.sql",
		},
		"duplicate migration filename 202605190001_shared.sql",
		"synthesized default chain is invalid",
	)
}

func TestRunMigrateUpRejectsDuplicateMigrationVersions(t *testing.T) {
	assertDuplicateSynthesizedDefaultChainError(
		t,
		map[string]string{
			"user": "202605280001_user.sql",
			"rbac": "202605280001_rbac.sql",
		},
		"duplicate migration version 202605280001",
		"synthesized default chain has version conflicts",
	)
}

func assertDuplicateSynthesizedDefaultChainError(t *testing.T, filenames map[string]string, expectedErr string, atlasGuardMessage string) {
	t.Helper()

	hooks := captureMigrateTestHooks()
	defer hooks.restore()

	root := t.TempDir()
	userDir := filepath.Join(root, "server", "modules", "user", "migrations")
	rbacDir := filepath.Join(root, "server", "modules", "rbac", "migrations")
	dirs := []string{userDir, rbacDir}
	createMigrationFixture(t, dirs, map[string]string{
		filepath.Join(userDir, filenames["user"]): "SELECT 1;\n",
		filepath.Join(rbacDir, filenames["rbac"]): "SELECT 1;\n",
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
		return []string{"modules/user/migrations", "modules/rbac/migrations"}, nil
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
		t.Fatal("expected synthesized default chain validation error")
	}
	if !strings.Contains(err.Error(), expectedErr) {
		t.Fatalf("expected duplicate migration guidance, got %v", err)
	}
	if atlasInvoked {
		t.Fatal("atlas should not run when " + atlasGuardMessage)
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
