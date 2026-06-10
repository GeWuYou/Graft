// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package audit

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"graft/server/internal/dashboard"
	"graft/server/internal/module"
	auditcontract "graft/server/modules/audit/contract"
	auditstore "graft/server/modules/audit/store"
)

const (
	auditRiskEventsWidgetID     = "audit.risk-events"
	auditRiskEventsWidgetOrder  = 100
	auditRiskEventsItemCap      = 5
	auditLogsQueryPreset        = "preset"
	auditLogsQueryScope         = "scope"
	auditOverviewQuickLinkID    = "audit.overview"
	auditLogsQuickLinkID        = "audit.logs"
	auditOverviewQuickLinkOrder = 140
	auditLogsQuickLinkOrder     = 150
)

func registerAuditDashboardWidget(ctx *module.Context, reader *Service) error {
	if ctx == nil || ctx.DashboardRegistry == nil {
		return nil
	}

	for _, link := range auditQuickLinks() {
		if err := ctx.DashboardRegistry.RegisterQuickLink(link); err != nil {
			return fmt.Errorf("register audit dashboard quick link: %w", err)
		}
	}

	if err := ctx.DashboardRegistry.Register(dashboard.WidgetDefinition{
		ID:             auditRiskEventsWidgetID,
		ModuleKey:      moduleID,
		TitleKey:       "dashboard.widget.auditRiskEvents.title",
		Title:          "Audit Risk Events",
		DescriptionKey: "dashboard.widget.auditRiskEvents.description",
		Description:    "Recent high-risk audit and security events.",
		Type:           dashboard.WidgetTypeAlertList,
		Size:           dashboard.WidgetSizeMedium,
		Category:       dashboard.WidgetCategorySecurity,
		Priority:       dashboard.WidgetPriorityWarning,
		Order:          auditRiskEventsWidgetOrder,
		RouteLocation:  auditcontract.AuditOverviewMenuPath,
		Action: dashboard.WidgetAction{
			LabelKey: "dashboard.actions.details",
			Label:    "View details",
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

func auditQuickLinks() []dashboard.QuickLinkDefinition {
	requiredPermissions := []string{auditcontract.AuditReadPermission.String()}
	return []dashboard.QuickLinkDefinition{
		{
			ID:                  auditOverviewQuickLinkID,
			ModuleKey:           moduleID,
			TitleKey:            auditcontract.AuditOverviewMenuTitle.String(),
			Title:               "Security Audit Overview",
			Icon:                "dashboard",
			RouteLocation:       auditcontract.AuditOverviewMenuPath,
			RequiredPermissions: append([]string(nil), requiredPermissions...),
			Order:               auditOverviewQuickLinkOrder,
		},
		{
			ID:                  auditLogsQuickLinkID,
			ModuleKey:           moduleID,
			TitleKey:            auditcontract.AuditLogMenuTitle.String(),
			Title:               "Audit Logs",
			Icon:                "history",
			RouteLocation:       auditcontract.AuditLogsMenuPath,
			RequiredPermissions: append([]string(nil), requiredPermissions...),
			Order:               auditLogsQuickLinkOrder,
		},
	}
}

func loadAuditRiskEventsWidget(ctx context.Context, reader *Service) (dashboard.WidgetPayload, error) {
	overview, err := reader.Overview(ctx, auditstore.AuditTimePresetLast24Hours)
	if err != nil {
		return nil, err
	}

	items := make([]map[string]any, 0, auditRiskEventsItemCap)
	if overview.Summary.HighRiskEvents > 0 {
		items = append(items, map[string]any{
			"id":              "audit.high-risk",
			"level":           "error",
			"title_key":       "dashboard.widget.auditRiskEvents.highRisk.title",
			"title":           "High-risk audit events",
			"description_key": "dashboard.widget.auditRiskEvents.highRisk.description",
			"description":     strconv.Itoa(overview.Summary.HighRiskEvents) + " high-risk events in the last 24 hours.",
			"count":           overview.Summary.HighRiskEvents,
			"route_location":  auditRiskDashboardLocation(auditstore.AuditBusinessCategoryHighRiskOperations),
		})
	}
	if overview.Summary.FailedOperations > 0 {
		items = append(items, map[string]any{
			"id":              "audit.failed-operations",
			"level":           "warning",
			"title_key":       "dashboard.widget.auditRiskEvents.failedOperations.title",
			"title":           "Failed operations",
			"description_key": "dashboard.widget.auditRiskEvents.failedOperations.description",
			"description":     strconv.Itoa(overview.Summary.FailedOperations) + " failed operations need review.",
			"count":           overview.Summary.FailedOperations,
			"route_location":  auditRiskDashboardLocation(auditstore.AuditBusinessCategoryFailedOperations),
		})
	}
	riskGroupCounts := auditRiskGroupCounts(overview.RiskGroups)
	items = appendAuditOverviewGroupItem(items, auditOverviewGroupItemDefinition{
		count:          riskGroupCounts[auditstore.AuditBusinessCategoryAuthFailures],
		id:             "audit.failed-auth",
		items:          overview.FailedAuth,
		scope:          auditstore.AuditBusinessCategoryAuthFailures,
		title:          "Authentication failures",
		titleKey:       "audit.overview.riskGroups.authFailures",
		descriptionKey: "dashboard.widget.auditRiskEvents.authFailures.description",
	})
	items = appendAuditOverviewGroupItem(items, auditOverviewGroupItemDefinition{
		count:          riskGroupCounts[auditstore.AuditBusinessCategoryPermissionDenials],
		id:             "audit.permission-denied",
		items:          overview.PermissionDenied,
		scope:          auditstore.AuditBusinessCategoryPermissionDenials,
		title:          "Permission denials",
		titleKey:       "audit.overview.riskGroups.permissionDenials",
		descriptionKey: "dashboard.widget.auditRiskEvents.permissionDenials.description",
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
		"empty":            "No audit risk events in the last 24 hours.",
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
	title          string
	titleKey       string
	descriptionKey string
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
		"id":              definition.id,
		"level":           "warning",
		"title_key":       definition.titleKey,
		"title":           definition.title,
		"description_key": definition.descriptionKey,
		"description":     description,
		"count":           definition.count,
		"occurred_at":     latest.CreatedAt,
		"route_location":  auditRiskDashboardLocation(definition.scope),
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

func auditRiskDashboardLocation(scope auditstore.AuditBusinessCategory) string {
	query := auditDashboardPresetQuery()
	query.Set(auditLogsQueryScope, string(scope))
	return auditLogsDashboardLocation(query)
}

func auditRiskGroupCounts(groups []auditstore.OverviewRiskGroup) map[auditstore.AuditBusinessCategory]int {
	counts := make(map[auditstore.AuditBusinessCategory]int, len(groups))
	for _, group := range groups {
		counts[auditstore.AuditBusinessCategory(group.Key)] = group.Count
	}
	return counts
}
