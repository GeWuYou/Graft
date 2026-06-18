// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

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

// registerMonitorDashboardWidget registers a system health monitoring widget with the dashboard registry. It returns an error if registration fails, or nil if the module context or dashboard registry is unavailable.
func registerMonitorDashboardWidget(moduleCtx *module.Context, instance *Module) error {
	if moduleCtx == nil || moduleCtx.DashboardRegistry == nil {
		return nil
	}

	if err := moduleCtx.DashboardRegistry.Register(dashboard.WidgetDefinition{
		ID:             monitorSystemHealthWidgetID,
		ModuleKey:      moduleID,
		TitleKey:       "dashboard.widget.monitorSystemHealth.title",
		Title:          "",
		DescriptionKey: "dashboard.widget.monitorSystemHealth.description",
		Description:    "",
		Type:           dashboard.WidgetTypeHealth,
		Size:           dashboard.WidgetSizeMedium,
		Category:       dashboard.WidgetCategorySystem,
		Priority:       dashboard.WidgetPriorityNormal,
		Order:          monitorSystemHealthWidgetOrder,
		RouteLocation:  monitorcontract.ServerStatusOverviewMenuPath,
		Action: dashboard.WidgetAction{
			LabelKey: "dashboard.actions.details",
			Label:    "",
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
// LoadMonitorSystemHealthWidget builds a dashboard widget payload containing system health status and anomaly information.
func loadMonitorSystemHealthWidget(ctx context.Context, moduleCtx *module.Context, instance *Module) (dashboard.WidgetPayload, error) {
	response, err := buildServerStatusResponse(ctx, moduleCtx, instance, monitorcontract.TrendRange10Minutes)
	if err != nil {
		return nil, err
	}

	items := []dashboard.HealthItem{
		{
			Key:            "database",
			LabelKey:       "dashboard.widget.monitorSystemHealth.database",
			Label:          "",
			Status:         dashboard.HealthStatus(response.Dependencies.Database.Status),
			DescriptionKey: "dashboard.widget.monitorSystemHealth.database" + dashboardStatusDescriptionSuffix(response.Dependencies.Database.Status),
			Description:    response.Dependencies.Database.Detail,
			RouteLocation:  monitorcontract.ServerStatusDependenciesMenuPath,
		},
		{
			Key:            "redis",
			LabelKey:       "dashboard.widget.monitorSystemHealth.redis",
			Label:          "",
			Status:         dashboard.HealthStatus(response.Dependencies.Redis.Status),
			DescriptionKey: "dashboard.widget.monitorSystemHealth.redis" + dashboardStatusDescriptionSuffix(response.Dependencies.Redis.Status),
			Description:    response.Dependencies.Redis.Detail,
			RouteLocation:  monitorcontract.ServerStatusDependenciesMenuPath,
		},
		{
			Key:            "anomalies",
			LabelKey:       "dashboard.widget.monitorSystemHealth.anomalies",
			Label:          "",
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
			Label:    "",
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
