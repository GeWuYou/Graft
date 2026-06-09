package monitor

import (
	"context"
	"fmt"
	"strconv"

	"graft/server/internal/dashboard"
	"graft/server/internal/module"
	monitorcontract "graft/server/modules/monitor/contract"
)

const (
	monitorSystemHealthWidgetID       = "monitor.system-health"
	monitorSystemHealthWidgetOrder    = 90
	monitorOverviewQuickLinkID        = "monitor.server.overview"
	monitorRuntimeQuickLinkID         = "monitor.server.runtime"
	monitorDependenciesQuickLinkID    = "monitor.server.dependencies"
	monitorOverviewQuickLinkOrder     = 60
	monitorRuntimeQuickLinkOrder      = 70
	monitorDependenciesQuickLinkOrder = 80
)

func registerMonitorDashboardWidget(moduleCtx *module.Context, instance *Module) error {
	if moduleCtx == nil || moduleCtx.DashboardRegistry == nil {
		return nil
	}

	for _, link := range monitorQuickLinks() {
		if err := moduleCtx.DashboardRegistry.RegisterQuickLink(link); err != nil {
			return fmt.Errorf("register monitor dashboard quick link: %w", err)
		}
	}

	if err := moduleCtx.DashboardRegistry.Register(dashboard.WidgetDefinition{
		ID:             monitorSystemHealthWidgetID,
		ModuleKey:      moduleID,
		TitleKey:       "dashboard.widget.monitorSystemHealth.title",
		Title:          "System Health",
		DescriptionKey: "dashboard.widget.monitorSystemHealth.description",
		Description:    "Current service health, dependency state, and active anomalies.",
		Type:           dashboard.WidgetTypeHealth,
		Size:           dashboard.WidgetSizeMedium,
		Category:       dashboard.WidgetCategorySystem,
		Priority:       dashboard.WidgetPriorityNormal,
		Order:          monitorSystemHealthWidgetOrder,
		RouteLocation:  monitorcontract.ServerStatusOverviewMenuPath,
		Action: dashboard.WidgetAction{
			LabelKey: "dashboard.actions.details",
			Label:    "View details",
			Route:    monitorcontract.ServerStatusOverviewMenuPath,
		},
		RequiredPermissions: []string{monitorcontract.ServerStatusReadPermission.String()},
		Loader: dashboard.WidgetLoaderFunc(func(loadCtx context.Context, _ dashboard.WidgetRequest) (dashboard.WidgetPayload, error) {
			return loadMonitorSystemHealthWidget(loadCtx, moduleCtx, instance)
		}),
	}); err != nil {
		return fmt.Errorf("register monitor dashboard widget: %w", err)
	}

	return nil
}

func monitorQuickLinks() []dashboard.QuickLinkDefinition {
	requiredPermissions := []string{monitorcontract.ServerStatusReadPermission.String()}
	return []dashboard.QuickLinkDefinition{
		{
			ID:                  monitorOverviewQuickLinkID,
			ModuleKey:           moduleID,
			TitleKey:            monitorcontract.ServerStatusOverviewMenuTitle.String(),
			Title:               "Server Overview",
			Icon:                "dashboard",
			RouteLocation:       monitorcontract.ServerStatusOverviewMenuPath,
			RequiredPermissions: append([]string(nil), requiredPermissions...),
			Order:               monitorOverviewQuickLinkOrder,
		},
		{
			ID:                  monitorRuntimeQuickLinkID,
			ModuleKey:           moduleID,
			TitleKey:            monitorcontract.ServerStatusRuntimeMenuTitle.String(),
			Title:               "Runtime",
			Icon:                "server",
			RouteLocation:       monitorcontract.ServerStatusRuntimeMenuPath,
			RequiredPermissions: append([]string(nil), requiredPermissions...),
			Order:               monitorRuntimeQuickLinkOrder,
		},
		{
			ID:                  monitorDependenciesQuickLinkID,
			ModuleKey:           moduleID,
			TitleKey:            monitorcontract.ServerStatusDependenciesMenuTitle.String(),
			Title:               "Dependencies",
			Icon:                "link",
			RouteLocation:       monitorcontract.ServerStatusDependenciesMenuPath,
			RequiredPermissions: append([]string(nil), requiredPermissions...),
			Order:               monitorDependenciesQuickLinkOrder,
		},
	}
}

func loadMonitorSystemHealthWidget(ctx context.Context, moduleCtx *module.Context, instance *Module) (dashboard.WidgetPayload, error) {
	response, err := buildServerStatusResponse(ctx, moduleCtx, instance, monitorcontract.TrendRange10Minutes)
	if err != nil {
		return nil, err
	}

	items := []dashboard.HealthItem{
		{
			Key:            "database",
			LabelKey:       "dashboard.widget.monitorSystemHealth.database",
			Label:          "Database",
			Status:         dashboard.HealthStatus(response.Dependencies.Database.Status),
			DescriptionKey: "dashboard.widget.monitorSystemHealth.database" + dashboardStatusDescriptionSuffix(response.Dependencies.Database.Status),
			Description:    response.Dependencies.Database.Detail,
			RouteLocation:  monitorcontract.ServerStatusDependenciesMenuPath,
		},
		{
			Key:            "redis",
			LabelKey:       "dashboard.widget.monitorSystemHealth.redis",
			Label:          "Redis",
			Status:         dashboard.HealthStatus(response.Dependencies.Redis.Status),
			DescriptionKey: "dashboard.widget.monitorSystemHealth.redis" + dashboardStatusDescriptionSuffix(response.Dependencies.Redis.Status),
			Description:    response.Dependencies.Redis.Detail,
			RouteLocation:  monitorcontract.ServerStatusDependenciesMenuPath,
		},
		{
			Key:            "anomalies",
			LabelKey:       "dashboard.widget.monitorSystemHealth.anomalies",
			Label:          "Active anomalies",
			Status:         monitorHealthStatusForAnomalies(len(response.Anomalies)),
			DescriptionKey: "dashboard.widget.monitorSystemHealth.anomaliesDescription",
			Description:    strconv.Itoa(len(response.Anomalies)) + " active anomalies in the monitor window.",
			RouteLocation:  monitorcontract.ServerStatusOverviewMenuPath,
		},
	}

	state := dashboard.WidgetStateNormal
	priority := dashboard.WidgetPriorityNormal
	if response.Status != "healthy" || len(response.Anomalies) > 0 {
		state = dashboard.WidgetStateWarning
		priority = dashboard.WidgetPriorityWarning
	}

	return dashboard.WidgetPayload{
		"summary": dashboard.HealthSummaryItem{
			Status:   dashboard.HealthStatus(response.Status),
			LabelKey: "dashboard.widget.monitorSystemHealth.summary",
			Label:    "System health",
		},
		"items":             items,
		"abnormal_services": len(response.Anomalies),
		"state":             string(state),
		"priority":          string(priority),
	}, nil
}

func dashboardStatusDescriptionSuffix(status string) string {
	switch status {
	case "healthy":
		return "HealthyDescription"
	case "degraded":
		return "DegradedDescription"
	case "disabled":
		return "DisabledDescription"
	default:
		return "UnknownDescription"
	}
}

func monitorHealthStatusForAnomalies(count int) dashboard.HealthStatus {
	if count > 0 {
		return dashboard.HealthStatusDegraded
	}
	return dashboard.HealthStatusHealthy
}
