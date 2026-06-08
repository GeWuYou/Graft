package httpx

import (
	"context"
	"strconv"
	"time"
)

const (
	// AccessLogDashboardWidgetID identifies the core-owned access-log dashboard insight.
	AccessLogDashboardWidgetID = "core.httpx.access-log.request-attention"
	// AccessLogDashboardWidgetOrder keeps access-log attention after module health widgets.
	AccessLogDashboardWidgetOrder   = 130
	accessLogSlowRequestThresholdMS = int64(1000)
	accessLogWidgetRecentLimit      = 2
	accessLogWidgetSourceCount      = 2
)

// AccessLogDashboardModuleKey returns the core system-capability owner for access-log dashboard data.
func AccessLogDashboardModuleKey() string {
	return accessLogModuleOwner
}

// AccessLogDashboardRouteLocation returns the canonical access-log explorer route.
func AccessLogDashboardRouteLocation() string {
	return accessLogMenuListPath
}

// LoadAccessLogRequestAttentionPayload returns access-log attention data without depending on dashboard internals.
func LoadAccessLogRequestAttentionPayload(ctx context.Context, repo AccessLogRepository) (map[string]any, error) {
	errorsResult, err := repo.ListAccessLogs(ctx, AccessLogListQuery{
		Page:         1,
		PageSize:     accessLogWidgetRecentLimit,
		StatusGroups: []AccessLogStatusGroup{AccessLogStatusGroup4xx, AccessLogStatusGroup5xx},
		Sorts:        []AccessLogSort{{Field: AccessLogSortOccurredAt, Order: AccessLogSortOrderDesc}},
	})
	if err != nil {
		return nil, err
	}

	slowResult, err := repo.ListAccessLogs(ctx, AccessLogListQuery{
		Page:          1,
		PageSize:      accessLogWidgetRecentLimit,
		DurationMinMS: int64Pointer(accessLogSlowRequestThresholdMS),
		Sorts:         []AccessLogSort{{Field: AccessLogSortDurationMS, Order: AccessLogSortOrderDesc}},
	})
	if err != nil {
		return nil, err
	}

	items := make([]map[string]any, 0, accessLogWidgetRecentLimit*accessLogWidgetSourceCount)
	for _, record := range errorsResult.Items {
		items = append(items, accessLogAlertItem("error", record, "HTTP error request", "error"))
	}
	for _, record := range slowResult.Items {
		items = append(items, accessLogAlertItem("slow", record, "Slow HTTP request", "warning"))
	}

	return map[string]any{
		"items":     items,
		"empty_key": "dashboard.widget.accessLogRequestAttention.empty",
		"empty":     "No recent error or slow requests.",
	}, nil
}

func accessLogAlertItem(prefix string, record AccessLog, title string, level string) map[string]any {
	occurredAt := record.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}

	return map[string]any{
		"id":             prefix + "." + strconv.FormatUint(record.ID, 10),
		"level":          level,
		"title_key":      "dashboard.widget.accessLogRequestAttention." + prefix,
		"title":          title,
		"description":    record.Method + " " + record.Path + " -> " + strconv.Itoa(record.StatusCode) + " in " + strconv.FormatInt(record.DurationMS, 10) + "ms",
		"occurred_at":    occurredAt,
		"route_location": accessLogMenuListPath,
	}
}

func int64Pointer(value int64) *int64 {
	return &value
}
