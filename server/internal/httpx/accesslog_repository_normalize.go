package httpx

import (
	"strings"
	"time"
)

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

func normalizeAccessLogSortOrder(order AccessLogSortOrder) AccessLogSortOrder {
	if order == AccessLogSortOrderAsc {
		return AccessLogSortOrderAsc
	}
	return AccessLogSortOrderDesc
}

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

func appendAccessLogStatusGroup(groups []AccessLogStatusGroup, value AccessLogStatusGroup) []AccessLogStatusGroup {
	for _, current := range groups {
		if current == value {
			return groups
		}
	}
	return append(groups, value)
}

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

func accessLogSortDirection(order AccessLogSortOrder) string {
	if normalizeAccessLogSortOrder(order) == AccessLogSortOrderAsc {
		return "ASC"
	}
	return "DESC"
}
