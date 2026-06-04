package logger

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type appLogSQLDialect string

const (
	appLogSQLDialectPostgres appLogSQLDialect = "postgres"
	appLogSQLDialectSQLite   appLogSQLDialect = "sqlite"
	appLogListClauseCapacity                  = 10
	appLogListOffsetArgCount                  = 2
	appLogKeywordClauseCount                  = 4
)

type appLogRepository struct {
	db      *sql.DB
	dialect appLogSQLDialect
}

type appLogQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// NewAppLogRepository builds the logger-owned SQL repository for app-log persistence.
func NewAppLogRepository(db *sql.DB) (AppLogRepository, error) {
	return newAppLogRepositoryWithDialect(db, appLogSQLDialectPostgres)
}

func newAppLogRepositoryWithDialect(db *sql.DB, dialect appLogSQLDialect) (AppLogRepository, error) {
	if db == nil {
		return nil, errors.New("app log repository requires a non-nil sql db")
	}
	if dialect == "" {
		dialect = appLogSQLDialectPostgres
	}

	return &appLogRepository{db: db, dialect: dialect}, nil
}

func (r *appLogRepository) CreateAppLog(ctx context.Context, input CreateAppLogInput) (AppLogRecord, error) {
	if r == nil || r.db == nil {
		return AppLogRecord{}, errors.New("app log repository is unavailable")
	}

	return r.createAppLog(ctx, r.db, input)
}

func (r *appLogRepository) DeleteAppLogsBefore(ctx context.Context, occurredBefore time.Time) (int64, error) {
	if r == nil || r.db == nil {
		return 0, errors.New("app log repository is unavailable")
	}

	result, err := r.db.ExecContext(
		ctx,
		fmt.Sprintf("DELETE FROM app_logs WHERE occurred_at < %s", r.placeholder(1)),
		occurredBefore.UTC(),
	)
	if err != nil {
		return 0, fmt.Errorf("delete app logs before cutoff: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("read deleted app log row count: %w", err)
	}

	return rowsAffected, nil
}

func (r *appLogRepository) ListAppLogs(ctx context.Context, query AppLogListQuery) (AppLogListResult, error) {
	if r == nil || r.db == nil {
		return AppLogListResult{}, errors.New("app log repository is unavailable")
	}

	normalized := normalizeAppLogListQuery(query)
	whereSQL, args := r.buildAppLogWhereClause(normalized)

	var total int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM app_logs"+whereSQL, args...).Scan(&total); err != nil {
		return AppLogListResult{}, fmt.Errorf("count app logs: %w", err)
	}

	listArgs := append([]any(nil), args...)
	listArgs = append(listArgs, normalized.PageSize, (normalized.Page-1)*normalized.PageSize)
	selectQuery := r.buildAppLogListSelectQuery(whereSQL, normalized.Sorters, len(args))

	rows, err := r.db.QueryContext(ctx, selectQuery, listArgs...)
	if err != nil {
		return AppLogListResult{}, fmt.Errorf("list app logs: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	items := make([]AppLogRecord, 0, normalized.PageSize)
	for rows.Next() {
		record, scanErr := scanAppLog(rows)
		if scanErr != nil {
			return AppLogListResult{}, fmt.Errorf("scan app log: %w", scanErr)
		}
		items = append(items, record)
	}
	if err := rows.Err(); err != nil {
		return AppLogListResult{}, fmt.Errorf("iterate app logs: %w", err)
	}

	return AppLogListResult{
		Items:    items,
		Total:    total,
		Page:     normalized.Page,
		PageSize: normalized.PageSize,
	}, nil
}

func (r *appLogRepository) GetAppLogByID(ctx context.Context, id uint64) (AppLogRecord, error) {
	if r == nil || r.db == nil {
		return AppLogRecord{}, errors.New("app log repository is unavailable")
	}
	if id == 0 {
		return AppLogRecord{}, ErrAppLogNotFound
	}

	query := r.buildAppLogDetailSelectQuery()
	record, err := scanAppLog(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AppLogRecord{}, ErrAppLogNotFound
		}
		return AppLogRecord{}, fmt.Errorf("get app log by id: %w", err)
	}

	return record, nil
}

func (r *appLogRepository) createAppLog(
	ctx context.Context,
	queryer appLogQueryer,
	input CreateAppLogInput,
) (AppLogRecord, error) {
	record, err := normalizeCreateAppLogInput(input)
	if err != nil {
		return AppLogRecord{}, err
	}
	fieldsJSON, err := marshalAppLogFields(record.Fields)
	if err != nil {
		return AppLogRecord{}, err
	}

	args := []any{
		record.OccurredAt,
		string(record.Severity),
		record.Component,
		nullableString(record.Operation),
		nullableString(record.RequestID),
		nullableString(record.TraceID),
		nullableString(record.Route),
		nullableString(record.Method),
		nullableString(record.Error),
		record.Message,
		fieldsJSON,
	}

	query := fmt.Sprintf(`INSERT INTO app_logs (
		occurred_at,
		severity,
		component,
		operation,
		request_id,
		trace_id,
		route,
		method,
		error,
		message,
		fields
	) VALUES (%s) RETURNING id`, r.placeholders(len(args)))

	var id int64
	if err := queryer.QueryRowContext(ctx, query, args...).Scan(&id); err != nil {
		return AppLogRecord{}, fmt.Errorf("create app log: %w", err)
	}
	if id < 0 {
		return AppLogRecord{}, fmt.Errorf("create app log: negative id %d", id)
	}

	record.ID = uint64(id)
	return record, nil
}

func normalizeCreateAppLogInput(input CreateAppLogInput) (AppLogRecord, error) {
	input.ID = 0
	if input.OccurredAt.IsZero() {
		input.OccurredAt = time.Now().UTC()
	}
	return input.Normalize()
}

func (r *appLogRepository) buildAppLogListSelectQuery(
	whereSQL string,
	sorters []AppLogSorter,
	filterArgCount int,
) string {
	var builder strings.Builder
	builder.WriteString(`SELECT
		id,
		occurred_at,
		severity,
		component,
		operation,
		request_id,
		trace_id,
		route,
		method,
		error,
		message,
		fields
	FROM app_logs`)
	builder.WriteString(whereSQL)
	builder.WriteString(" ORDER BY ")
	builder.WriteString(buildAppLogOrderByClause(sorters))
	builder.WriteString(" LIMIT ")
	builder.WriteString(r.placeholder(filterArgCount + 1))
	builder.WriteString(" OFFSET ")
	builder.WriteString(r.placeholder(filterArgCount + appLogListOffsetArgCount))
	return builder.String()
}

func buildAppLogOrderByClause(sorters []AppLogSorter) string {
	if len(sorters) == 0 {
		return "occurred_at DESC, id DESC"
	}

	parts := make([]string, 0, len(sorters)+1)
	for _, sorter := range sorters {
		column := appLogSortColumn(sorter.Field)
		if column == "" {
			continue
		}
		order := "DESC"
		if sorter.Order == AppLogSortOrderAsc {
			order = "ASC"
		}
		parts = append(parts, column+" "+order)
	}

	parts = append(parts, "id DESC")
	return strings.Join(parts, ", ")
}

func appLogSortColumn(field AppLogSortField) string {
	switch field {
	case AppLogSortFieldOccurredAt:
		return "occurred_at"
	case AppLogSortFieldSeverity:
		return "severity"
	case AppLogSortFieldComponent:
		return "component"
	default:
		return ""
	}
}

func (r *appLogRepository) buildAppLogDetailSelectQuery() string {
	return `SELECT
		id,
		occurred_at,
		severity,
		component,
		operation,
		request_id,
		trace_id,
		route,
		method,
		error,
		message,
		fields
	FROM app_logs WHERE id = ` + r.placeholder(1)
}

func (r *appLogRepository) buildAppLogWhereClause(query AppLogListQuery) (string, []any) {
	conditions := make([]string, 0, appLogListClauseCapacity)
	args := make([]any, 0, appLogListClauseCapacity)

	appendAppLogEqualityFilter(&conditions, &args, r, "severity =", string(query.Severity))
	appendAppLogEqualityFilter(&conditions, &args, r, "component =", query.Component)
	appendAppLogEqualityFilter(&conditions, &args, r, "operation =", query.Operation)
	appendAppLogEqualityFilter(&conditions, &args, r, "request_id =", query.RequestID)
	appendAppLogEqualityFilter(&conditions, &args, r, "trace_id =", query.TraceID)
	appendAppLogEqualityFilter(&conditions, &args, r, "route =", query.Route)
	appendAppLogEqualityFilter(&conditions, &args, r, "method =", query.Method)
	appendAppLogErrorFilter(&conditions, &args, r, query.Error)
	appendAppLogMessageFilter(&conditions, &args, r, query.Message)
	appendAppLogKeywordFilter(&conditions, &args, r, query.Keyword)
	appendAppLogOptionalTimeFilter(&conditions, &args, r, "occurred_at >=", query.OccurredFrom)
	appendAppLogOptionalTimeFilter(&conditions, &args, r, "occurred_at <=", query.OccurredTo)

	if len(conditions) == 0 {
		return "", args
	}

	return " WHERE " + strings.Join(conditions, " AND "), args
}

func appendAppLogEqualityFilter(
	conditions *[]string,
	args *[]any,
	repo *appLogRepository,
	operator string,
	value string,
) {
	if value == "" {
		return
	}
	*args = append(*args, value)
	*conditions = append(*conditions, operator+" "+repo.placeholder(len(*args)))
}

func appendAppLogErrorFilter(
	conditions *[]string,
	args *[]any,
	repo *appLogRepository,
	value string,
) {
	if value == "" {
		return
	}
	*args = append(*args, "%"+escapeAppLogLikePattern(strings.ToLower(value))+"%")
	*conditions = append(*conditions, "LOWER(COALESCE(error, '')) LIKE "+repo.placeholder(len(*args))+" ESCAPE '\\'")
}

func appendAppLogMessageFilter(
	conditions *[]string,
	args *[]any,
	repo *appLogRepository,
	value string,
) {
	if value == "" {
		return
	}
	*args = append(*args, "%"+escapeAppLogLikePattern(strings.ToLower(value))+"%")
	*conditions = append(*conditions, "LOWER(message) LIKE "+repo.placeholder(len(*args))+" ESCAPE '\\'")
}

func appendAppLogKeywordFilter(
	conditions *[]string,
	args *[]any,
	repo *appLogRepository,
	keyword string,
) {
	trimmed := strings.ToLower(strings.TrimSpace(keyword))
	if trimmed == "" {
		return
	}

	pattern := "%" + escapeAppLogLikePattern(trimmed) + "%"
	orClauses := make([]string, 0, appLogKeywordClauseCount)
	for _, expression := range []string{
		"LOWER(component) LIKE %s ESCAPE '\\'",
		"LOWER(COALESCE(operation, '')) LIKE %s ESCAPE '\\'",
		"LOWER(message) LIKE %s ESCAPE '\\'",
		"LOWER(COALESCE(error, '')) LIKE %s ESCAPE '\\'",
	} {
		*args = append(*args, pattern)
		orClauses = append(orClauses, fmt.Sprintf(expression, repo.placeholder(len(*args))))
	}
	*conditions = append(*conditions, "("+strings.Join(orClauses, " OR ")+")")
}

func appendAppLogOptionalTimeFilter(
	conditions *[]string,
	args *[]any,
	repo *appLogRepository,
	operator string,
	value *time.Time,
) {
	if value == nil {
		return
	}
	*args = append(*args, value.UTC())
	*conditions = append(*conditions, operator+" "+repo.placeholder(len(*args)))
}

func (r *appLogRepository) placeholders(count int) string {
	values := make([]string, 0, count)
	for index := 1; index <= count; index++ {
		values = append(values, r.placeholder(index))
	}
	return strings.Join(values, ", ")
}

func (r *appLogRepository) placeholder(index int) string {
	if r != nil && r.dialect == appLogSQLDialectSQLite {
		return "?"
	}
	return "$" + strconv.Itoa(index)
}

func escapeAppLogLikePattern(value string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"%", "\\%",
		"_", "\\_",
	)
	return replacer.Replace(value)
}

func marshalAppLogFields(fields map[string]string) (string, error) {
	if fields == nil {
		fields = map[string]string{}
	}
	payload, err := json.Marshal(fields)
	if err != nil {
		return "", fmt.Errorf("marshal app log fields: %w", err)
	}
	return string(payload), nil
}

type appLogScanner interface {
	Scan(dest ...any) error
}

func scanAppLog(scanner appLogScanner) (AppLogRecord, error) {
	var (
		id         int64
		severity   string
		operation  sql.NullString
		requestID  sql.NullString
		traceID    sql.NullString
		route      sql.NullString
		method     sql.NullString
		errText    sql.NullString
		fieldsJSON string
		record     AppLogRecord
	)

	if err := scanner.Scan(
		&id,
		&record.OccurredAt,
		&severity,
		&record.Component,
		&operation,
		&requestID,
		&traceID,
		&route,
		&method,
		&errText,
		&record.Message,
		&fieldsJSON,
	); err != nil {
		return AppLogRecord{}, err
	}
	if id >= 0 {
		record.ID = uint64(id)
	}
	record.OccurredAt = record.OccurredAt.UTC()
	record.Severity = AppLogSeverity(severity)
	record.Operation = operation.String
	record.RequestID = requestID.String
	record.TraceID = traceID.String
	record.Route = route.String
	record.Method = method.String
	record.Error = errText.String
	record.Fields = map[string]string{}
	if strings.TrimSpace(fieldsJSON) != "" {
		if err := json.Unmarshal([]byte(fieldsJSON), &record.Fields); err != nil {
			return AppLogRecord{}, fmt.Errorf("decode app log fields: %w", err)
		}
	}

	return record, nil
}

func nullableString(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}
