package cli

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	atlasmigrate "ariga.io/atlas/sql/migrate"
	atlaspostgres "ariga.io/atlas/sql/postgres"
	"github.com/spf13/cobra"

	"graft/server/internal/config"
	"graft/server/internal/moduleregistry"
)

// defaultMigrationDir 定义 `server` 模块默认迁移链使用的 registry 选择器。
const defaultMigrationDir = moduleregistry.DefaultMigrationDir

const migrationVersionMatchCount = 2

var migrationVersionPattern = regexp.MustCompile(`^(\d+)_.*\.sql$`)

// 这些变量保留为可替换的命令边界，便于测试覆盖 cwd、compile-time registry、
// 嵌入式迁移资源解析以及 Atlas 执行装配。
var migrateGetwd = os.Getwd
var migrateRegistryMigrationDirs = moduleregistry.MigrationDirs
var migrateEmbeddedMigrationDirByPath = moduleregistry.EmbeddedMigrationDirByPath
var migrateReadDir = os.ReadDir
var migrateOpenExecutor = openAtlasExecutor

// migrateUpOptions 封装一次显式迁移执行所需的输入。
type migrateUpOptions struct {
	migrationDir string
	workingDir   string
}

type atlasExecutorHandle struct {
	executor atlasExecutor
	close    func() error
}

type atlasExecutor interface {
	ExecuteN(context.Context, int) error
}

type migrationDirSource struct {
	path          string
	dir           atlasmigrate.Dir
	hasAtlasState bool
}

// newMigrateCommand 创建显式数据库迁移命令树。
//
// 迁移能力保持在独立的 `graft migrate` 子树下，避免普通运行时启动路径
// 隐式修改数据库 schema。
func newMigrateCommand() *cobra.Command {
	var migrationDir string

	command := &cobra.Command{
		Use:   "migrate",
		Short: "Run explicit database migration commands",
	}
	command.PersistentFlags().StringVar(&migrationDir, "dir", defaultMigrationDir, "migration directory or owner-aligned default chain")

	command.AddCommand(&cobra.Command{
		Use:   "up",
		Short: "Apply pending Atlas versioned migrations",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runMigrateUp(cmd, migrateUpOptions{migrationDir: migrationDir})
		},
	})

	return command
}

// runMigrateUp 执行一次 Atlas 版本化迁移应用。
//
// 参数：
//   - cmd: 当前 Cobra 命令，用于继承上下文和标准输入输出。
//   - opts: 迁移目录与工作目录等显式执行选项。
//
// 返回值：
//   - error: 当配置加载、迁移目录解析或 Atlas SDK 执行失败时返回错误。
func runMigrateUp(cmd *cobra.Command, opts migrateUpOptions) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	workingDir := opts.workingDir
	if strings.TrimSpace(workingDir) == "" {
		workingDir, err = migrateGetwd()
		if err != nil {
			return fmt.Errorf("resolve working directory: %w", err)
		}
	}

	dir, err := buildAtlasMigrationDir(workingDir, opts.migrationDir)
	if err != nil {
		return fmt.Errorf("resolve migration dir: %w", err)
	}

	commandContext := cmd.Context()
	if commandContext == nil {
		commandContext = context.Background()
	}

	handle, err := migrateOpenExecutor(cfg.Database.URL, dir, newAtlasCommandLogger(cmd))
	if err != nil {
		return err
	}
	defer func() {
		if handle.close != nil {
			_ = handle.close()
		}
	}()

	if err := handle.executor.ExecuteN(commandContext, 0); err != nil {
		if errors.Is(err, atlasmigrate.ErrNoPendingFiles) {
			return nil
		}
		return fmt.Errorf("apply atlas migrations: %w", err)
	}

	return nil
}

func openAtlasExecutor(databaseURL string, dir atlasmigrate.Dir, logger atlasmigrate.Logger) (*atlasExecutorHandle, error) {
	sqlDB, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open postgres database pool: %w", err)
	}

	driver, err := atlaspostgres.Open(sqlDB)
	if err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("open atlas postgres driver: %w", err)
	}

	executor, err := atlasmigrate.NewExecutor(
		driver,
		dir,
		newAtlasRevisionStore(sqlDB),
		atlasmigrate.WithLogger(logger),
		atlasmigrate.WithOperatorVersion("graft"),
	)
	if err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("create atlas migration executor: %w", err)
	}

	return &atlasExecutorHandle{
		executor: executor,
		close:    sqlDB.Close,
	}, nil
}

func buildAtlasMigrationDir(baseDir string, migrationDir string) (atlasmigrate.Dir, error) {
	if strings.TrimSpace(migrationDir) == "" {
		return nil, fmt.Errorf("migration dir is required")
	}

	if migrationDir != defaultMigrationDir {
		return loadSingleAtlasMigrationDir(baseDir, migrationDir)
	}

	return buildDefaultAtlasMigrationDir(baseDir)
}

func loadSingleAtlasMigrationDir(baseDir string, migrationDir string) (atlasmigrate.Dir, error) {
	source, found, err := loadMigrationDirSource(baseDir, migrationDir)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("cannot find migration dir %q from %s: %w", migrationDir, baseDir, os.ErrNotExist)
	}
	return source.dir, nil
}

func buildDefaultAtlasMigrationDir(baseDir string) (atlasmigrate.Dir, error) {
	searchDirs, err := migrateRegistryMigrationDirs()
	if err != nil {
		return nil, fmt.Errorf("load compile-time migration registry: %w", err)
	}

	sources := make([]migrationDirSource, 0, len(searchDirs))
	for _, current := range searchDirs {
		source, found, err := loadMigrationDirSource(baseDir, current)
		if err != nil {
			return nil, err
		}
		if !found || !source.hasAtlasState {
			continue
		}
		if err := atlasmigrate.Validate(source.dir); err != nil {
			return nil, fmt.Errorf("validate migration dir %s: %w", source.path, err)
		}
		sources = append(sources, source)
	}

	if len(sources) == 0 {
		return nil, fmt.Errorf("no migration directories with atlas state found in compile-time registry")
	}

	dir, err := synthesizeDefaultMigrationDir(sources)
	if err != nil {
		return nil, err
	}
	if err := atlasmigrate.Validate(dir); err != nil {
		return nil, fmt.Errorf("validate synthesized default migration dir: %w", err)
	}
	return dir, nil
}

func loadMigrationDirSource(baseDir string, migrationDir string) (migrationDirSource, bool, error) {
	if embedded, found, err := loadEmbeddedMigrationDirSource(migrationDir); found || err != nil {
		return embedded, found, err
	}

	absDir, err := resolveMigrationDir(baseDir, migrationDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return migrationDirSource{}, false, nil
		}
		return migrationDirSource{}, false, err
	}

	dir, err := atlasmigrate.NewLocalDir(absDir)
	if err != nil {
		return migrationDirSource{}, false, fmt.Errorf("open migration dir %s: %w", absDir, err)
	}
	hasAtlasState, err := directoryContainsAtlasState(absDir)
	if err != nil {
		return migrationDirSource{}, false, err
	}

	return migrationDirSource{
		path:          migrationDir,
		dir:           dir,
		hasAtlasState: hasAtlasState,
	}, true, nil
}

func loadEmbeddedMigrationDirSource(migrationDir string) (migrationDirSource, bool, error) {
	embedded, ok := migrateEmbeddedMigrationDirByPath(migrationDir)
	if !ok {
		return migrationDirSource{}, false, nil
	}

	dir := &atlasmigrate.MemDir{}
	for _, file := range embedded.Files {
		if err := dir.WriteFile(file.Name, file.Contents); err != nil {
			return migrationDirSource{}, false, fmt.Errorf("write embedded migration file %s/%s: %w", migrationDir, file.Name, err)
		}
	}

	return migrationDirSource{
		path:          migrationDir,
		dir:           dir,
		hasAtlasState: embeddedMigrationDirHasAtlasState(embedded),
	}, true, nil
}

func embeddedMigrationDirHasAtlasState(dir moduleregistry.EmbeddedMigrationDir) bool {
	for _, file := range dir.Files {
		if file.Name == atlasmigrate.HashFileName {
			return true
		}
	}
	return false
}

func synthesizeDefaultMigrationDir(sourceDirs []migrationDirSource) (atlasmigrate.Dir, error) {
	memDir := &atlasmigrate.MemDir{}
	copiedNames := make(map[string]string, len(sourceDirs))
	copiedVersions := make(map[string]string, len(sourceDirs))
	totalCopied := 0

	for _, sourceDir := range sourceDirs {
		copied, err := copyMigrationSourceFiles(memDir, sourceDir, copiedNames, copiedVersions)
		if err != nil {
			return nil, err
		}
		totalCopied += copied
	}
	if totalCopied == 0 {
		return nil, fmt.Errorf("default migration chain has no SQL migration files")
	}
	sum, err := memDir.Checksum()
	if err != nil {
		return nil, fmt.Errorf("compute synthesized migration checksum: %w", err)
	}
	if err := atlasmigrate.WriteSumFile(memDir, sum); err != nil {
		return nil, fmt.Errorf("write synthesized migration checksum: %w", err)
	}

	return memDir, nil
}

func copyMigrationSourceFiles(
	memDir *atlasmigrate.MemDir,
	sourceDir migrationDirSource,
	copiedNames map[string]string,
	copiedVersions map[string]string,
) (int, error) {
	files, err := sourceDir.dir.Files()
	if err != nil {
		return 0, fmt.Errorf("read migration dir %s: %w", sourceDir.path, err)
	}

	copiedCount := 0
	for _, file := range files {
		if err := validateSynthesizedMigrationFile(sourceDir.path, file.Name(), copiedNames, copiedVersions); err != nil {
			return 0, err
		}
		if err := memDir.WriteFile(file.Name(), file.Bytes()); err != nil {
			return 0, fmt.Errorf("write synthesized migration file %s: %w", file.Name(), err)
		}
		copiedNames[file.Name()] = sourceDir.path
		copiedCount++
	}

	return copiedCount, nil
}

func validateSynthesizedMigrationFile(
	sourcePath string,
	name string,
	copiedNames map[string]string,
	copiedVersions map[string]string,
) error {
	if previousSource, exists := copiedNames[name]; exists {
		return fmt.Errorf("duplicate migration filename %s from %s and %s", name, previousSource, sourcePath)
	}
	if version := migrationFileVersion(name); version != "" {
		if previousSource, exists := copiedVersions[version]; exists {
			return fmt.Errorf("duplicate migration version %s from %s and %s", version, previousSource, sourcePath)
		}
		copiedVersions[version] = sourcePath
	}

	return nil
}

func migrationFileVersion(name string) string {
	matches := migrationVersionPattern.FindStringSubmatch(name)
	if len(matches) != migrationVersionMatchCount {
		return ""
	}

	return matches[1]
}

// resolveMigrationDirs 从当前目录向上搜索可用的迁移目录集合。
//
// 默认目录不再直接等同于单个 core 迁移路径；它会先回到 compile-time
// registry 读取当前进程声明的完整目录集合，再逐一解析为绝对路径。
func resolveMigrationDirs(baseDir string, migrationDir string) ([]string, error) {
	if strings.TrimSpace(migrationDir) == "" {
		return nil, fmt.Errorf("migration dir is required")
	}

	includeAllResolvedDirs := true
	searchDirs := []string{migrationDir}
	if migrationDir == defaultMigrationDir {
		var err error
		searchDirs, err = migrateRegistryMigrationDirs()
		if err != nil {
			return nil, fmt.Errorf("load compile-time migration registry: %w", err)
		}
		includeAllResolvedDirs = false
	}

	resolved := make([]string, 0, len(searchDirs))
	for _, current := range searchDirs {
		absDir, err := resolveMigrationDir(baseDir, current)
		if err != nil {
			if shouldSkipMissingMigrationDir(includeAllResolvedDirs, err) {
				continue
			}
			return nil, err
		}

		resolved, err = appendResolvedMigrationDir(resolved, absDir, includeAllResolvedDirs)
		if err != nil {
			return nil, err
		}
	}

	if len(resolved) == 0 {
		return nil, fmt.Errorf("no migration directories with atlas state found in compile-time registry")
	}

	return resolved, nil
}

func shouldSkipMissingMigrationDir(includeAllResolvedDirs bool, err error) bool {
	return !includeAllResolvedDirs && errors.Is(err, os.ErrNotExist)
}

func appendResolvedMigrationDir(resolved []string, absDir string, includeAllResolvedDirs bool) ([]string, error) {
	if includeAllResolvedDirs {
		return append(resolved, absDir), nil
	}

	hasAtlasState, err := directoryContainsAtlasState(absDir)
	if err != nil {
		return nil, err
	}
	if !hasAtlasState {
		return resolved, nil
	}

	return append(resolved, absDir), nil
}

// resolveMigrationDir 从当前目录向上搜索可用的单个迁移目录。
//
// 默认目录同时支持仓库根目录和 `server` 模块根目录两种工作目录，减少 IDE、
// Shell 和测试环境切换时对单一 cwd 约定的依赖。
func resolveMigrationDir(baseDir string, migrationDir string) (string, error) {
	if strings.TrimSpace(migrationDir) == "" {
		return "", fmt.Errorf("migration dir is required")
	}

	searchDirs := []string{migrationDir}
	if !strings.HasPrefix(filepath.ToSlash(migrationDir), "server/") {
		searchDirs = append(searchDirs, filepath.Join("server", migrationDir))
	}

	current := baseDir
	for {
		for _, relativeDir := range searchDirs {
			candidate := filepath.Join(current, relativeDir)
			info, err := os.Stat(candidate)
			if err == nil {
				if !info.IsDir() {
					return "", fmt.Errorf("migration dir %s is not a directory", candidate)
				}

				absDir, err := filepath.Abs(candidate)
				if err != nil {
					return "", fmt.Errorf("resolve migration dir %s: %w", candidate, err)
				}

				return absDir, nil
			}
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return "", fmt.Errorf("cannot find migration dir %q from %s: %w", migrationDir, baseDir, os.ErrNotExist)
}

func directoryContainsAtlasState(absDir string) (bool, error) {
	entries, err := migrateReadDir(absDir)
	if err != nil {
		return false, fmt.Errorf("read migration dir %s: %w", absDir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if entry.Name() == atlasmigrate.HashFileName {
			return true, nil
		}
	}

	return false, nil
}

type atlasRevisionStore struct {
	db       *sql.DB
	initOnce sync.Once
	initErr  error
}

func newAtlasRevisionStore(db *sql.DB) *atlasRevisionStore {
	return &atlasRevisionStore{db: db}
}

func (s *atlasRevisionStore) Ident() *atlasmigrate.TableIdent {
	return &atlasmigrate.TableIdent{Name: "atlas_schema_revisions"}
}

func (s *atlasRevisionStore) ReadRevisions(ctx context.Context) ([]*atlasmigrate.Revision, error) {
	if err := s.ensureTable(ctx); err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(
		ctx,
		`SELECT version, description, type, applied, total, executed_at, execution_time, error, error_stmt, hash, partial_hashes, operator_version
		FROM atlas_schema_revisions
		ORDER BY version ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("query revision history: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	revisions := make([]*atlasmigrate.Revision, 0)
	for rows.Next() {
		revision, err := scanAtlasRevision(rows.Scan)
		if err != nil {
			return nil, err
		}
		revisions = append(revisions, revision)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate revision history: %w", err)
	}

	return revisions, nil
}

func (s *atlasRevisionStore) ReadRevision(ctx context.Context, version string) (*atlasmigrate.Revision, error) {
	if err := s.ensureTable(ctx); err != nil {
		return nil, err
	}

	row := s.db.QueryRowContext(
		ctx,
		`SELECT version, description, type, applied, total, executed_at, execution_time, error, error_stmt, hash, partial_hashes, operator_version
		FROM atlas_schema_revisions
		WHERE version = $1`,
		version,
	)

	revision, err := scanAtlasRevision(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, atlasmigrate.ErrRevisionNotExist
	}
	if err != nil {
		return nil, err
	}

	return revision, nil
}

func (s *atlasRevisionStore) WriteRevision(ctx context.Context, revision *atlasmigrate.Revision) error {
	if err := s.ensureTable(ctx); err != nil {
		return err
	}

	var partialHashes any
	if len(revision.PartialHashes) > 0 {
		encoded, err := json.Marshal(revision.PartialHashes)
		if err != nil {
			return fmt.Errorf("marshal partial hashes for revision %s: %w", revision.Version, err)
		}
		partialHashes = encoded
	}
	revisionType, err := revisionTypeToInt64(revision.Type)
	if err != nil {
		return fmt.Errorf("encode revision type for %s: %w", revision.Version, err)
	}

	if _, err := s.db.ExecContext(
		ctx,
		`INSERT INTO atlas_schema_revisions (
			version, description, type, applied, total, executed_at, execution_time, error, error_stmt, hash, partial_hashes, operator_version
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)
		ON CONFLICT (version) DO UPDATE SET
			description = EXCLUDED.description,
			type = EXCLUDED.type,
			applied = EXCLUDED.applied,
			total = EXCLUDED.total,
			executed_at = EXCLUDED.executed_at,
			execution_time = EXCLUDED.execution_time,
			error = EXCLUDED.error,
			error_stmt = EXCLUDED.error_stmt,
			hash = EXCLUDED.hash,
			partial_hashes = EXCLUDED.partial_hashes,
			operator_version = EXCLUDED.operator_version`,
		revision.Version,
		revision.Description,
		revisionType,
		revision.Applied,
		revision.Total,
		revision.ExecutedAt,
		revision.ExecutionTime.Nanoseconds(),
		revision.Error,
		revision.ErrorStmt,
		revision.Hash,
		partialHashes,
		revision.OperatorVersion,
	); err != nil {
		return fmt.Errorf("upsert revision %s: %w", revision.Version, err)
	}

	return nil
}

func (s *atlasRevisionStore) DeleteRevision(ctx context.Context, version string) error {
	if err := s.ensureTable(ctx); err != nil {
		return err
	}

	if _, err := s.db.ExecContext(ctx, `DELETE FROM atlas_schema_revisions WHERE version = $1`, version); err != nil {
		return fmt.Errorf("delete revision %s: %w", version, err)
	}
	return nil
}

func (s *atlasRevisionStore) ensureTable(ctx context.Context) error {
	s.initOnce.Do(func() {
		_, s.initErr = s.db.ExecContext(
			ctx,
			`CREATE TABLE IF NOT EXISTS atlas_schema_revisions (
				version VARCHAR(255) PRIMARY KEY,
				description TEXT NOT NULL DEFAULT '',
				type BIGINT NOT NULL DEFAULT 0,
				applied BIGINT NOT NULL DEFAULT 0,
				total BIGINT NOT NULL DEFAULT 0,
				executed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
				execution_time BIGINT NOT NULL DEFAULT 0,
				error TEXT NOT NULL DEFAULT '',
				error_stmt TEXT NOT NULL DEFAULT '',
				hash TEXT NOT NULL DEFAULT '',
				partial_hashes JSONB NULL,
				operator_version TEXT NOT NULL DEFAULT ''
			)`,
		)
	})
	if s.initErr != nil {
		return fmt.Errorf("ensure atlas_schema_revisions table: %w", s.initErr)
	}
	return nil
}

func scanAtlasRevision(scan func(dest ...any) error) (*atlasmigrate.Revision, error) {
	var (
		version         string
		description     string
		revisionType    int64
		applied         int
		total           int
		executedAt      time.Time
		executionTimeNS int64
		errorText       string
		errorStmt       string
		hash            string
		partialHashes   []byte
		operatorVersion string
	)

	if err := scan(
		&version,
		&description,
		&revisionType,
		&applied,
		&total,
		&executedAt,
		&executionTimeNS,
		&errorText,
		&errorStmt,
		&hash,
		&partialHashes,
		&operatorVersion,
	); err != nil {
		return nil, err
	}

	var hashes []string
	if len(partialHashes) > 0 {
		if err := json.Unmarshal(partialHashes, &hashes); err != nil {
			return nil, fmt.Errorf("decode partial hashes for revision %s: %w", version, err)
		}
	}
	migrationType, err := revisionTypeFromInt64(revisionType)
	if err != nil {
		return nil, fmt.Errorf("decode revision type for %s: %w", version, err)
	}

	return &atlasmigrate.Revision{
		Version:         version,
		Description:     description,
		Type:            migrationType,
		Applied:         applied,
		Total:           total,
		ExecutedAt:      executedAt,
		ExecutionTime:   time.Duration(executionTimeNS),
		Error:           errorText,
		ErrorStmt:       errorStmt,
		Hash:            hash,
		PartialHashes:   hashes,
		OperatorVersion: operatorVersion,
	}, nil
}

func revisionTypeToInt64(value atlasmigrate.RevisionType) (int64, error) {
	raw := uint64(value)
	if raw > math.MaxInt64 {
		return 0, fmt.Errorf("revision type %d exceeds int64 storage", raw)
	}
	return int64(raw), nil
}

func revisionTypeFromInt64(value int64) (atlasmigrate.RevisionType, error) {
	if value < 0 {
		return 0, fmt.Errorf("revision type %d cannot be negative", value)
	}
	return atlasmigrate.RevisionType(value), nil
}

type atlasCommandLogger struct {
	stdout io.Writer
	stderr io.Writer
}

func newAtlasCommandLogger(cmd *cobra.Command) atlasmigrate.Logger {
	if cmd == nil {
		return atlasmigrate.NopLogger{}
	}
	return atlasCommandLogger{
		stdout: cmd.OutOrStdout(),
		stderr: cmd.ErrOrStderr(),
	}
}

func (l atlasCommandLogger) Log(entry atlasmigrate.LogEntry) {
	switch current := entry.(type) {
	case atlasmigrate.LogExecution:
		if len(current.Files) == 0 {
			_, _ = fmt.Fprintln(l.stdout, "No pending migrations.")
			return
		}
		_, _ = fmt.Fprintf(l.stdout, "Applying %d migration file(s)...\n", len(current.Files))
	case atlasmigrate.LogFile:
		_, _ = fmt.Fprintf(l.stdout, "Applying %s\n", current.File.Name())
	case atlasmigrate.LogDone:
		_, _ = fmt.Fprintln(l.stdout, "Migration complete.")
	case atlasmigrate.LogError:
		if current.Error != nil {
			_, _ = fmt.Fprintln(l.stderr, current.Error.Error())
		}
	}
}
