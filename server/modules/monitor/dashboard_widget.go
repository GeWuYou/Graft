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

// registerMonitorDashboardWidget 向仪表盘注册系统健康 widget。
// 模块上下文或注册表不可用时跳过注册；注册失败则返回错误。
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

// loadMonitorSystemHealthWidget 构造由活跃异常聚合的仪表盘 widget 数据。
func loadMonitorSystemHealthWidget(ctx context.Context, moduleCtx *module.Context, instance *Module) (dashboard.WidgetPayload, error) {
	response, err := buildServerStatusResponse(ctx, moduleCtx, instance, monitorcontract.TrendRange10Minutes)
	if err != nil {
		return nil, err
	}

	anomalyCount := len(response.Anomalies)
	items := []dashboard.HealthItem{
		{
			Key:            "anomalies",
			LabelKey:       "dashboard.widget.monitorSystemHealth.anomalies",
			Label:          "",
			Status:         monitorHealthStatusForAnomalies(anomalyCount),
			DescriptionKey: "dashboard.widget.monitorSystemHealth.anomaliesDescription",
			Description:    strconv.Itoa(anomalyCount) + " active anomalies in the monitor window.",
			RouteLocation:  monitorcontract.ServerStatusOverviewMenuPath,
		},
	}

	state := dashboard.WidgetStateNormal
	priority := dashboard.WidgetPriorityNormal
	if anomalyCount > 0 {
		state = dashboard.WidgetStateWarning
		priority = dashboard.WidgetPriorityWarning
	}

	return dashboard.WidgetPayload{
		"summary": dashboard.HealthSummaryItem{
			Status:   monitorHealthStatusForAnomalies(anomalyCount),
			LabelKey: "dashboard.widget.monitorSystemHealth.summary",
			Label:    "",
		},
		"items":             items,
		"abnormal_services": anomalyCount,
		"state":             string(state),
		"priority":          string(priority),
	}, nil
}

func monitorHealthStatusForAnomalies(count int) dashboard.HealthStatus {
	if count > 0 {
		return dashboard.HealthStatusDegraded
	}
	return dashboard.HealthStatusHealthy
}
