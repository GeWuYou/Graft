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

func escapeAccessLogLikePattern(value string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"%", "\\%",
		"_", "\\_",
	)
	return replacer.Replace(value)
}
