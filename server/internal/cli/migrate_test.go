package cli

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	atlasmigrate "ariga.io/atlas/sql/migrate"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"

	"graft/server/internal/moduleregistry"
)

type migrateTestHooks struct {
	getwd                      func() (string, error)
	registryMigrationDirs      func() ([]string, error)
	embeddedMigrationDirByPath func(string) (moduleregistry.EmbeddedMigrationDir, bool)
	readDir                    func(string) ([]os.DirEntry, error)
	openExecutor               func(string, atlasmigrate.Dir, atlasmigrate.Logger, bool) (*atlasExecutorHandle, error)
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

func openTestAtlasRevisionStore(t *testing.T) (*atlasRevisionStore, *sql.DB) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "atlas-revisions.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	store := newAtlasRevisionStore(db)
	if _, err := db.Exec(`
		CREATE TABLE atlas_schema_revisions (
			version TEXT PRIMARY KEY,
			description TEXT NOT NULL DEFAULT '',
			type INTEGER NOT NULL DEFAULT 0,
			applied INTEGER NOT NULL DEFAULT 0,
			total INTEGER NOT NULL DEFAULT 0,
			executed_at TIMESTAMP NOT NULL,
			execution_time INTEGER NOT NULL DEFAULT 0,
			error TEXT NOT NULL DEFAULT '',
			error_stmt TEXT NOT NULL DEFAULT '',
			hash TEXT NOT NULL DEFAULT '',
			partial_hashes BLOB NULL,
			operator_version TEXT NOT NULL DEFAULT ''
		)
	`); err != nil {
		t.Fatalf("create revision table: %v", err)
	}
	store.initOnce.Do(func() {})

	return store, db
}

func openEmptyTestAtlasRevisionStore(t *testing.T) (*atlasRevisionStore, *sql.DB) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "atlas-revisions-empty.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	return newAtlasRevisionStore(db), db
}

func requireEqualStoredRevision(t *testing.T, expected, actual *atlasmigrate.Revision) {
	t.Helper()
	if actual == nil {
		t.Fatal("expected revision, got nil")
	}

	if expected.Version != actual.Version {
		t.Fatalf("expected version %q, got %q", expected.Version, actual.Version)
	}
	if expected.Description != actual.Description {
		t.Fatalf("expected description %q, got %q", expected.Description, actual.Description)
	}
	if expected.Type != actual.Type {
		t.Fatalf("expected type %v, got %v", expected.Type, actual.Type)
	}
	if expected.Applied != actual.Applied {
		t.Fatalf("expected applied %d, got %d", expected.Applied, actual.Applied)
	}
	if expected.Total != actual.Total {
		t.Fatalf("expected total %d, got %d", expected.Total, actual.Total)
	}
	if !expected.ExecutedAt.Equal(actual.ExecutedAt) {
		t.Fatalf("expected executed_at %s, got %s", expected.ExecutedAt, actual.ExecutedAt)
	}
	if expected.ExecutionTime != actual.ExecutionTime {
		t.Fatalf("expected execution_time %s, got %s", expected.ExecutionTime, actual.ExecutionTime)
	}
	if expected.Error != actual.Error {
		t.Fatalf("expected error %q, got %q", expected.Error, actual.Error)
	}
	if expected.ErrorStmt != actual.ErrorStmt {
		t.Fatalf("expected error_stmt %q, got %q", expected.ErrorStmt, actual.ErrorStmt)
	}
	if expected.Hash != actual.Hash {
		t.Fatalf("expected hash %q, got %q", expected.Hash, actual.Hash)
	}
	if !reflect.DeepEqual(expected.PartialHashes, actual.PartialHashes) {
		t.Fatalf("expected partial_hashes %v, got %v", expected.PartialHashes, actual.PartialHashes)
	}
	if expected.OperatorVersion != actual.OperatorVersion {
		t.Fatalf("expected operator_version %q, got %q", expected.OperatorVersion, actual.OperatorVersion)
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

func TestAtlasRevisionStoreRoundTripAndUpdate(t *testing.T) {
	store, _ := openTestAtlasRevisionStore(t)
	ctx := context.Background()

	initial := &atlasmigrate.Revision{
		Version:         "202606220001",
		Description:     "create users",
		Type:            atlasmigrate.RevisionTypeExecute | atlasmigrate.RevisionTypeResolved,
		Applied:         2,
		Total:           3,
		ExecutedAt:      time.Unix(1719043200, 123456789).UTC(),
		ExecutionTime:   1875 * time.Millisecond,
		Error:           "statement failed",
		ErrorStmt:       "ALTER TABLE users ADD COLUMN email text",
		Hash:            "hash-v1",
		PartialHashes:   []string{"stmt-1", "stmt-2"},
		OperatorVersion: "graft",
	}
	if err := store.WriteRevision(ctx, initial); err != nil {
		t.Fatalf("write initial revision: %v", err)
	}

	stored, err := store.ReadRevision(ctx, initial.Version)
	if err != nil {
		t.Fatalf("read initial revision: %v", err)
	}
	requireEqualStoredRevision(t, initial, stored)

	updated := &atlasmigrate.Revision{
		Version:         initial.Version,
		Description:     "create users finalized",
		Type:            atlasmigrate.RevisionTypeExecute,
		Applied:         3,
		Total:           3,
		ExecutedAt:      initial.ExecutedAt.Add(2 * time.Minute),
		ExecutionTime:   2500 * time.Millisecond,
		Error:           "",
		ErrorStmt:       "",
		Hash:            "hash-v2",
		PartialHashes:   nil,
		OperatorVersion: "graft-operator",
	}
	if err := store.WriteRevision(ctx, updated); err != nil {
		t.Fatalf("update revision: %v", err)
	}

	reloaded, err := store.ReadRevision(ctx, updated.Version)
	if err != nil {
		t.Fatalf("read updated revision: %v", err)
	}
	requireEqualStoredRevision(t, updated, reloaded)

	revisions, err := store.ReadRevisions(ctx)
	if err != nil {
		t.Fatalf("read revisions: %v", err)
	}
	if len(revisions) != 1 {
		t.Fatalf("expected 1 revision, got %d", len(revisions))
	}
	requireEqualStoredRevision(t, updated, revisions[0])
}

func TestAtlasRevisionStoreReadRevisionMissing(t *testing.T) {
	store, _ := openTestAtlasRevisionStore(t)

	_, err := store.ReadRevision(context.Background(), "missing")
	if !errors.Is(err, atlasmigrate.ErrRevisionNotExist) {
		t.Fatalf("expected ErrRevisionNotExist, got %v", err)
	}
}

func TestAtlasRevisionStoreReadRevisionsWithoutTableReturnsEmpty(t *testing.T) {
	store, db := openEmptyTestAtlasRevisionStore(t)

	revisions, err := store.ReadRevisions(context.Background())
	if err != nil {
		t.Fatalf("read revisions without table: %v", err)
	}
	if len(revisions) != 0 {
		t.Fatalf("expected no revisions, got %d", len(revisions))
	}

	var count int
	if err := db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = 'atlas_schema_revisions'`).Scan(&count); err != nil {
		t.Fatalf("query sqlite schema: %v", err)
	}
	if count != 0 {
		t.Fatal("expected read revisions not to create atlas_schema_revisions")
	}
}

func TestAtlasRevisionStoreReadRevisionWithoutTableReturnsNotExist(t *testing.T) {
	store, db := openEmptyTestAtlasRevisionStore(t)

	_, err := store.ReadRevision(context.Background(), "missing")
	if !errors.Is(err, atlasmigrate.ErrRevisionNotExist) {
		t.Fatalf("expected ErrRevisionNotExist, got %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = 'atlas_schema_revisions'`).Scan(&count); err != nil {
		t.Fatalf("query sqlite schema: %v", err)
	}
	if count != 0 {
		t.Fatal("expected read revision not to create atlas_schema_revisions")
	}
}

func TestAtlasRevisionStoreDeleteRevision(t *testing.T) {
	store, _ := openTestAtlasRevisionStore(t)
	ctx := context.Background()

	revision := &atlasmigrate.Revision{
		Version:       "202606220002",
		Description:   "delete me",
		Type:          atlasmigrate.RevisionTypeBaseline,
		ExecutedAt:    time.Unix(1719046800, 0).UTC(),
		ExecutionTime: 10 * time.Millisecond,
	}
	if err := store.WriteRevision(ctx, revision); err != nil {
		t.Fatalf("write revision: %v", err)
	}

	if err := store.DeleteRevision(ctx, revision.Version); err != nil {
		t.Fatalf("delete revision: %v", err)
	}

	_, err := store.ReadRevision(ctx, revision.Version)
	if !errors.Is(err, atlasmigrate.ErrRevisionNotExist) {
		t.Fatalf("expected ErrRevisionNotExist after delete, got %v", err)
	}
}

func TestAtlasRevisionStoreDeleteRevisionWithoutTableIsNoOp(t *testing.T) {
	store, db := openEmptyTestAtlasRevisionStore(t)

	if err := store.DeleteRevision(context.Background(), "missing"); err != nil {
		t.Fatalf("delete revision without table: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = 'atlas_schema_revisions'`).Scan(&count); err != nil {
		t.Fatalf("query sqlite schema: %v", err)
	}
	if count != 0 {
		t.Fatal("expected delete revision not to create atlas_schema_revisions")
	}
}

func TestAtlasRevisionStoreWriteRevisionRejectsNil(t *testing.T) {
	store, _ := openTestAtlasRevisionStore(t)

	err := store.WriteRevision(context.Background(), nil)
	if err == nil {
		t.Fatal("expected nil revision error")
	}
	if !strings.Contains(err.Error(), "revision is required") {
		t.Fatalf("expected nil revision guidance, got %v", err)
	}
}

func TestAtlasRevisionStoreReadRevisionsOrdersByVersion(t *testing.T) {
	store, _ := openTestAtlasRevisionStore(t)
	ctx := context.Background()

	versions := []string{"202606220010", "202606220002", "202606220005"}
	for _, version := range versions {
		if err := store.WriteRevision(ctx, &atlasmigrate.Revision{
			Version:       version,
			Description:   version,
			Type:          atlasmigrate.RevisionTypeExecute,
			ExecutedAt:    time.Unix(1719043200, 0).UTC(),
			ExecutionTime: time.Duration(len(version)) * time.Millisecond,
		}); err != nil {
			t.Fatalf("write revision %s: %v", version, err)
		}
	}

	revisions, err := store.ReadRevisions(ctx)
	if err != nil {
		t.Fatalf("read revisions: %v", err)
	}

	got := make([]string, 0, len(revisions))
	for _, revision := range revisions {
		got = append(got, revision.Version)
	}

	expected := []string{"202606220002", "202606220005", "202606220010"}
	if !reflect.DeepEqual(expected, got) {
		t.Fatalf("expected ordered versions %v, got %v", expected, got)
	}
}

func TestAtlasRevisionStoreEnsureTableCreatesExpectedColumns(t *testing.T) {
	expectedFragments := []string{
		"version VARCHAR(255) PRIMARY KEY",
		"description TEXT NOT NULL DEFAULT ''",
		"type BIGINT NOT NULL DEFAULT 0",
		"applied BIGINT NOT NULL DEFAULT 0",
		"total BIGINT NOT NULL DEFAULT 0",
		"executed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()",
		"execution_time BIGINT NOT NULL DEFAULT 0",
		"error TEXT NOT NULL DEFAULT ''",
		"error_stmt TEXT NOT NULL DEFAULT ''",
		"hash TEXT NOT NULL DEFAULT ''",
		"partial_hashes JSONB NULL",
		"operator_version TEXT NOT NULL DEFAULT ''",
	}
	for _, fragment := range expectedFragments {
		if !strings.Contains(atlasRevisionStoreCreateTableSQL, fragment) {
			t.Fatalf("expected ensureTable SQL to contain %q", fragment)
		}
	}
}

func TestAtlasRevisionStoreWriteRevisionStoresNilPartialHashesAsNull(t *testing.T) {
	store, db := openTestAtlasRevisionStore(t)
	ctx := context.Background()

	revision := &atlasmigrate.Revision{
		Version:       "202606220003",
		Description:   "nil partial hashes",
		Type:          atlasmigrate.RevisionTypeExecute,
		ExecutedAt:    time.Unix(1719046800, 0).UTC(),
		ExecutionTime: 25 * time.Millisecond,
	}
	if err := store.WriteRevision(ctx, revision); err != nil {
		t.Fatalf("write revision: %v", err)
	}

	var raw sql.NullString
	if err := db.QueryRowContext(ctx, `SELECT partial_hashes FROM atlas_schema_revisions WHERE version = ?`, revision.Version).Scan(&raw); err != nil {
		t.Fatalf("read partial_hashes: %v", err)
	}
	if raw.Valid {
		t.Fatalf("expected NULL partial_hashes, got %q", raw.String)
	}
}

func TestAtlasRevisionStoreWriteRevisionStoresPartialHashesAsJSONArray(t *testing.T) {
	store, db := openTestAtlasRevisionStore(t)
	ctx := context.Background()

	revision := &atlasmigrate.Revision{
		Version:       "202606220004",
		Description:   "partial hashes",
		Type:          atlasmigrate.RevisionTypeExecute,
		ExecutedAt:    time.Unix(1719046801, 0).UTC(),
		ExecutionTime: 25 * time.Millisecond,
		PartialHashes: []string{"stmt-a", "stmt-b"},
	}
	if err := store.WriteRevision(ctx, revision); err != nil {
		t.Fatalf("write revision: %v", err)
	}

	var raw string
	if err := db.QueryRowContext(ctx, `SELECT partial_hashes FROM atlas_schema_revisions WHERE version = ?`, revision.Version).Scan(&raw); err != nil {
		t.Fatalf("read partial_hashes: %v", err)
	}

	var decoded []string
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		t.Fatalf("decode stored partial_hashes: %v", err)
	}
	if !reflect.DeepEqual(revision.PartialHashes, decoded) {
		t.Fatalf("expected partial_hashes %v, got %v", revision.PartialHashes, decoded)
	}
}

func TestScanAtlasRevisionRejectsInvalidPartialHashes(t *testing.T) {
	_, err := scanAtlasRevision(func(dest ...any) error {
		*dest[0].(*string) = "202606220005"
		*dest[1].(*string) = "broken partial hashes"
		*dest[2].(*int64) = 0
		*dest[3].(*int) = 0
		*dest[4].(*int) = 0
		*dest[5].(*time.Time) = time.Unix(1719046802, 0).UTC()
		*dest[6].(*int64) = int64((5 * time.Millisecond).Nanoseconds())
		*dest[7].(*string) = ""
		*dest[8].(*string) = ""
		*dest[9].(*string) = ""
		*dest[10].(*[]byte) = []byte(`{"not":"an-array"}`)
		*dest[11].(*string) = "graft"
		return nil
	})
	if err == nil {
		t.Fatal("expected invalid partial hashes error")
	}
	if !strings.Contains(err.Error(), "decode partial hashes for revision 202606220005") {
		t.Fatalf("expected partial hash decode guidance, got %v", err)
	}
}

func TestScanAtlasRevisionRejectsNegativeExecutionTime(t *testing.T) {
	revision, err := scanAtlasRevision(func(dest ...any) error {
		*dest[0].(*string) = "202606220006"
		*dest[1].(*string) = "negative execution time"
		*dest[2].(*int64) = 0
		*dest[3].(*int) = 0
		*dest[4].(*int) = 0
		*dest[5].(*time.Time) = time.Unix(1719046803, 0).UTC()
		*dest[6].(*int64) = -1
		*dest[7].(*string) = "failed"
		*dest[8].(*string) = "SELECT 1"
		*dest[9].(*string) = "hash"
		*dest[10].(*[]byte) = nil
		*dest[11].(*string) = "graft"
		return nil
	})
	if err != nil {
		t.Fatalf("expected negative duration to round-trip as stored data, got %v", err)
	}
	if revision.ExecutionTime != -1 {
		t.Fatalf("expected execution time -1ns, got %s", revision.ExecutionTime)
	}
	if revision.Error != "failed" || revision.ErrorStmt != "SELECT 1" {
		t.Fatalf("expected error fields to round-trip, got %q / %q", revision.Error, revision.ErrorStmt)
	}
}

func TestRevisionTypeEncodingRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		value atlasmigrate.RevisionType
	}{
		{
			name:  "baseline",
			value: atlasmigrate.RevisionTypeBaseline,
		},
		{
			name:  "execute",
			value: atlasmigrate.RevisionTypeExecute,
		},
		{
			name:  "resolved",
			value: atlasmigrate.RevisionTypeResolved,
		},
		{
			name:  "combined",
			value: atlasmigrate.RevisionTypeExecute | atlasmigrate.RevisionTypeResolved,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := revisionTypeToInt64(tt.value)
			if err != nil {
				t.Fatalf("encode revision type: %v", err)
			}

			decoded, err := revisionTypeFromInt64(encoded)
			if err != nil {
				t.Fatalf("decode revision type: %v", err)
			}

			if decoded != tt.value {
				t.Fatalf("expected type %v, got %v", tt.value, decoded)
			}
		})
	}
}

func TestRevisionTypeFromInt64RejectsNegative(t *testing.T) {
	_, err := revisionTypeFromInt64(-1)
	if err == nil {
		t.Fatal("expected negative revision type error")
	}
	if !strings.Contains(err.Error(), "cannot be negative") {
		t.Fatalf("expected negative revision type guidance, got %v", err)
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

func TestResolveMigrationDirsUsesExplicitExternalPath(t *testing.T) {
	root := t.TempDir()
	externalDir := filepath.Join(root, "tmp-migrations")
	if err := os.MkdirAll(externalDir, 0o750); err != nil {
		t.Fatalf("mkdir %s: %v", externalDir, err)
	}

	resolved, err := resolveMigrationDirs(root, "file:tmp-migrations")
	if err != nil {
		t.Fatalf("resolve migration dirs: %v", err)
	}

	expected := []string{externalDir}
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
		assertSQLMigrationDirHasValidAtlasState(t, workingDir, dir)
	}
}

func TestEmbeddedMigrationRegistryDirsHaveValidAtlasState(t *testing.T) {
	for _, embedded := range moduleregistry.EmbeddedMigrationDirs() {
		source, found, err := loadEmbeddedMigrationDirSource(embedded.Path)
		if err != nil {
			t.Fatalf("load embedded migration dir %s: %v", embedded.Path, err)
		}
		if !found {
			t.Fatalf("embedded migration dir %s not found", embedded.Path)
		}
		if !source.hasAtlasState {
			continue
		}
		if err := atlasmigrate.Validate(source.dir); err != nil {
			t.Fatalf("validate embedded migration dir %s: %v", embedded.Path, err)
		}
	}
}

func assertSQLMigrationDirHasValidAtlasState(t *testing.T, workingDir string, dir string) {
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
	if !hasAtlasState {
		return
	}

	atlasDir, err := atlasmigrate.NewLocalDir(absDir)
	if err != nil {
		t.Fatalf("open atlas dir %s: %v", absDir, err)
	}
	if err := atlasmigrate.Validate(atlasDir); err != nil {
		t.Fatalf("validate atlas dir %s: %v", dir, err)
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

func TestBuildAtlasMigrationDirRejectsRepoOwnedSelectorWithoutEmbeddedAssets(t *testing.T) {
	hooks := captureMigrateTestHooks()
	defer hooks.restore()

	root := t.TempDir()
	repoDir := filepath.Join(root, "server", "modules", "user", "migrations")
	createMigrationFixture(t, []string{repoDir}, map[string]string{
		filepath.Join(repoDir, "202605190001_user.sql"): "CREATE TABLE users (id bigint);\n",
	})
	writeAtlasStateFiles(t, []string{repoDir})

	migrateEmbeddedMigrationDirByPath = func(string) (moduleregistry.EmbeddedMigrationDir, bool) {
		return moduleregistry.EmbeddedMigrationDir{}, false
	}

	_, err := buildAtlasMigrationDir(root, "modules/user/migrations")
	if err == nil {
		t.Fatal("expected missing embedded assets error")
	}
	if !strings.Contains(err.Error(), "compile-time embedded migration dir \"modules/user/migrations\" is not available") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildAtlasMigrationDirUsesExplicitExternalPath(t *testing.T) {
	root := t.TempDir()
	externalDir := filepath.Join(root, "tmp-migrations")
	createMigrationFixture(t, []string{externalDir}, map[string]string{
		filepath.Join(externalDir, "202605190001_user.sql"):   "CREATE TABLE users (id bigint);\n",
		filepath.Join(externalDir, atlasmigrate.HashFileName): "h1:test\n202605190001_user.sql h1:file\n",
	})

	dir, err := buildAtlasMigrationDir(root, "file:tmp-migrations")
	if err != nil {
		t.Fatalf("build external atlas migration dir: %v", err)
	}

	files, err := dir.Files()
	if err != nil {
		t.Fatalf("read external migration dir files: %v", err)
	}
	if len(files) != 1 || files[0].Name() != "202605190001_user.sql" {
		t.Fatalf("unexpected files %#v", files)
	}
}

func TestBuildAtlasMigrationDirRejectsImplicitExternalPath(t *testing.T) {
	_, err := buildAtlasMigrationDir(t.TempDir(), "./tmp-migrations")
	if err == nil {
		t.Fatal("expected explicit external path error")
	}
	if !strings.Contains(err.Error(), "must use explicit file: prefix") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildAtlasMigrationDirRejectsServerPrefixedRepoOwnedSelector(t *testing.T) {
	_, err := buildAtlasMigrationDir(t.TempDir(), "server/modules/user/migrations")
	if err == nil {
		t.Fatal("expected server-prefixed selector error")
	}
	if !strings.Contains(err.Error(), "must use owner-aligned path without \"server/\"") {
		t.Fatalf("unexpected error: %v", err)
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

func TestBuildAtlasMigrationDirSynthesizesDefaultChainWithoutCopiedAtlasSumFiles(t *testing.T) {
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

	sumFile, err := dir.Open(atlasmigrate.HashFileName)
	if err != nil {
		t.Fatalf("open synthesized atlas.sum: %v", err)
	}
	defer func() {
		_ = sumFile.Close()
	}()

	sumContent, err := io.ReadAll(sumFile)
	if err != nil {
		t.Fatalf("read synthesized atlas.sum: %v", err)
	}
	if len(sumContent) == 0 {
		t.Fatal("expected synthesized atlas.sum content")
	}
	if strings.Contains(string(sumContent), "h1:test") {
		t.Fatalf("expected synthesized atlas.sum to be recomputed, got raw embedded content %q", string(sumContent))
	}

	sumFiles, err := dir.Files()
	if err != nil {
		t.Fatalf("read synthesized migration files: %v", err)
	}
	if len(sumFiles) != 2 {
		t.Fatalf("expected synthesized SQL migration files only, got %#v", sumFiles)
	}

	synthesizedSum, err := dir.Checksum()
	if err != nil {
		t.Fatalf("compute synthesized checksum: %v", err)
	}
	expectedSum, err := synthesizedSum.MarshalText()
	if err != nil {
		t.Fatalf("marshal synthesized checksum: %v", err)
	}
	if string(sumContent) != string(expectedSum) {
		t.Fatalf("expected synthesized atlas.sum %q, got %q", expectedSum, string(sumContent))
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
	migrateOpenExecutor = func(_ string, _ atlasmigrate.Dir, _ atlasmigrate.Logger, _ bool) (*atlasExecutorHandle, error) {
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
	migrateOpenExecutor = func(databaseURL string, dir atlasmigrate.Dir, _ atlasmigrate.Logger, allowDirty bool) (*atlasExecutorHandle, error) {
		if databaseURL == "" {
			t.Fatal("expected database URL")
		}
		if allowDirty {
			t.Fatal("default migrate up should not allow dirty")
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
	migrateOpenExecutor = func(_ string, _ atlasmigrate.Dir, _ atlasmigrate.Logger, _ bool) (*atlasExecutorHandle, error) {
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
	migrateOpenExecutor = func(_ string, _ atlasmigrate.Dir, _ atlasmigrate.Logger, _ bool) (*atlasExecutorHandle, error) {
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

func TestRunMigrateValidateUsesEmbeddedDefaultChain(t *testing.T) {
	hooks := captureMigrateTestHooks()
	defer hooks.restore()

	migrateRegistryMigrationDirs = func() ([]string, error) {
		return []string{"modules/user/migrations"}, nil
	}
	migrateEmbeddedMigrationDirByPath = func(path string) (moduleregistry.EmbeddedMigrationDir, bool) {
		if path != "modules/user/migrations" {
			return moduleregistry.EmbeddedMigrationDir{}, false
		}
		return embeddedMigrationDir(t, path, map[string]string{
			"202605190001_user.sql": "CREATE TABLE users (id bigint);\n",
		}), true
	}
	migrateOpenExecutor = func(string, atlasmigrate.Dir, atlasmigrate.Logger, bool) (*atlasExecutorHandle, error) {
		t.Fatal("migrate validate must not open an executor")
		return nil, nil
	}

	if err := runMigrateValidate(migrateResolveOptions{
		migrationDir: defaultMigrationDir,
		workingDir:   t.TempDir(),
	}); err != nil {
		t.Fatalf("run migrate validate: %v", err)
	}
}

func TestRunMigrateValidateUsesExplicitExternalPath(t *testing.T) {
	hooks := captureMigrateTestHooks()
	defer hooks.restore()

	root := t.TempDir()
	externalDir := filepath.Join(root, "tmp-migrations")
	createMigrationFixture(t, []string{externalDir}, map[string]string{
		filepath.Join(externalDir, "202605190001_user.sql"): "CREATE TABLE users (id bigint);\n",
	})
	writeAtlasStateFiles(t, []string{externalDir})

	if err := runMigrateValidate(migrateResolveOptions{
		migrationDir: "file:tmp-migrations",
		workingDir:   root,
	}); err != nil {
		t.Fatalf("run migrate validate: %v", err)
	}
}

func TestRunMigrateValidateRejectsRepoOwnedSelectorWithoutEmbeddedAssets(t *testing.T) {
	hooks := captureMigrateTestHooks()
	defer hooks.restore()

	migrateEmbeddedMigrationDirByPath = func(string) (moduleregistry.EmbeddedMigrationDir, bool) {
		return moduleregistry.EmbeddedMigrationDir{}, false
	}

	err := runMigrateValidate(migrateResolveOptions{
		migrationDir: "modules/user/migrations",
		workingDir:   t.TempDir(),
	})
	if err == nil {
		t.Fatal("expected missing embedded assets error")
	}
	if !strings.Contains(err.Error(), "compile-time embedded migration dir \"modules/user/migrations\" is not available") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewMigrateCommandRegistersValidateSubcommand(t *testing.T) {
	command := newMigrateCommand()
	validateCommand, _, err := command.Find([]string{"validate"})
	if err != nil {
		t.Fatalf("find validate command: %v", err)
	}
	if validateCommand == nil {
		t.Fatal("expected validate subcommand")
	}
	if validateCommand.Name() != "validate" {
		t.Fatalf("expected validate command name, got %q", validateCommand.Name())
	}
}

func TestRunMigrateUpPassesAllowDirtyToExecutor(t *testing.T) {
	hooks := captureMigrateTestHooks()
	defer hooks.restore()

	setMigrateCommandTestEnv(t)
	migrateEmbeddedMigrationDirByPath = func(path string) (moduleregistry.EmbeddedMigrationDir, bool) {
		return embeddedMigrationDir(t, path, map[string]string{
			"202605190001_user.sql": "CREATE TABLE users (id bigint);\n",
		}), true
	}

	receivedAllowDirty := false
	migrateOpenExecutor = func(_ string, _ atlasmigrate.Dir, _ atlasmigrate.Logger, allowDirty bool) (*atlasExecutorHandle, error) {
		receivedAllowDirty = allowDirty
		return &atlasExecutorHandle{
			executor: fakeAtlasExecutor{
				executeN: func(context.Context, int) error { return nil },
			},
		}, nil
	}

	if err := runMigrateUp(newSilentMigrateCommand(), migrateUpOptions{
		migrationDir: "modules/user/migrations",
		workingDir:   t.TempDir(),
		allowDirty:   true,
	}); err != nil {
		t.Fatalf("run migrate up with allow-dirty: %v", err)
	}

	if !receivedAllowDirty {
		t.Fatal("expected runMigrateUp to pass allow-dirty to the executor")
	}
}
