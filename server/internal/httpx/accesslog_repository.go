package httpx

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type accessLogSQLDialect string

const (
	accessLogSQLDialectPostgres accessLogSQLDialect = "postgres"
	accessLogSQLDialectSQLite   accessLogSQLDialect = "sqlite"
)

// AccessLog describes one persisted canonical access-log record.
type AccessLog struct {
	ID           uint64
	RequestID    string
	Method       string
	Path         string
	Route        string
	StatusCode   int
	DurationMS   int64
	ClientIP     string
	UserAgent    string
	UserID       *uint64
	Username     string
	RequestSize  *int64
	ResponseSize *int64
	OccurredAt   time.Time
}

// CreateAccessLogInput describes the canonical request facts persisted by the runtime owner.
type CreateAccessLogInput struct {
	RequestID    string
	Method       string
	Path         string
	Route        string
	StatusCode   int
	DurationMS   int64
	ClientIP     string
	UserAgent    string
	UserID       *uint64
	Username     string
	RequestSize  *int64
	ResponseSize *int64
	OccurredAt   time.Time
}

// AccessLogRepository owns durable persistence for canonical access logs.
type AccessLogRepository interface {
	CreateAccessLog(ctx context.Context, input CreateAccessLogInput) (AccessLog, error)
	CreateAccessLogs(ctx context.Context, inputs []CreateAccessLogInput) ([]AccessLog, error)
	DeleteAccessLogsBefore(ctx context.Context, occurredBefore time.Time) (int64, error)
}

type accessLogRepository struct {
	db      *sql.DB
	dialect accessLogSQLDialect
}

type accessLogQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

// NewAccessLogRepository builds the core-owned SQL repository for access-log persistence.
func NewAccessLogRepository(db *sql.DB) (AccessLogRepository, error) {
	return newAccessLogRepositoryWithDialect(db, accessLogSQLDialectPostgres)
}

func newAccessLogRepositoryWithDialect(db *sql.DB, dialect accessLogSQLDialect) (AccessLogRepository, error) {
	if db == nil {
		return nil, errors.New("access log repository requires a non-nil sql db")
	}
	if dialect == "" {
		dialect = accessLogSQLDialectPostgres
	}

	return &accessLogRepository{db: db, dialect: dialect}, nil
}

func (r *accessLogRepository) CreateAccessLog(ctx context.Context, input CreateAccessLogInput) (AccessLog, error) {
	if r == nil || r.db == nil {
		return AccessLog{}, errors.New("access log repository is unavailable")
	}

	return r.createAccessLog(ctx, r.db, input)
}

func (r *accessLogRepository) CreateAccessLogs(ctx context.Context, inputs []CreateAccessLogInput) ([]AccessLog, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("access log repository is unavailable")
	}
	if len(inputs) == 0 {
		return []AccessLog{}, nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin access log batch create transaction: %w", err)
	}

	items := make([]AccessLog, 0, len(inputs))
	for _, input := range inputs {
		record, createErr := r.createAccessLog(ctx, tx, input)
		if createErr != nil {
			_ = tx.Rollback()
			return nil, createErr
		}
		items = append(items, record)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit access log batch create transaction: %w", err)
	}

	return items, nil
}

func (r *accessLogRepository) DeleteAccessLogsBefore(ctx context.Context, occurredBefore time.Time) (int64, error) {
	if r == nil || r.db == nil {
		return 0, errors.New("access log repository is unavailable")
	}

	result, err := r.db.ExecContext(
		ctx,
		fmt.Sprintf("DELETE FROM access_logs WHERE occurred_at < %s", r.placeholder(1)),
		occurredBefore.UTC(),
	)
	if err != nil {
		return 0, fmt.Errorf("delete access logs before cutoff: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("read deleted access log row count: %w", err)
	}

	return rowsAffected, nil
}

func (r *accessLogRepository) createAccessLog(
	ctx context.Context,
	queryer accessLogQueryer,
	input CreateAccessLogInput,
) (AccessLog, error) {
	record := normalizeCreateAccessLogInput(input)

	args := []any{
		record.RequestID,
		record.Method,
		record.Path,
		nullableString(record.Route),
		record.StatusCode,
		record.DurationMS,
		nullableString(record.ClientIP),
		nullableString(record.UserAgent),
		nullableUint64(record.UserID),
		nullableString(record.Username),
		nullableInt64(record.RequestSize),
		nullableInt64(record.ResponseSize),
		record.OccurredAt,
	}

	query := fmt.Sprintf(`INSERT INTO access_logs (
		request_id,
		method,
		path,
		route,
		status_code,
		duration_ms,
		client_ip,
		user_agent,
		user_id,
		username,
		request_size,
		response_size,
		occurred_at
	) VALUES (%s) RETURNING id`, r.placeholders(len(args)))

	var id int64
	if err := queryer.QueryRowContext(ctx, query, args...).Scan(&id); err != nil {
		return AccessLog{}, fmt.Errorf("create access log: %w", err)
	}

	record.ID = uint64(id)
	return record, nil
}

func (r *accessLogRepository) placeholders(count int) string {
	values := make([]string, 0, count)
	for index := 1; index <= count; index++ {
		values = append(values, r.placeholder(index))
	}
	return strings.Join(values, ", ")
}

func (r *accessLogRepository) placeholder(index int) string {
	if r != nil && r.dialect == accessLogSQLDialectSQLite {
		return "?"
	}
	return "$" + strconv.Itoa(index)
}

func normalizeCreateAccessLogInput(input CreateAccessLogInput) AccessLog {
	return AccessLog{
		RequestID:    strings.TrimSpace(input.RequestID),
		Method:       strings.TrimSpace(input.Method),
		Path:         sanitizeAccessLogPath(input.Path),
		Route:        sanitizeAccessLogRoute(input.Route),
		StatusCode:   input.StatusCode,
		DurationMS:   input.DurationMS,
		ClientIP:     strings.TrimSpace(input.ClientIP),
		UserAgent:    sanitizeAccessLogFreeText(input.UserAgent),
		UserID:       cloneUint64Pointer(input.UserID),
		Username:     strings.TrimSpace(input.Username),
		RequestSize:  cloneInt64Pointer(input.RequestSize),
		ResponseSize: cloneInt64Pointer(input.ResponseSize),
		OccurredAt:   normalizeOccurredAt(input.OccurredAt),
	}
}

func normalizeOccurredAt(occurredAt time.Time) time.Time {
	if occurredAt.IsZero() {
		return time.Now().UTC()
	}
	return occurredAt.UTC()
}

func nullableString(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func nullableUint64(value *uint64) any {
	if value == nil {
		return nil
	}
	return int64(*value)
}

func nullableInt64(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}

func cloneUint64Pointer(value *uint64) *uint64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneInt64Pointer(value *int64) *int64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
