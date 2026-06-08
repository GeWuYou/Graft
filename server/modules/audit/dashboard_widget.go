package audit

import (
	"context"
	"fmt"
	"strconv"

	"graft/server/internal/dashboard"
	"graft/server/internal/module"
	auditcontract "graft/server/modules/audit/contract"
	auditstore "graft/server/modules/audit/store"
)

const (
	auditRiskEventsWidgetID    = "audit.risk-events"
	auditRiskEventsWidgetOrder = 100
	auditRiskEventsItemCap     = 5
)

func registerAuditDashboardWidget(ctx *module.Context, reader *Service) error {
	if ctx == nil || ctx.DashboardRegistry == nil {
		return nil
	}

	if err := ctx.DashboardRegistry.Register(dashboard.WidgetDefinition{
		ID:                  auditRiskEventsWidgetID,
		ModuleKey:           moduleID,
		TitleKey:            "dashboard.widget.auditRiskEvents.title",
		Title:               "Audit Risk Events",
		DescriptionKey:      "dashboard.widget.auditRiskEvents.description",
		Description:         "Recent high-risk audit and security events.",
		Type:                dashboard.WidgetTypeAlertList,
		Size:                dashboard.WidgetSizeMedium,
		Order:               auditRiskEventsWidgetOrder,
		RouteLocation:       auditcontract.AuditOverviewMenuPath,
		RequiredPermissions: []string{auditcontract.AuditReadPermission.String()},
		Loader: dashboard.WidgetLoaderFunc(func(ctx context.Context, _ dashboard.WidgetRequest) (dashboard.WidgetPayload, error) {
			return loadAuditRiskEventsWidget(ctx, reader)
		}),
	}); err != nil {
		return fmt.Errorf("register audit dashboard widget: %w", err)
	}

	return nil
}

func loadAuditRiskEventsWidget(ctx context.Context, reader *Service) (dashboard.WidgetPayload, error) {
	overview, err := reader.Overview(ctx, auditstore.AuditTimePresetLast24Hours)
	if err != nil {
		return nil, err
	}

	items := make([]map[string]any, 0, auditRiskEventsItemCap)
	if overview.Summary.HighRiskEvents > 0 {
		items = append(items, map[string]any{
			"id":             "audit.high-risk",
			"level":          "error",
			"title_key":      "dashboard.widget.auditRiskEvents.highRisk.title",
			"title":          "High-risk audit events",
			"description":    strconv.Itoa(overview.Summary.HighRiskEvents) + " high-risk events in the last 24 hours.",
			"route_location": auditcontract.AuditOverviewMenuPath,
		})
	}
	if overview.Summary.FailedOperations > 0 {
		items = append(items, map[string]any{
			"id":             "audit.failed-operations",
			"level":          "warning",
			"title_key":      "dashboard.widget.auditRiskEvents.failedOperations.title",
			"title":          "Failed operations",
			"description":    strconv.Itoa(overview.Summary.FailedOperations) + " failed operations need review.",
			"route_location": auditcontract.AuditOverviewMenuPath,
		})
	}
	items = appendAuditOverviewItems(items, "audit.failed-auth", "Failed authentication", overview.FailedAuth)
	items = appendAuditOverviewItems(items, "audit.permission-denied", "Permission denied", overview.PermissionDenied)

	return dashboard.WidgetPayload{
		"items":     items,
		"empty_key": "dashboard.widget.auditRiskEvents.empty",
		"empty":     "No audit risk events in the last 24 hours.",
	}, nil
}

func appendAuditOverviewItems(items []map[string]any, prefix string, title string, source []auditstore.OverviewItem) []map[string]any {
	const maxAuditPreviewItems = 2
	for index, item := range source {
		if index >= maxAuditPreviewItems {
			break
		}
		description := item.Message
		if description == "" {
			description = item.Action
		}
		items = append(items, map[string]any{
			"id":             prefix + "." + strconv.FormatUint(item.ID, 10),
			"level":          "warning",
			"title_key":      "dashboard.widget.auditRiskEvents.recent.title",
			"title":          title,
			"description":    description,
			"occurred_at":    item.CreatedAt,
			"route_location": auditcontract.AuditOverviewMenuPath,
		})
	}
	return items
}
