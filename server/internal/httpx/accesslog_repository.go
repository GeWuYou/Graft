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
	accessLogSQLDialectPostgres  accessLogSQLDialect = "postgres"
	accessLogSQLDialectSQLite    accessLogSQLDialect = "sqlite"
	accessLogDefaultPageSize                         = 20
	accessLogMaxPageSize                             = 100
	accessLogListClauseCapacity                      = 10
	accessLogListOffsetArgCount                      = 2
	accessLogDeleteLimitArgIndex                     = 2
	accessLogKeywordClauseCount                      = 3
	accessLogStatus4xxMin                            = 400
	accessLogStatus4xxMax                            = 499
	accessLogStatus5xxMin                            = 500
	accessLogStatus5xxMax                            = 599
)

// AccessLog describes one persisted canonical access-log record.
type AccessLog struct {
	ID           uint64
	RequestID    string
	TraceID      string
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
	StartedAt    time.Time
	OccurredAt   time.Time
}

// CreateAccessLogInput describes the canonical request facts persisted by the runtime owner.
type CreateAccessLogInput struct {
	RequestID    string
	TraceID      string
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
	StartedAt    time.Time
	OccurredAt   time.Time
}

// AccessLogRepository owns durable persistence for canonical access logs.
type AccessLogRepository interface {
	CreateAccessLog(ctx context.Context, input CreateAccessLogInput) (AccessLog, error)
	CreateAccessLogs(ctx context.Context, inputs []CreateAccessLogInput) ([]AccessLog, error)
	DeleteAccessLogsBefore(ctx context.Context, occurredBefore time.Time) (int64, error)
	DeleteAccessLogsBeforeLimit(ctx context.Context, occurredBefore time.Time, limit int) (int64, error)
	ListAccessLogs(ctx context.Context, query AccessLogListQuery) (AccessLogListResult, error)
	GetAccessLogByID(ctx context.Context, id uint64) (AccessLog, error)
}

// ErrAccessLogNotFound 表示按 canonical id 未找到访问日志记录。
var ErrAccessLogNotFound = errors.New("access log not found")

// AccessLogListQuery 描述 access-log explorer 可消费的标准筛选和排序条件。
type AccessLogListQuery struct {
	Page          int
	PageSize      int
	RequestID     string
	TraceID       string
	Keyword       string
	UserID        *uint64
	Username      string
	Method        string
	Path          string
	PathMatchMode AccessLogPathMatchMode
	Route         string
	StatusCode    *int
	StatusGroups  []AccessLogStatusGroup
	DurationMinMS *int64
	DurationMaxMS *int64
	StartedFrom   *time.Time
	StartedTo     *time.Time
	OccurredFrom  *time.Time
	OccurredTo    *time.Time
	Sorts         []AccessLogSort
}

// AccessLogListResult 承载访问日志列表查询的分页结果。
type AccessLogListResult struct {
	Items    []AccessLog
	Total    int64
	Page     int
	PageSize int
}

// AccessLogPathMatchMode 约束路径筛选的匹配方式。
type AccessLogPathMatchMode string

const (
	// AccessLogPathMatchExact 使用完整路径精确匹配。
	AccessLogPathMatchExact AccessLogPathMatchMode = "exact"
	// AccessLogPathMatchPrefix 使用路径前缀匹配。
	AccessLogPathMatchPrefix AccessLogPathMatchMode = "prefix"
)

// AccessLogSortField 约束 access-log explorer 支持的排序字段。
type AccessLogSortField string

const (
	// AccessLogSortStartedAt 按请求开始时间排序。
	AccessLogSortStartedAt AccessLogSortField = "started_at"
	// AccessLogSortOccurredAt 按发生时间排序。
	AccessLogSortOccurredAt AccessLogSortField = "occurred_at"
	// AccessLogSortDurationMS 按耗时排序。
	AccessLogSortDurationMS AccessLogSortField = "duration_ms"
	// AccessLogSortStatusCode 按状态码排序。
	AccessLogSortStatusCode AccessLogSortField = "status_code"
)

// AccessLogSortOrder 约束 access-log explorer 支持的排序方向。
type AccessLogSortOrder string

const (
	// AccessLogSortOrderAsc 表示升序。
	AccessLogSortOrderAsc AccessLogSortOrder = "asc"
	// AccessLogSortOrderDesc 表示降序。
	AccessLogSortOrderDesc AccessLogSortOrder = "desc"
)

// AccessLogStatusGroup 约束 access-log explorer 支持的状态码分组。
type AccessLogStatusGroup string

const (
	// AccessLogStatusGroup4xx 表示 400-499 的客户端错误状态码。
	AccessLogStatusGroup4xx AccessLogStatusGroup = "4xx"
	// AccessLogStatusGroup5xx 表示 500-599 的服务端错误状态码。
	AccessLogStatusGroup5xx AccessLogStatusGroup = "5xx"
)

// AccessLogSort 表示一个稳定的优先级排序项。
type AccessLogSort struct {
	Field AccessLogSortField
	Order AccessLogSortOrder
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

func (r *accessLogRepository) DeleteAccessLogsBeforeLimit(ctx context.Context, occurredBefore time.Time, limit int) (int64, error) {
	if r == nil || r.db == nil {
		return 0, errors.New("access log repository is unavailable")
	}
	if limit <= 0 {
		return 0, errors.New("access log delete limit must be greater than zero")
	}

	//nolint:gosec // Query shape is fixed; placeholders come from the internal dialect helper and values stay parameterized.
	query := fmt.Sprintf(
		"DELETE FROM access_logs WHERE id IN (SELECT id FROM access_logs WHERE occurred_at < %s ORDER BY occurred_at ASC, id ASC LIMIT %s)",
		r.placeholder(1),
		r.placeholder(accessLogDeleteLimitArgIndex),
	)
	result, err := r.db.ExecContext(ctx, query, occurredBefore.UTC(), limit)
	if err != nil {
		return 0, fmt.Errorf("delete access logs before cutoff with limit: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("read deleted access log row count: %w", err)
	}

	return rowsAffected, nil
}

func (r *accessLogRepository) ListAccessLogs(ctx context.Context, query AccessLogListQuery) (AccessLogListResult, error) {
	if r == nil || r.db == nil {
		return AccessLogListResult{}, errors.New("access log repository is unavailable")
	}

	normalized := normalizeAccessLogListQuery(query)
	whereSQL, args := r.buildAccessLogWhereClause(normalized)

	countQuery := "SELECT COUNT(*) FROM access_logs" + whereSQL
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return AccessLogListResult{}, fmt.Errorf("count access logs: %w", err)
	}

	listArgs := append([]any(nil), args...)
	listArgs = append(listArgs, normalized.PageSize, (normalized.Page-1)*normalized.PageSize)
	selectQuery := r.buildAccessLogListSelectQuery(whereSQL, normalized, len(args))

	rows, err := r.db.QueryContext(ctx, selectQuery, listArgs...)
	if err != nil {
		return AccessLogListResult{}, fmt.Errorf("list access logs: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	items := make([]AccessLog, 0, normalized.PageSize)
	for rows.Next() {
		record, scanErr := scanAccessLog(rows)
		if scanErr != nil {
			return AccessLogListResult{}, fmt.Errorf("scan access log: %w", scanErr)
		}
		items = append(items, record)
	}
	if err := rows.Err(); err != nil {
		return AccessLogListResult{}, fmt.Errorf("iterate access logs: %w", err)
	}

	return AccessLogListResult{
		Items:    items,
		Total:    total,
		Page:     normalized.Page,
		PageSize: normalized.PageSize,
	}, nil
}

func (r *accessLogRepository) GetAccessLogByID(ctx context.Context, id uint64) (AccessLog, error) {
	if r == nil || r.db == nil {
		return AccessLog{}, errors.New("access log repository is unavailable")
	}

	//nolint:gosec // 占位符仅由内部 dialect helper 生成，不接受外部输入。
	query := `SELECT
		id,
		request_id,
		trace_id,
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
		started_at,
		occurred_at
	FROM access_logs WHERE id = ` + r.placeholder(1)

	row := r.db.QueryRowContext(ctx, query, id)
	record, err := scanAccessLog(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AccessLog{}, ErrAccessLogNotFound
		}
		return AccessLog{}, fmt.Errorf("get access log: %w", err)
	}

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
