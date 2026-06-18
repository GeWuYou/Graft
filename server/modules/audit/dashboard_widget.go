// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package audit

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"graft/server/internal/dashboard"
	"graft/server/internal/module"
	auditcontract "graft/server/modules/audit/contract"
	auditstore "graft/server/modules/audit/store"
)

const (
	auditRiskEventsWidgetID     = "audit.risk-events"
	auditRiskEventsWidgetOrder  = 100
	auditRiskEventsItemCap      = 3
	auditLogsQueryPreset        = "preset"
	auditLogsQueryBusiness      = "business_category"
	auditLogsQueryResults       = "results"
	auditLogsQueryRiskLevels    = "risk_levels"
)

// Registers an audit risk events dashboard widget.
// It returns nil if ctx or ctx.DashboardRegistry is nil.
// It returns an error if widget registration fails.
func registerAuditDashboardWidget(ctx *module.Context, reader *Service) error {
	if ctx == nil || ctx.DashboardRegistry == nil {
		return nil
	}

	if err := ctx.DashboardRegistry.Register(dashboard.WidgetDefinition{
		ID:             auditRiskEventsWidgetID,
		ModuleKey:      moduleID,
		TitleKey:       "dashboard.widget.auditRiskEvents.title",
		DescriptionKey: "dashboard.widget.auditRiskEvents.description",
		Type:           dashboard.WidgetTypeAlertList,
		Size:           dashboard.WidgetSizeMedium,
		Category:       dashboard.WidgetCategorySecurity,
		Priority:       dashboard.WidgetPriorityWarning,
		Order:          auditRiskEventsWidgetOrder,
		RouteLocation:  auditcontract.AuditOverviewMenuPath,
		Action: dashboard.WidgetAction{
			LabelKey: "dashboard.actions.details",
			Route:    auditcontract.AuditOverviewMenuPath,
		},
		RequiredPermissions: []string{auditcontract.AuditReadPermission.String()},
		Loader: dashboard.WidgetLoaderFunc(func(ctx context.Context, _ dashboard.WidgetRequest) (dashboard.WidgetPayload, error) {
			return loadAuditRiskEventsWidget(ctx, reader)
		}),
	}); err != nil {
		return fmt.Errorf("register audit dashboard widget: %w", err)
	}

	return nil
}
// LoadAuditRiskEventsWidget builds the audit risk events dashboard widget payload for the last 24 hours, including alert items for high-risk events, failed operations, and failed authentications.
func loadAuditRiskEventsWidget(ctx context.Context, reader *Service) (dashboard.WidgetPayload, error) {
	overview, err := reader.Overview(ctx, auditstore.AuditTimePresetLast24Hours)
	if err != nil {
		return nil, err
	}

	items := make([]map[string]any, 0, auditRiskEventsItemCap)
	if overview.Summary.HighRiskEvents > 0 {
		items = append(items, map[string]any{
			"id":               "audit.high-risk",
			"level":            "error",
			"title_key":        "dashboard.widget.auditRiskEvents.highRisk.title",
			"description_key":  "dashboard.widget.auditRiskEvents.highRisk.description",
			"description":      strconv.Itoa(overview.Summary.HighRiskEvents) + " high-risk events in the last 24 hours.",
			"count":            overview.Summary.HighRiskEvents,
			"action_label_key": "dashboard.widget.auditRiskEvents.highRisk.action",
			"route_location":   auditHighRiskDashboardLocation(),
		})
	}
	if overview.Summary.FailedOperations > 0 {
		items = append(items, map[string]any{
			"id":               "audit.failed-operations",
			"level":            "warning",
			"title_key":        "dashboard.widget.auditRiskEvents.failedOperations.title",
			"description_key":  "dashboard.widget.auditRiskEvents.failedOperations.description",
			"description":      strconv.Itoa(overview.Summary.FailedOperations) + " failed operations need review.",
			"count":            overview.Summary.FailedOperations,
			"action_label_key": "dashboard.widget.auditRiskEvents.failedOperations.action",
			"route_location":   auditFailedOperationsDashboardLocation(),
		})
	}
	riskGroupCounts := auditRiskGroupCounts(overview.RiskGroups)
	items = appendAuditOverviewGroupItem(items, auditOverviewGroupItemDefinition{
		count:          riskGroupCounts[auditstore.AuditBusinessCategoryAuthFailures],
		id:             "audit.failed-auth",
		items:          overview.FailedAuth,
		scope:          auditstore.AuditBusinessCategoryAuthFailures,
		TitleKey:       "audit.overview.riskGroups.authFailures",
		DescriptionKey: "dashboard.widget.auditRiskEvents.authFailures.description",
		ActionLabelKey: "dashboard.widget.auditRiskEvents.authFailures.action",
	})

	highRiskEvents := overview.Summary.HighRiskEvents
	state := dashboard.WidgetStateHidden
	priority := dashboard.WidgetPriorityWarning
	if len(items) > 0 {
		state = dashboard.WidgetStateWarning
	}
	if highRiskEvents > 0 {
		state = dashboard.WidgetStateCritical
		priority = dashboard.WidgetPriorityCritical
	}

	return dashboard.WidgetPayload{
		"items":            items,
		"empty_key":        "dashboard.widget.auditRiskEvents.empty",
		"visible":          len(items) > 0,
		"state":            string(state),
		"priority":         string(priority),
		"high_risk_events": highRiskEvents,
	}, nil
}

type auditOverviewGroupItemDefinition struct {
	count          int
	id             string
	items          []auditstore.OverviewItem
	scope          auditstore.AuditBusinessCategory
	TitleKey       string
	DescriptionKey string
	ActionLabelKey string
}

func appendAuditOverviewGroupItem(items []map[string]any, definition auditOverviewGroupItemDefinition) []map[string]any {
	if definition.count <= 0 || len(definition.items) == 0 {
		return items
	}
	latest := definition.items[0]
	description := latest.Message
	if description == "" {
		description = latest.Action
	}
	item := map[string]any{
		"id":               definition.id,
		"level":            "warning",
		"title_key":        definition.TitleKey,
		"description_key":  definition.DescriptionKey,
		"description":      description,
		"count":            definition.count,
		"occurred_at":      latest.CreatedAt,
		"action_label_key": definition.ActionLabelKey,
		"route_location":   auditBusinessCategoryDashboardLocation(definition.scope),
	}
	return append(items, item)
}

func auditDashboardPresetQuery() url.Values {
	query := url.Values{}
	query.Set(auditLogsQueryPreset, string(auditstore.AuditTimePresetLast24Hours))
	return query
}

func auditLogsDashboardLocation(query url.Values) string {
	if len(query) == 0 {
		return auditcontract.AuditLogsMenuPath
	}
	return auditcontract.AuditLogsMenuPath + "?" + query.Encode()
}

func auditBusinessCategoryDashboardLocation(category auditstore.AuditBusinessCategory) string {
	query := auditDashboardPresetQuery()
	query.Set(auditLogsQueryBusiness, string(category))
	return auditLogsDashboardLocation(query)
}

func auditHighRiskDashboardLocation() string {
	query := auditDashboardPresetQuery()
	query.Set(auditLogsQueryRiskLevels, strings.Join([]string{
		string(auditstore.AuditRiskLevelHigh),
		string(auditstore.AuditRiskLevelCritical),
	}, ","))
	return auditLogsDashboardLocation(query)
}

func auditFailedOperationsDashboardLocation() string {
	query := auditDashboardPresetQuery()
	query.Set(auditLogsQueryResults, strings.Join([]string{
		string(auditstore.AuditResultFailed),
		string(auditstore.AuditResultDenied),
		string(auditstore.AuditResultError),
	}, ","))
	return auditLogsDashboardLocation(query)
}

func auditRiskGroupCounts(groups []auditstore.OverviewRiskGroup) map[auditstore.AuditBusinessCategory]int {
	counts := make(map[auditstore.AuditBusinessCategory]int, len(groups))
	for _, group := range groups {
		counts[auditstore.AuditBusinessCategory(group.Key)] = group.Count
	}
	return counts
}
