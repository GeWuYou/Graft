package httpx

import (
	"fmt"
	"strings"
	"time"
)

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
	builder.WriteByte(' ')
	builder.WriteString(buildAccessLogOrderByClause(query.Sorts))
	builder.WriteString(" LIMIT ")
	builder.WriteString(r.placeholder(filterArgCount + 1))
	builder.WriteString(" OFFSET ")
	builder.WriteString(r.placeholder(filterArgCount + accessLogListOffsetArgCount))
	return builder.String()
}

// buildAccessLogOrderByClause 生成访问日志列表查询的 ORDER BY 子句。
// 当未提供有效排序时，使用 `occurred_at DESC, id DESC` 作为默认排序。
func buildAccessLogOrderByClause(sorts []AccessLogSort) string {
	normalized := normalizeAccessLogSorts(sorts)
	if len(normalized) == 0 {
		return "ORDER BY occurred_at DESC, id DESC"
	}

	clauses := make([]string, 0, len(normalized)+1)
	for _, sort := range normalized {
		column := accessLogSortColumn(sort.Field)
		if column == "" {
			continue
		}
		clauses = append(clauses, column+" "+accessLogSortDirection(sort.Order))
	}
	if len(clauses) == 0 {
		return "ORDER BY occurred_at DESC, id DESC"
	}
	clauses = append(clauses, "id DESC")
	return "ORDER BY " + strings.Join(clauses, ", ")
}

func (r *accessLogRepository) buildAccessLogWhereClause(query AccessLogListQuery) (string, []any) {
	conditions := make([]string, 0, accessLogListClauseCapacity)
	args := make([]any, 0, accessLogListClauseCapacity)

	appendAccessLogEqualityFilter(&conditions, &args, r, "request_id =", query.RequestID)
	appendAccessLogEqualityFilter(&conditions, &args, r, "trace_id =", query.TraceID)
	appendAccessLogKeywordFilter(&conditions, &args, r, query.Keyword)
	appendAccessLogOptionalUint64Filter(&conditions, &args, r, "user_id =", query.UserID)
	appendAccessLogEqualityFilter(&conditions, &args, r, "username =", query.Username)
	appendAccessLogEqualityFilter(&conditions, &args, r, "method =", query.Method)
	appendAccessLogPathFilter(&conditions, &args, r, query)
	appendAccessLogEqualityFilter(&conditions, &args, r, "route =", query.Route)
	appendAccessLogOptionalIntFilter(&conditions, &args, r, "status_code =", query.StatusCode)
	appendAccessLogStatusGroupFilter(&conditions, &args, r, query.StatusGroups)
	appendAccessLogOptionalInt64Filter(&conditions, &args, r, "duration_ms >=", query.DurationMinMS)
	appendAccessLogOptionalInt64Filter(&conditions, &args, r, "duration_ms <=", query.DurationMaxMS)
	appendAccessLogOptionalTimeFilter(&conditions, &args, r, "started_at >=", query.StartedFrom)
	appendAccessLogOptionalTimeFilter(&conditions, &args, r, "started_at <=", query.StartedTo)
	appendAccessLogOptionalTimeFilter(&conditions, &args, r, "occurred_at >=", query.OccurredFrom)
	appendAccessLogOptionalTimeFilter(&conditions, &args, r, "occurred_at <=", query.OccurredTo)

	if len(conditions) == 0 {
		return "", args
	}

	return " WHERE " + strings.Join(conditions, " AND "), args
}

// appendAccessLogEqualityFilter 在值非空时追加等值筛选条件及其绑定参数。
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

// appendAccessLogOptionalUint64Filter 在值存在时追加一个 uint64 条件及其绑定参数。
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

// appendAccessLogOptionalIntFilter 根据可选整数值追加 SQL 条件和绑定参数。
// 当 value 为 nil 时不追加任何内容；否则会将其加入 args，并向 conditions 添加一条使用占位符的条件。
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

// appendAccessLogOptionalInt64Filter appends an int64 filter when a value is provided.
// If value is nil, it leaves conditions and args unchanged.
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

// appendAccessLogOptionalTimeFilter 在值存在时追加时间比较条件，并将参数转换为 UTC。
// 当 value 为 nil 时不添加任何条件或参数。
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

// appendAccessLogPathFilter 为访问日志路径添加过滤条件。
// 当匹配模式为前缀匹配时，使用带转义的 LIKE 条件；否则使用精确匹配。
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

// appendAccessLogKeywordFilter 添加访问日志关键字过滤条件。
// 关键字会同时匹配 request_id、path 和 username，并以大小写不敏感的方式进行包含匹配。
func appendAccessLogKeywordFilter(
	conditions *[]string,
	args *[]any,
	repo *accessLogRepository,
	keyword string,
) {
	trimmed := strings.ToLower(strings.TrimSpace(keyword))
	if trimmed == "" {
		return
	}

	pattern := "%" + escapeAccessLogLikePattern(trimmed) + "%"
	orClauses := make([]string, 0, accessLogKeywordClauseCount)
	for _, expression := range []string{
		"LOWER(request_id) LIKE %s ESCAPE '\\'",
		"LOWER(path) LIKE %s ESCAPE '\\'",
		"LOWER(COALESCE(username, '')) LIKE %s ESCAPE '\\'",
	} {
		*args = append(*args, pattern)
		orClauses = append(orClauses, fmt.Sprintf(expression, repo.placeholder(len(*args))))
	}
	*conditions = append(*conditions, "("+strings.Join(orClauses, " OR ")+")")
}

// appendAccessLogStatusGroupFilter 将状态组条件追加到查询条件中。
// 将 4xx 和 5xx 状态组展开为 `status_code` 的区间条件，并以 `OR` 组合后追加到 `conditions`，同时将对应的绑定参数追加到 `args`。
func appendAccessLogStatusGroupFilter(
	conditions *[]string,
	args *[]any,
	repo *accessLogRepository,
	groups []AccessLogStatusGroup,
) {
	if len(groups) == 0 {
		return
	}

	orClauses := make([]string, 0, len(groups))
	for _, group := range groups {
		switch group {
		case AccessLogStatusGroup4xx:
			*args = append(*args, accessLogStatus4xxMin, accessLogStatus4xxMax)
			orClauses = append(orClauses, fmt.Sprintf(
				"(status_code >= %s AND status_code <= %s)",
				repo.placeholder(len(*args)-1),
				repo.placeholder(len(*args)),
			))
		case AccessLogStatusGroup5xx:
			*args = append(*args, accessLogStatus5xxMin, accessLogStatus5xxMax)
			orClauses = append(orClauses, fmt.Sprintf(
				"(status_code >= %s AND status_code <= %s)",
				repo.placeholder(len(*args)-1),
				repo.placeholder(len(*args)),
			))
		}
	}
	if len(orClauses) > 0 {
		*conditions = append(*conditions, "("+strings.Join(orClauses, " OR ")+")")
	}
}

// escapeAccessLogLikePattern 转义用于 SQL LIKE 的模式字符。
// 它会将反斜杠、百分号和下划线替换为带转义前缀的形式，便于配合 ESCAPE '\' 使用。
// 返回转义后的字符串。
func escapeAccessLogLikePattern(value string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"%", "\\%",
		"_", "\\_",
	)
	return replacer.Replace(value)
}
