// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package httpx

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

const (
	// AccessLogDashboardWidgetID identifies the core-owned access-log dashboard insight.
	AccessLogDashboardWidgetID = "core.httpx.access-log.request-attention"
	// AccessLogDashboardWidgetOrder keeps access-log attention after module health widgets.
	AccessLogDashboardWidgetOrder = 130
	accessLogSlowRequestThresholdMS  = int64(1000)
	accessLogWidgetRecentLimit       = 2
	accessLogWidgetSourceCount       = 3
	accessLogQueryStatusGroup        = "status_group"
	accessLogQueryDurationMinMS      = "duration_min_ms"
)

// AccessLogDashboardModuleKey returns the core system-capability owner for access-log dashboard data.
func AccessLogDashboardModuleKey() string {
	return accessLogModuleOwner
}

// AccessLogDashboardRouteLocation returns the canonical access-log explorer route.
func AccessLogDashboardRouteLocation() string {
	return accessLogMenuListPath
}

// LoadAccessLogRequestAttentionPayload loads access log attention data (4xx errors, 5xx errors, and slow requests) for dashboard display.
//
// Returns a map containing the aggregated attention items with visibility and severity metadata.
func LoadAccessLogRequestAttentionPayload(ctx context.Context, repo AccessLogRepository) (map[string]any, error) {
	clientErrorsResult, err := repo.ListAccessLogs(ctx, AccessLogListQuery{
		Page:         1,
		PageSize:     accessLogWidgetRecentLimit,
		StatusGroups: []AccessLogStatusGroup{AccessLogStatusGroup4xx},
		Sorts:        []AccessLogSort{{Field: AccessLogSortOccurredAt, Order: AccessLogSortOrderDesc}},
	})
	if err != nil {
		return nil, fmt.Errorf("load access log dashboard 4xx requests: %w", err)
	}

	serverErrorsResult, err := repo.ListAccessLogs(ctx, AccessLogListQuery{
		Page:         1,
		PageSize:     accessLogWidgetRecentLimit,
		StatusGroups: []AccessLogStatusGroup{AccessLogStatusGroup5xx},
		Sorts:        []AccessLogSort{{Field: AccessLogSortOccurredAt, Order: AccessLogSortOrderDesc}},
	})
	if err != nil {
		return nil, fmt.Errorf("load access log dashboard 5xx requests: %w", err)
	}

	slowResult, err := repo.ListAccessLogs(ctx, AccessLogListQuery{
		Page:          1,
		PageSize:      accessLogWidgetRecentLimit,
		DurationMinMS: int64Pointer(accessLogSlowRequestThresholdMS),
		Sorts:         []AccessLogSort{{Field: AccessLogSortDurationMS, Order: AccessLogSortOrderDesc}},
	})
	if err != nil {
		return nil, fmt.Errorf("load access log dashboard slow requests: %w", err)
	}

	items := make([]map[string]any, 0, accessLogWidgetRecentLimit*accessLogWidgetSourceCount)
	items = appendAccessLogStatusGroupItem(items, "error.4xx", clientErrorsResult, AccessLogStatusGroup4xx)
	items = appendAccessLogStatusGroupItem(items, "error.5xx", serverErrorsResult, AccessLogStatusGroup5xx)
	items = appendAccessLogSlowRequestItem(items, slowResult)

	visible := len(items) > 0
	state := "hidden"
	priority := "warning"
	if visible {
		state = "warning"
	}
	for _, item := range items {
		if item["level"] == "error" {
			state = "critical"
			priority = "critical"
			break
		}
	}

	return map[string]any{
		"items":     items,
		"empty_key": "dashboard.widget.accessLogRequestAttention.empty",
		"visible":   visible,
		"state":     state,
		"priority":  priority,
	}, nil
}

func appendAccessLogStatusGroupItem(
	items []map[string]any,
	id string,
	result AccessLogListResult,
	statusGroup AccessLogStatusGroup,
) []map[string]any {
	if result.Total <= 0 || len(result.Items) == 0 {
		return items
	}
	record := result.Items[0]
	return append(items, accessLogAlertItem(accessLogAlertItemDefinition{
		count:         int(result.Total),
		id:            id,
		level:         "error",
		record:        record,
		routeLocation: accessLogDashboardRouteLocation(url.Values{accessLogQueryStatusGroup: []string{string(statusGroup)}}),
		TitleKey:      "dashboard.widget.accessLogRequestAttention.error",
	}))
}

func appendAccessLogSlowRequestItem(items []map[string]any, result AccessLogListResult) []map[string]any {
	if result.Total <= 0 || len(result.Items) == 0 {
		return items
	}
	record := result.Items[0]
	return append(items, accessLogAlertItem(accessLogAlertItemDefinition{
		count:  int(result.Total),
		id:     "slow",
		level:  "warning",
		record: record,
		routeLocation: accessLogDashboardRouteLocation(url.Values{
			accessLogQueryDurationMinMS: []string{strconv.FormatInt(accessLogSlowRequestThresholdMS, 10)},
		}),
		TitleKey: "dashboard.widget.accessLogRequestAttention.slow",
	}))
}

type accessLogAlertItemDefinition struct {
	count         int
	id            string
	level         string
	record        AccessLog
	routeLocation string
	TitleKey      string
}

func accessLogAlertItem(definition accessLogAlertItemDefinition) map[string]any {
	occurredAt := definition.record.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}

	return map[string]any{
		"id":             definition.id,
		"level":          definition.level,
		"title_key":      definition.TitleKey,
		"description":    definition.record.Method + " " + definition.record.Path + " -> " + strconv.Itoa(definition.record.StatusCode) + " in " + strconv.FormatInt(definition.record.DurationMS, 10) + "ms",
		"count":          definition.count,
		"occurred_at":    occurredAt,
		"route_location": definition.routeLocation,
	}
}

func accessLogDashboardRouteLocation(query url.Values) string {
	if len(query) == 0 {
		return accessLogMenuListPath
	}
	return accessLogMenuListPath + "?" + query.Encode()
}

func int64Pointer(value int64) *int64 {
	return &value
}
