package cli

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	atlasmigrate "ariga.io/atlas/sql/migrate"
	"github.com/spf13/cobra"

	"graft/server/internal/moduleregistry"
)

type migrateTestHooks struct {
	getwd                      func() (string, error)
	registryMigrationDirs      func() ([]string, error)
	embeddedMigrationDirByPath func(string) (moduleregistry.EmbeddedMigrationDir, bool)
	readDir                    func(string) ([]os.DirEntry, error)
	openExecutor               func(string, atlasmigrate.Dir, atlasmigrate.Logger) (*atlasExecutorHandle, error)
}

func captureMigrateTestHooks() migrateTestHooks {
	return migrateTestHooks{
		getwd:                      migrateGetwd,
		registryMigrationDirs:      migrateRegistryMigrationDirs,
		embeddedMigrationDirByPath: migrateEmbeddedMigrationDirByPath,
		readDir:                    migrateReadDir,
		openExecutor:               migrateOpenExecutor,
	}
}

func (hooks migrateTestHooks) restore() {
	migrateGetwd = hooks.getwd
	migrateRegistryMigrationDirs = hooks.registryMigrationDirs
	migrateEmbeddedMigrationDirByPath = hooks.embeddedMigrationDirByPath
	migrateReadDir = hooks.readDir
	migrateOpenExecutor = hooks.openExecutor
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
		atlasDir, err := atlasmigrate.NewLocalDir(dir)
		if err != nil {
			t.Fatalf("open atlas dir %s: %v", dir, err)
		}
		sum, err := atlasDir.Checksum()
		if err != nil {
			t.Fatalf("compute atlas checksum in %s: %v", dir, err)
		}
		if err := atlasmigrate.WriteSumFile(atlasDir, sum); err != nil {
			t.Fatalf("write atlas.sum in %s: %v", dir, err)
		}
	}
}

func embeddedMigrationDir(t *testing.T, path string, files map[string]string) moduleregistry.EmbeddedMigrationDir {
	t.Helper()

	memDir := &atlasmigrate.MemDir{}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		if err := memDir.WriteFile(name, []byte(files[name])); err != nil {
			t.Fatalf("write embedded file %s: %v", name, err)
		}
	}
	sum, err := memDir.Checksum()
	if err != nil {
		t.Fatalf("compute embedded checksum: %v", err)
	}
	if err := atlasmigrate.WriteSumFile(memDir, sum); err != nil {
		t.Fatalf("write embedded atlas.sum: %v", err)
	}

	entries, err := memDir.Files()
	if err != nil {
		t.Fatalf("read embedded files: %v", err)
	}

	result := moduleregistry.EmbeddedMigrationDir{
		Path:  path,
		Files: make([]moduleregistry.EmbeddedMigrationFile, 0, len(entries)+1),
	}
	for _, file := range entries {
		result.Files = append(result.Files, moduleregistry.EmbeddedMigrationFile{
			Name:     file.Name(),
			Contents: append([]byte(nil), file.Bytes()...),
		})
	}
	sumFile, err := memDir.Open(atlasmigrate.HashFileName)
	if err != nil {
		t.Fatalf("open embedded atlas.sum: %v", err)
	}
	defer func() {
		_ = sumFile.Close()
	}()

	content, err := io.ReadAll(sumFile)
	if err != nil {
		t.Fatalf("read embedded atlas.sum: %v", err)
	}
	result.Files = append(result.Files, moduleregistry.EmbeddedMigrationFile{
		Name:     atlasmigrate.HashFileName,
		Contents: content,
	})

	return result
}

type fakeAtlasExecutor struct {
	executeN func(context.Context, int) error
}

func (f fakeAtlasExecutor) ExecuteN(ctx context.Context, n int) error {
	if f.executeN != nil {
		return f.executeN(ctx, n)
	}
	return nil
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

func TestResolveMigrationDirRejectsMissingPath(t *testing.T) {
	root := t.TempDir()

	_, err := resolveMigrationDir(root, "modules/user/migrations")
	if err == nil {
		t.Fatal("expected missing migration dir error")
	}
}

func TestResolveMigrationDirsUsesCompileTimeRegistry(t *testing.T) {
	hooks := captureMigrateTestHooks()
	defer hooks.restore()

	root := t.TempDir()
	coreDir := filepath.Join(root, "server", "internal", "httpx", "migrations")
	auditDir := filepath.Join(root, "server", "modules", "audit", "migrations")
	moduleDir := filepath.Join(root, "server", "modules", "user", "migrations")
	for _, dir := range []string{coreDir, auditDir, moduleDir} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	writeAtlasStateFiles(t, []string{coreDir, auditDir, moduleDir})

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

func TestResolveMigrationDirsSkipsRegistryDirsWithoutAtlasState(t *testing.T) {
	hooks := captureMigrateTestHooks()
	defer hooks.restore()

	root := t.TempDir()
	coreDir := filepath.Join(root, "server", "internal", "httpx", "migrations")
	auditDir := filepath.Join(root, "server", "modules", "audit", "migrations")
	moduleDir := filepath.Join(root, "server", "modules", "user", "migrations")
	for _, dir := range []string{coreDir, auditDir, moduleDir} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	writeAtlasStateFiles(t, []string{coreDir, auditDir})

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

func TestResolveMigrationDirsRejectsRegistryWithoutAtlasState(t *testing.T) {
	hooks := captureMigrateTestHooks()
	defer hooks.restore()

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

func TestResolveMigrationDirsKeepsExplicitLiveDir(t *testing.T) {
	hooks := captureMigrateTestHooks()
	defer hooks.restore()

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
		hasAtlasState = hasAtlasState || entry.Name() == atlasmigrate.HashFileName
	}

	return hasSQL, hasAtlasState
}

func TestBuildAtlasMigrationDirUsesEmbeddedDirForExplicitPath(t *testing.T) {
	hooks := captureMigrateTestHooks()
	defer hooks.restore()

	migrateEmbeddedMigrationDirByPath = func(path string) (moduleregistry.EmbeddedMigrationDir, bool) {
		if path != "modules/user/migrations" {
			return moduleregistry.EmbeddedMigrationDir{}, false
		}
		return embeddedMigrationDir(t, path, map[string]string{
			"202605190001_user.sql": "CREATE TABLE users (id bigint);\n",
		}), true
	}

	dir, err := buildAtlasMigrationDir(t.TempDir(), "modules/user/migrations")
	if err != nil {
		t.Fatalf("build atlas migration dir: %v", err)
	}

	files, err := dir.Files()
	if err != nil {
		t.Fatalf("read embedded migration dir files: %v", err)
	}
	if len(files) != 1 || files[0].Name() != "202605190001_user.sql" {
		t.Fatalf("unexpected files %#v", files)
	}
}

func TestBuildAtlasMigrationDirSynthesizesDefaultChainFromEmbeddedSources(t *testing.T) {
	hooks := captureMigrateTestHooks()
	defer hooks.restore()

	migrateRegistryMigrationDirs = func() ([]string, error) {
		return []string{"modules/user/migrations", "modules/rbac/migrations"}, nil
	}
	migrateEmbeddedMigrationDirByPath = func(path string) (moduleregistry.EmbeddedMigrationDir, bool) {
		switch path {
		case "modules/user/migrations":
			return embeddedMigrationDir(t, path, map[string]string{
				"202605190001_user.sql": "CREATE TABLE users (id bigint);\n",
			}), true
		case "modules/rbac/migrations":
			return embeddedMigrationDir(t, path, map[string]string{
				"202605190002_rbac.sql": "CREATE TABLE roles (id bigint);\n",
			}), true
		default:
			return moduleregistry.EmbeddedMigrationDir{}, false
		}
	}

	dir, err := buildAtlasMigrationDir(t.TempDir(), defaultMigrationDir)
	if err != nil {
		t.Fatalf("build default atlas migration dir: %v", err)
	}

	files, err := dir.Files()
	if err != nil {
		t.Fatalf("read synthesized files: %v", err)
	}

	names := make([]string, 0, len(files))
	for _, file := range files {
		names = append(names, file.Name())
	}
	slices.Sort(names)
	expected := []string{"202605190001_user.sql", "202605190002_rbac.sql"}
	if !reflect.DeepEqual(names, expected) {
		t.Fatalf("expected %v, got %v", expected, names)
	}

	if err := atlasmigrate.Validate(dir); err != nil {
		t.Fatalf("validate synthesized dir: %v", err)
	}
}

func TestBuildAtlasMigrationDirRejectsDuplicateMigrationFilename(t *testing.T) {
	hooks := captureMigrateTestHooks()
	defer hooks.restore()

	migrateRegistryMigrationDirs = func() ([]string, error) {
		return []string{"modules/user/migrations", "modules/rbac/migrations"}, nil
	}
	migrateEmbeddedMigrationDirByPath = func(path string) (moduleregistry.EmbeddedMigrationDir, bool) {
		return embeddedMigrationDir(t, path, map[string]string{
			"202605190001_shared.sql": "SELECT 1;\n",
		}), true
	}

	_, err := buildAtlasMigrationDir(t.TempDir(), defaultMigrationDir)
	if err == nil {
		t.Fatal("expected duplicate filename error")
	}
	if !strings.Contains(err.Error(), "duplicate migration filename 202605190001_shared.sql") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildAtlasMigrationDirRejectsDuplicateMigrationVersion(t *testing.T) {
	hooks := captureMigrateTestHooks()
	defer hooks.restore()

	migrateRegistryMigrationDirs = func() ([]string, error) {
		return []string{"modules/user/migrations", "modules/rbac/migrations"}, nil
	}
	migrateEmbeddedMigrationDirByPath = func(path string) (moduleregistry.EmbeddedMigrationDir, bool) {
		switch path {
		case "modules/user/migrations":
			return embeddedMigrationDir(t, path, map[string]string{
				"202605280001_user.sql": "SELECT 1;\n",
			}), true
		case "modules/rbac/migrations":
			return embeddedMigrationDir(t, path, map[string]string{
				"202605280001_rbac.sql": "SELECT 1;\n",
			}), true
		default:
			return moduleregistry.EmbeddedMigrationDir{}, false
		}
	}

	_, err := buildAtlasMigrationDir(t.TempDir(), defaultMigrationDir)
	if err == nil {
		t.Fatal("expected duplicate version error")
	}
	if !strings.Contains(err.Error(), "duplicate migration version 202605280001") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunMigrateUpFallsBackToBackgroundContext(t *testing.T) {
	hooks := captureMigrateTestHooks()
	defer hooks.restore()

	root := t.TempDir()
	migrationDir := filepath.Join(root, "server", "modules", "user", "migrations")
	createMigrationFixture(t, []string{migrationDir}, map[string]string{
		filepath.Join(migrationDir, "202605190001_user.sql"):   "CREATE TABLE users (id bigint);\n",
		filepath.Join(migrationDir, atlasmigrate.HashFileName): "h1:test\n202605190001_user.sql h1:file\n",
	})

	setMigrateCommandTestEnv(t)

	migrateGetwd = func() (string, error) {
		return root, nil
	}

	capturedCtx := context.Context(nil)
	migrateOpenExecutor = func(_ string, _ atlasmigrate.Dir, _ atlasmigrate.Logger) (*atlasExecutorHandle, error) {
		return &atlasExecutorHandle{
			executor: fakeAtlasExecutor{
				executeN: func(ctx context.Context, _ int) error {
					capturedCtx = ctx
					return nil
				},
			},
		}, nil
	}

	cmd := &cobra.Command{}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	if err := runMigrateUp(cmd, migrateUpOptions{migrationDir: "modules/user/migrations"}); err != nil {
		t.Fatalf("run migrate up: %v", err)
	}

	if capturedCtx == nil {
		t.Fatal("expected migrate command to receive fallback context")
	}
}

func TestRunMigrateUpExecutesDefaultChain(t *testing.T) {
	hooks := captureMigrateTestHooks()
	defer hooks.restore()

	setMigrateCommandTestEnv(t)

	migrateRegistryMigrationDirs = func() ([]string, error) {
		return []string{"modules/user/migrations"}, nil
	}
	migrateEmbeddedMigrationDirByPath = func(path string) (moduleregistry.EmbeddedMigrationDir, bool) {
		return embeddedMigrationDir(t, path, map[string]string{
			"202605190001_user.sql": "CREATE TABLE users (id bigint);\n",
		}), true
	}

	executed := false
	migrateOpenExecutor = func(databaseURL string, dir atlasmigrate.Dir, _ atlasmigrate.Logger) (*atlasExecutorHandle, error) {
		if databaseURL == "" {
			t.Fatal("expected database URL")
		}
		files, err := dir.Files()
		if err != nil {
			t.Fatalf("read files: %v", err)
		}
		if len(files) != 1 || files[0].Name() != "202605190001_user.sql" {
			t.Fatalf("unexpected files %#v", files)
		}
		return &atlasExecutorHandle{
			executor: fakeAtlasExecutor{
				executeN: func(_ context.Context, n int) error {
					executed = true
					if n != 0 {
						t.Fatalf("expected ExecuteN(0), got %d", n)
					}
					return nil
				},
			},
		}, nil
	}

	if err := runMigrateUp(newSilentMigrateCommand(), migrateUpOptions{migrationDir: defaultMigrationDir, workingDir: t.TempDir()}); err != nil {
		t.Fatalf("run migrate up: %v", err)
	}
	if !executed {
		t.Fatal("expected executor to run")
	}
}

func TestRunMigrateUpTreatsNoPendingAsSuccess(t *testing.T) {
	hooks := captureMigrateTestHooks()
	defer hooks.restore()

	setMigrateCommandTestEnv(t)
	migrateEmbeddedMigrationDirByPath = func(path string) (moduleregistry.EmbeddedMigrationDir, bool) {
		return embeddedMigrationDir(t, path, map[string]string{
			"202605190001_user.sql": "CREATE TABLE users (id bigint);\n",
		}), true
	}
	migrateOpenExecutor = func(_ string, _ atlasmigrate.Dir, _ atlasmigrate.Logger) (*atlasExecutorHandle, error) {
		return &atlasExecutorHandle{
			executor: fakeAtlasExecutor{
				executeN: func(context.Context, int) error {
					return atlasmigrate.ErrNoPendingFiles
				},
			},
		}, nil
	}

	if err := runMigrateUp(newSilentMigrateCommand(), migrateUpOptions{migrationDir: "modules/user/migrations", workingDir: t.TempDir()}); err != nil {
		t.Fatalf("expected no-pending path to succeed, got %v", err)
	}
}

func TestRunMigrateUpPropagatesExecutorOpenError(t *testing.T) {
	hooks := captureMigrateTestHooks()
	defer hooks.restore()

	setMigrateCommandTestEnv(t)
	migrateEmbeddedMigrationDirByPath = func(path string) (moduleregistry.EmbeddedMigrationDir, bool) {
		return embeddedMigrationDir(t, path, map[string]string{
			"202605190001_user.sql": "CREATE TABLE users (id bigint);\n",
		}), true
	}
	migrateOpenExecutor = func(_ string, _ atlasmigrate.Dir, _ atlasmigrate.Logger) (*atlasExecutorHandle, error) {
		return nil, errors.New("open atlas executor failed")
	}

	err := runMigrateUp(newSilentMigrateCommand(), migrateUpOptions{migrationDir: "modules/user/migrations", workingDir: t.TempDir()})
	if err == nil {
		t.Fatal("expected executor open error")
	}
	if !strings.Contains(err.Error(), "open atlas executor failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}
