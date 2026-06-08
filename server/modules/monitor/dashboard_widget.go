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
	monitorSystemHealthWidgetID    = "monitor.system-health"
	monitorSystemHealthWidgetOrder = 90
)

func registerMonitorDashboardWidget(moduleCtx *module.Context, instance *Module) error {
	if moduleCtx == nil || moduleCtx.DashboardRegistry == nil {
		return nil
	}

	if err := moduleCtx.DashboardRegistry.Register(dashboard.WidgetDefinition{
		ID:                  monitorSystemHealthWidgetID,
		ModuleKey:           moduleID,
		TitleKey:            "dashboard.widget.monitorSystemHealth.title",
		Title:               "System Health",
		DescriptionKey:      "dashboard.widget.monitorSystemHealth.description",
		Description:         "Current service health, dependency state, and active anomalies.",
		Type:                dashboard.WidgetTypeHealth,
		Size:                dashboard.WidgetSizeMedium,
		Order:               monitorSystemHealthWidgetOrder,
		RouteLocation:       monitorcontract.ServerStatusOverviewMenuPath,
		RequiredPermissions: []string{monitorcontract.ServerStatusReadPermission.String()},
		Loader: dashboard.WidgetLoaderFunc(func(loadCtx context.Context, _ dashboard.WidgetRequest) (dashboard.WidgetPayload, error) {
			return loadMonitorSystemHealthWidget(loadCtx, moduleCtx, instance)
		}),
	}); err != nil {
		return fmt.Errorf("register monitor dashboard widget: %w", err)
	}

	return nil
}

func loadMonitorSystemHealthWidget(ctx context.Context, moduleCtx *module.Context, instance *Module) (dashboard.WidgetPayload, error) {
	response, err := buildServerStatusResponse(ctx, moduleCtx, instance, monitorcontract.TrendRange10Minutes)
	if err != nil {
		return nil, err
	}

	items := []dashboard.HealthItem{
		{
			Key:           "database",
			LabelKey:      "dashboard.widget.monitorSystemHealth.database",
			Label:         "Database",
			Status:        dashboard.HealthStatus(response.Dependencies.Database.Status),
			Description:   response.Dependencies.Database.Detail,
			RouteLocation: monitorcontract.ServerStatusDependenciesMenuPath,
		},
		{
			Key:           "redis",
			LabelKey:      "dashboard.widget.monitorSystemHealth.redis",
			Label:         "Redis",
			Status:        dashboard.HealthStatus(response.Dependencies.Redis.Status),
			Description:   response.Dependencies.Redis.Detail,
			RouteLocation: monitorcontract.ServerStatusDependenciesMenuPath,
		},
		{
			Key:           "anomalies",
			LabelKey:      "dashboard.widget.monitorSystemHealth.anomalies",
			Label:         "Active anomalies",
			Status:        monitorHealthStatusForAnomalies(len(response.Anomalies)),
			Description:   strconv.Itoa(len(response.Anomalies)) + " active anomalies in the monitor window.",
			RouteLocation: monitorcontract.ServerStatusOverviewMenuPath,
		},
	}

	return dashboard.WidgetPayload{
		"summary": dashboard.HealthSummaryItem{
			Status:   dashboard.HealthStatus(response.Status),
			LabelKey: "dashboard.widget.monitorSystemHealth.summary",
			Label:    "System health",
		},
		"items": items,
	}, nil
}

func monitorHealthStatusForAnomalies(count int) dashboard.HealthStatus {
	if count > 0 {
		return dashboard.HealthStatusDegraded
	}
	return dashboard.HealthStatusHealthy
}
