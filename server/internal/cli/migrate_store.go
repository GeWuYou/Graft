package cli

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
	"sync"
	"time"

	atlasmigrate "ariga.io/atlas/sql/migrate"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/spf13/cobra"
)

type atlasRevisionStore struct {
	db       *sql.DB
	initOnce sync.Once
	initErr  error
}

var _ atlasmigrate.RevisionReadWriter = (*atlasRevisionStore)(nil)

const atlasRevisionStoreCreateTableSQL = `CREATE TABLE IF NOT EXISTS atlas_schema_revisions (
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
			)`

// newAtlasRevisionStore 为给定的数据库连接创建一个新的 Atlas 修订存储实例。
func newAtlasRevisionStore(db *sql.DB) *atlasRevisionStore {
	return &atlasRevisionStore{db: db}
}

func (s *atlasRevisionStore) Ident() *atlasmigrate.TableIdent {
	return &atlasmigrate.TableIdent{Name: "atlas_schema_revisions"}
}

func (s *atlasRevisionStore) ReadRevisions(ctx context.Context) ([]*atlasmigrate.Revision, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT version, description, type, applied, total, executed_at, execution_time, error, error_stmt, hash, partial_hashes, operator_version
		FROM atlas_schema_revisions
		ORDER BY version ASC`,
	)
	if err != nil {
		if isAtlasRevisionTableMissingError(err) {
			return nil, nil
		}
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
	row := s.db.QueryRowContext(
		ctx,
		`SELECT version, description, type, applied, total, executed_at, execution_time, error, error_stmt, hash, partial_hashes, operator_version
		FROM atlas_schema_revisions
		WHERE version = $1`,
		version,
	)

	revision, err := scanAtlasRevision(row.Scan)
	if isAtlasRevisionTableMissingError(err) {
		return nil, atlasmigrate.ErrRevisionNotExist
	}
	if errors.Is(err, sql.ErrNoRows) {
		return nil, atlasmigrate.ErrRevisionNotExist
	}
	if err != nil {
		return nil, err
	}

	return revision, nil
}

func (s *atlasRevisionStore) WriteRevision(ctx context.Context, revision *atlasmigrate.Revision) error {
	if revision == nil {
		return fmt.Errorf("write revision: revision is required")
	}
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
	if _, err := s.db.ExecContext(ctx, `DELETE FROM atlas_schema_revisions WHERE version = $1`, version); err != nil {
		if isAtlasRevisionTableMissingError(err) {
			return nil
		}
		return fmt.Errorf("delete revision %s: %w", version, err)
	}
	return nil
}

func (s *atlasRevisionStore) ensureTable(ctx context.Context) error {
	s.initOnce.Do(func() {
		_, s.initErr = s.db.ExecContext(ctx, atlasRevisionStoreCreateTableSQL)
	})
	if s.initErr != nil {
		return fmt.Errorf("ensure atlas_schema_revisions table: %w", s.initErr)
	}
	return nil
}

func isAtlasRevisionTableMissingError(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "42P01" {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "no such table: atlas_schema_revisions") ||
		strings.Contains(message, "relation \"atlas_schema_revisions\" does not exist")
}

// scanAtlasRevision 将数据库行数据扫描并映射为一个 Atlas 迁移版本记录。
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

// revisionTypeToInt64 converts a revision type value to an int64, returning an error if the value exceeds math.MaxInt64.
func revisionTypeToInt64(value atlasmigrate.RevisionType) (int64, error) {
	raw := uint64(value)
	if raw > math.MaxInt64 {
		return 0, fmt.Errorf("revision type %d exceeds int64 storage", raw)
	}
	return int64(raw), nil
}

// revisionTypeFromInt64 将 int64 值转换为 RevisionType，如果该值为负则返回错误。
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

// newAtlasCommandLogger creates an Atlas logger that writes to the command's standard output and error streams, or returns a no-op logger if the command is nil.
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
