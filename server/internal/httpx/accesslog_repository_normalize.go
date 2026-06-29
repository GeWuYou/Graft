package httpx

import (
	"strings"
	"time"
)

// normalizeCreateAccessLogInput 将创建访问日志的输入规范化为 AccessLog。
// 它会清洗文本字段、克隆指针字段，并将时间字段统一为 UTC 后返回。
func normalizeCreateAccessLogInput(input CreateAccessLogInput) AccessLog {
	requestID := sanitizeAccessLogStableText(input.RequestID)
	traceID := normalizeAccessLogTraceID(sanitizeAccessLogStableText(input.TraceID), requestID)

	return AccessLog{
		RequestID:    requestID,
		TraceID:      traceID,
		Method:       sanitizeAccessLogStableText(input.Method),
		Path:         sanitizeAccessLogPath(input.Path),
		Route:        sanitizeAccessLogRoute(input.Route),
		StatusCode:   input.StatusCode,
		DurationMS:   input.DurationMS,
		ClientIP:     sanitizeAccessLogStableText(input.ClientIP),
		UserAgent:    sanitizeAccessLogFreeText(input.UserAgent),
		UserID:       cloneUint64Pointer(input.UserID),
		Username:     sanitizeAccessLogStableText(input.Username),
		RequestSize:  cloneInt64Pointer(input.RequestSize),
		ResponseSize: cloneInt64Pointer(input.ResponseSize),
		StartedAt:    normalizeStartedAt(input.StartedAt, input.OccurredAt),
		OccurredAt:   normalizeOccurredAt(input.OccurredAt),
	}
}

// 当 startedAt 为空时，返回规范化后的 occurredAt；否则返回 startedAt 的 UTC 时间。
func normalizeStartedAt(startedAt time.Time, occurredAt time.Time) time.Time {
	if startedAt.IsZero() {
		return normalizeOccurredAt(occurredAt)
	}
	return startedAt.UTC()
}

// normalizeOccurredAt 将发生时间规范化为 UTC。
// 当 occurredAt 为空值时，返回当前 UTC 时间；否则返回 occurredAt 的 UTC 表示。
// @return 规范化后的发生时间。
func normalizeOccurredAt(occurredAt time.Time) time.Time {
	if occurredAt.IsZero() {
		return time.Now().UTC()
	}
	return occurredAt.UTC()
}

// normalizeAccessLogListQuery 规范化访问日志列表查询条件，统一分页、文本字段、路径、分组和排序参数。
// 它会修剪字符串空白，校正页码与页大小，并对状态分组和排序项进行标准化与去重。
func normalizeAccessLogListQuery(query AccessLogListQuery) AccessLogListQuery {
	query.Page = normalizePositivePage(query.Page)
	query.PageSize = normalizePageSize(query.PageSize)
	query.RequestID = strings.TrimSpace(query.RequestID)
	query.TraceID = strings.TrimSpace(query.TraceID)
	query.Keyword = strings.TrimSpace(query.Keyword)
	query.Username = strings.TrimSpace(query.Username)
	query.Method = strings.TrimSpace(query.Method)
	query.Path = sanitizeAccessLogPath(query.Path)
	query.Route = sanitizeAccessLogRoute(query.Route)
	query.PathMatchMode = normalizeAccessLogPathMatchMode(query.PathMatchMode)
	query.StatusGroups = normalizeAccessLogStatusGroups(query.StatusGroups)
	query.Sorts = normalizeAccessLogSorts(query.Sorts)
	return query
}

// normalizeAccessLogTraceID 规范化访问日志的 TraceID。
// 当 traceID 为空或与 requestID 相同时，返回空字符串；否则返回原始 traceID。
func normalizeAccessLogTraceID(traceID string, requestID string) string {
	if traceID == "" || traceID == requestID {
		return ""
	}
	return traceID
}

// normalizePositivePage 将页码规范化为至少 1。
//
// 返回值为大于或等于 1 的页码。
func normalizePositivePage(page int) int {
	if page < 1 {
		return 1
	}
	return page
}

// normalizePageSize 将页大小约束在允许范围内。
// 当 pageSize 小于 1 时返回默认页大小；当 pageSize 大于最大值时返回最大页大小；否则返回原值。
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

// normalizeAccessLogPathMatchMode 规范化访问日志路径匹配模式，未识别的值会被归一为精确匹配。
func normalizeAccessLogPathMatchMode(mode AccessLogPathMatchMode) AccessLogPathMatchMode {
	if mode == AccessLogPathMatchPrefix {
		return AccessLogPathMatchPrefix
	}
	return AccessLogPathMatchExact
}

// normalizeAccessLogSortOrder 规范化访问日志的排序方向。
// 当传入值为升序时返回升序，否则返回降序。
func normalizeAccessLogSortOrder(order AccessLogSortOrder) AccessLogSortOrder {
	if order == AccessLogSortOrderAsc {
		return AccessLogSortOrderAsc
	}
	return AccessLogSortOrderDesc
}

// normalizeAccessLogSortField 将已知的访问日志排序字段规范化为其自身，未知值返回空字符串。
func normalizeAccessLogSortField(field AccessLogSortField) AccessLogSortField {
	switch field {
	case AccessLogSortStartedAt:
		return AccessLogSortStartedAt
	case AccessLogSortDurationMS:
		return AccessLogSortDurationMS
	case AccessLogSortStatusCode:
		return AccessLogSortStatusCode
	case AccessLogSortOccurredAt:
		return AccessLogSortOccurredAt
	default:
		return ""
	}
}

// normalizeAccessLogStatusGroups 规范化访问日志状态分组列表。
//
// 仅保留 `4xx` 和 `5xx` 分组，并去除重复项。
//
// @return 规范化后的状态分组列表；当输入为空时返回 `nil`。
func normalizeAccessLogStatusGroups(groups []AccessLogStatusGroup) []AccessLogStatusGroup {
	if len(groups) == 0 {
		return nil
	}

	normalized := make([]AccessLogStatusGroup, 0, len(groups))
	for _, group := range groups {
		switch AccessLogStatusGroup(strings.TrimSpace(string(group))) {
		case AccessLogStatusGroup4xx:
			normalized = appendAccessLogStatusGroup(normalized, AccessLogStatusGroup4xx)
		case AccessLogStatusGroup5xx:
			normalized = appendAccessLogStatusGroup(normalized, AccessLogStatusGroup5xx)
		}
	}
	return normalized
}

// appendAccessLogStatusGroup 向状态组列表追加一个值，已存在时保持原样。
// @returns 包含该值的状态组列表；若列表中已存在该值，则返回原列表。
func appendAccessLogStatusGroup(groups []AccessLogStatusGroup, value AccessLogStatusGroup) []AccessLogStatusGroup {
	for _, current := range groups {
		if current == value {
			return groups
		}
	}
	return append(groups, value)
}

// normalizeAccessLogSorts 规范化访问日志的排序条件，并去除重复和无效项。
// 返回规范化后的排序条件；当输入为空时返回 nil。
func normalizeAccessLogSorts(sorts []AccessLogSort) []AccessLogSort {
	if len(sorts) == 0 {
		return nil
	}

	normalized := make([]AccessLogSort, 0, len(sorts))
	seen := make(map[AccessLogSortField]struct{}, len(sorts))
	for _, sort := range sorts {
		field := normalizeAccessLogSortField(sort.Field)
		if field == "" {
			continue
		}
		if _, exists := seen[field]; exists {
			continue
		}
		seen[field] = struct{}{}
		normalized = append(normalized, AccessLogSort{
			Field: field,
			Order: normalizeAccessLogSortOrder(sort.Order),
		})
	}
	return normalized
}

// accessLogSortColumn 将访问日志排序字段映射为数据库列名。
func accessLogSortColumn(field AccessLogSortField) string {
	switch field {
	case AccessLogSortStartedAt:
		return "started_at"
	case AccessLogSortDurationMS:
		return "duration_ms"
	case AccessLogSortStatusCode:
		return "status_code"
	case AccessLogSortOccurredAt:
		return "occurred_at"
	default:
		return ""
	}
}

// accessLogSortDirection 将访问日志排序方向映射为 SQL 方向字符串。
// @returns 当排序方向为升序时返回 "ASC"，否则返回 "DESC"。
func accessLogSortDirection(order AccessLogSortOrder) string {
	if normalizeAccessLogSortOrder(order) == AccessLogSortOrderAsc {
		return "ASC"
	}
	return "DESC"
}
