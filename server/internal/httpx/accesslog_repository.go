package httpx

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

type accessLogSQLDialect string

const (
	accessLogSQLDialectPostgres accessLogSQLDialect = "postgres"
	accessLogSQLDialectSQLite   accessLogSQLDialect = "sqlite"
	accessLogDefaultPageSize                        = 20
	accessLogMaxPageSize                            = 100
	accessLogListClauseCapacity                     = 10
	accessLogListOffsetArgCount                     = 2
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
	UserID        *uint64
	Username      string
	Method        string
	Path          string
	PathMatchMode AccessLogPathMatchMode
	Route         string
	StatusCode    *int
	DurationMinMS *int64
	DurationMaxMS *int64
	StartedFrom   *time.Time
	StartedTo     *time.Time
	OccurredFrom  *time.Time
	OccurredTo    *time.Time
	SortBy        AccessLogSortField
	SortOrder     AccessLogSortOrder
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

func (r *accessLogRepository) createAccessLog(
	ctx context.Context,
	queryer accessLogQueryer,
	input CreateAccessLogInput,
) (AccessLog, error) {
	record := normalizeCreateAccessLogInput(input)
	userIDValue, err := nullableUint64(record.UserID)
	if err != nil {
		return AccessLog{}, fmt.Errorf("create access log: %w", err)
	}

	args := []any{
		record.RequestID,
		record.TraceID,
		record.Method,
		record.Path,
		nullableString(record.Route),
		record.StatusCode,
		record.DurationMS,
		nullableString(record.ClientIP),
		nullableString(record.UserAgent),
		userIDValue,
		nullableString(record.Username),
		nullableInt64(record.RequestSize),
		nullableInt64(record.ResponseSize),
		record.StartedAt,
		record.OccurredAt,
	}

	query := fmt.Sprintf(`INSERT INTO access_logs (
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
	) VALUES (%s) RETURNING id`, r.placeholders(len(args)))

	var id int64
	if err := queryer.QueryRowContext(ctx, query, args...).Scan(&id); err != nil {
		return AccessLog{}, fmt.Errorf("create access log: %w", err)
	}
	if id < 0 {
		return AccessLog{}, fmt.Errorf("create access log: negative id %d", id)
	}

	//nolint:gosec // Negative values are rejected above, so this database-generated identifier stays non-negative.
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
	requestID := strings.TrimSpace(input.RequestID)
	traceID := normalizeAccessLogTraceID(strings.TrimSpace(input.TraceID), requestID)

	return AccessLog{
		RequestID:    requestID,
		TraceID:      traceID,
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
		StartedAt:    normalizeStartedAt(input.StartedAt, input.OccurredAt),
		OccurredAt:   normalizeOccurredAt(input.OccurredAt),
	}
}

func normalizeStartedAt(startedAt time.Time, occurredAt time.Time) time.Time {
	if startedAt.IsZero() {
		return normalizeOccurredAt(occurredAt)
	}
	return startedAt.UTC()
}

func normalizeOccurredAt(occurredAt time.Time) time.Time {
	if occurredAt.IsZero() {
		return time.Now().UTC()
	}
	return occurredAt.UTC()
}

func normalizeAccessLogListQuery(query AccessLogListQuery) AccessLogListQuery {
	query.Page = normalizePositivePage(query.Page)
	query.PageSize = normalizePageSize(query.PageSize)
	query.RequestID = strings.TrimSpace(query.RequestID)
	query.TraceID = strings.TrimSpace(query.TraceID)
	query.Username = strings.TrimSpace(query.Username)
	query.Method = strings.TrimSpace(query.Method)
	query.Path = sanitizeAccessLogPath(query.Path)
	query.Route = sanitizeAccessLogRoute(query.Route)
	query.PathMatchMode = normalizeAccessLogPathMatchMode(query.PathMatchMode)
	query.SortBy = normalizeAccessLogSortField(query.SortBy)
	query.SortOrder = normalizeAccessLogSortOrder(query.SortOrder)
	return query
}

func normalizeAccessLogTraceID(traceID string, requestID string) string {
	if traceID == "" || traceID == requestID {
		return ""
	}
	return traceID
}

func normalizePositivePage(page int) int {
	if page < 1 {
		return 1
	}
	return page
}

func normalizePageSize(pageSize int) int {
	switch {
	case pageSize < 1:
		return accessLogDefaultPageSize
	case pageSize > accessLogMaxPageSize:
		return accessLogMaxPageSize
	default:
		return pageSize
	}
}

func normalizeAccessLogPathMatchMode(mode AccessLogPathMatchMode) AccessLogPathMatchMode {
	if mode == AccessLogPathMatchPrefix {
		return AccessLogPathMatchPrefix
	}
	return AccessLogPathMatchExact
}

func normalizeAccessLogSortField(field AccessLogSortField) AccessLogSortField {
	switch field {
	case AccessLogSortStartedAt, AccessLogSortOccurredAt, AccessLogSortDurationMS, AccessLogSortStatusCode:
		return field
	default:
		return AccessLogSortStartedAt
	}
}

func normalizeAccessLogSortOrder(order AccessLogSortOrder) AccessLogSortOrder {
	if order == AccessLogSortOrderAsc {
		return AccessLogSortOrderAsc
	}
	return AccessLogSortOrderDesc
}

func accessLogSortColumn(field AccessLogSortField) string {
	switch field {
	case AccessLogSortStartedAt:
		return "started_at"
	case AccessLogSortDurationMS:
		return "duration_ms"
	case AccessLogSortStatusCode:
		return "status_code"
	case AccessLogSortOccurredAt:
		fallthrough
	default:
		return "occurred_at"
	}
}

func accessLogSortDirection(order AccessLogSortOrder) string {
	switch order {
	case AccessLogSortOrderAsc:
		return "ASC"
	case AccessLogSortOrderDesc:
		fallthrough
	default:
		return "DESC"
	}
}

func (r *accessLogRepository) buildAccessLogListSelectQuery(
	whereSQL string,
	query AccessLogListQuery,
	filterArgCount int,
) string {
	var builder strings.Builder
	builder.WriteString(`SELECT
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
	FROM access_logs`)
	builder.WriteString(whereSQL)
	builder.WriteString(" ORDER BY ")
	builder.WriteString(accessLogSortColumn(query.SortBy))
	builder.WriteByte(' ')
	builder.WriteString(accessLogSortDirection(query.SortOrder))
	builder.WriteString(", id DESC LIMIT ")
	builder.WriteString(r.placeholder(filterArgCount + 1))
	builder.WriteString(" OFFSET ")
	builder.WriteString(r.placeholder(filterArgCount + accessLogListOffsetArgCount))
	return builder.String()
}

func (r *accessLogRepository) buildAccessLogWhereClause(query AccessLogListQuery) (string, []any) {
	conditions := make([]string, 0, accessLogListClauseCapacity)
	args := make([]any, 0, accessLogListClauseCapacity)

	appendAccessLogEqualityFilter(&conditions, &args, r, "request_id =", query.RequestID)
	appendAccessLogEqualityFilter(&conditions, &args, r, "request_id =", query.TraceID)
	appendAccessLogOptionalUint64Filter(&conditions, &args, r, "user_id =", query.UserID)
	appendAccessLogEqualityFilter(&conditions, &args, r, "username =", query.Username)
	appendAccessLogEqualityFilter(&conditions, &args, r, "method =", query.Method)
	appendAccessLogPathFilter(&conditions, &args, r, query)
	appendAccessLogEqualityFilter(&conditions, &args, r, "route =", query.Route)
	appendAccessLogOptionalIntFilter(&conditions, &args, r, "status_code =", query.StatusCode)
	appendAccessLogOptionalInt64Filter(&conditions, &args, r, "duration_ms >=", query.DurationMinMS)
	appendAccessLogOptionalInt64Filter(&conditions, &args, r, "duration_ms <=", query.DurationMaxMS)
	appendAccessLogOptionalTimeFilter(&conditions, &args, r, "started_at >=", query.StartedFrom)
	appendAccessLogOptionalTimeFilter(&conditions, &args, r, "started_at <=", query.StartedTo)
	appendAccessLogOptionalTimeFilter(&conditions, &args, r, "occurred_at >=", query.OccurredFrom)
	appendAccessLogOptionalTimeFilter(&conditions, &args, r, "occurred_at <=", query.OccurredTo)

	if len(conditions) == 0 {
		return "", args
	}

	where := " WHERE " + strings.Join(conditions, " AND ")
	return where, args
}

func appendAccessLogEqualityFilter(
	conditions *[]string,
	args *[]any,
	repo *accessLogRepository,
	operator string,
	value string,
) {
	if value == "" {
		return
	}
	*args = append(*args, value)
	*conditions = append(*conditions, operator+" "+repo.placeholder(len(*args)))
}

func appendAccessLogOptionalUint64Filter(
	conditions *[]string,
	args *[]any,
	repo *accessLogRepository,
	operator string,
	value *uint64,
) {
	if value == nil {
		return
	}
	*args = append(*args, *value)
	*conditions = append(*conditions, operator+" "+repo.placeholder(len(*args)))
}

func appendAccessLogOptionalIntFilter(
	conditions *[]string,
	args *[]any,
	repo *accessLogRepository,
	operator string,
	value *int,
) {
	if value == nil {
		return
	}
	*args = append(*args, *value)
	*conditions = append(*conditions, operator+" "+repo.placeholder(len(*args)))
}

func appendAccessLogOptionalInt64Filter(
	conditions *[]string,
	args *[]any,
	repo *accessLogRepository,
	operator string,
	value *int64,
) {
	if value == nil {
		return
	}
	*args = append(*args, *value)
	*conditions = append(*conditions, operator+" "+repo.placeholder(len(*args)))
}

func appendAccessLogOptionalTimeFilter(
	conditions *[]string,
	args *[]any,
	repo *accessLogRepository,
	operator string,
	value *time.Time,
) {
	if value == nil {
		return
	}
	*args = append(*args, value.UTC())
	*conditions = append(*conditions, operator+" "+repo.placeholder(len(*args)))
}

func appendAccessLogPathFilter(
	conditions *[]string,
	args *[]any,
	repo *accessLogRepository,
	query AccessLogListQuery,
) {
	if query.Path == "" {
		return
	}

	if query.PathMatchMode == AccessLogPathMatchPrefix {
		*args = append(*args, escapeAccessLogLikePattern(query.Path)+"%")
		*conditions = append(*conditions, "path LIKE "+repo.placeholder(len(*args))+" ESCAPE '\\'")
		return
	}
	appendAccessLogEqualityFilter(conditions, args, repo, "path =", query.Path)
}

func escapeAccessLogLikePattern(value string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"%", "\\%",
		"_", "\\_",
	)
	return replacer.Replace(value)
}

type accessLogScanner interface {
	Scan(dest ...any) error
}

func scanAccessLog(scanner accessLogScanner) (AccessLog, error) {
	var (
		id           int64
		traceID      sql.NullString
		route        sql.NullString
		clientIP     sql.NullString
		userAgent    sql.NullString
		userID       sql.NullInt64
		username     sql.NullString
		requestSize  sql.NullInt64
		responseSize sql.NullInt64
		startedAt    time.Time
		record       AccessLog
	)

	if err := scanner.Scan(
		&id,
		&record.RequestID,
		&traceID,
		&record.Method,
		&record.Path,
		&route,
		&record.StatusCode,
		&record.DurationMS,
		&clientIP,
		&userAgent,
		&userID,
		&username,
		&requestSize,
		&responseSize,
		&startedAt,
		&record.OccurredAt,
	); err != nil {
		return AccessLog{}, err
	}
	if id >= 0 {
		record.ID = uint64(id)
	}
	record.TraceID = traceID.String
	record.Route = route.String
	record.ClientIP = clientIP.String
	record.UserAgent = userAgent.String
	record.Username = username.String
	if userID.Valid && userID.Int64 >= 0 {
		value := uint64(userID.Int64)
		record.UserID = &value
	}
	if requestSize.Valid {
		value := requestSize.Int64
		record.RequestSize = &value
	}
	if responseSize.Valid {
		value := responseSize.Int64
		record.ResponseSize = &value
	}
	record.StartedAt = startedAt.UTC()
	record.OccurredAt = record.OccurredAt.UTC()

	return record, nil
}

func nullableString(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func nullableUint64(value *uint64) (any, error) {
	if value == nil {
		return nil, nil
	}
	if *value > math.MaxInt64 {
		return nil, fmt.Errorf("user id %d exceeds bigint range", *value)
	}

	return int64(*value), nil
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
