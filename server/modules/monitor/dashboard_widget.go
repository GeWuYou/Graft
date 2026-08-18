package monitor

import (
	"context"
	"fmt"

	generated "graft/server/internal/contract/openapi/generated"
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

// loadMonitorSystemHealthWidget 复用服务器状态聚合结果构造仪表盘健康投影，避免为首页重复采集依赖或主机数据。
func loadMonitorSystemHealthWidget(ctx context.Context, moduleCtx *module.Context, instance *Module) (dashboard.WidgetPayload, error) {
	response, err := buildServerStatusResponse(ctx, moduleCtx, instance, monitorcontract.TrendRange10Minutes)
	if err != nil {
		return nil, err
	}
	return monitorSystemHealthPayload(response), nil
}

func monitorSystemHealthPayload(response generated.ServerStatusResponse) dashboard.WidgetPayload {
	anomalyCount := len(response.Anomalies)
	items := []dashboard.HealthItem{
		{
			Key:            "host-resources",
			LabelKey:       "dashboard.widget.monitorSystemHealth.hostResources",
			Status:         monitorHostResourceStatus(response),
			DescriptionKey: "dashboard.widget.monitorSystemHealth.hostResourcesDescription",
			RouteLocation:  monitorcontract.ServerStatusOverviewMenuPath,
		},
		{
			Key:            "module-health",
			LabelKey:       "dashboard.widget.monitorSystemHealth.moduleHealth",
			Status:         monitorModuleHealthStatus(response.Summary),
			DescriptionKey: "dashboard.widget.monitorSystemHealth.moduleHealthDescription",
			RouteLocation:  monitorcontract.ServerStatusOverviewMenuPath,
		},
		{
			Key:            "anomalies",
			LabelKey:       "dashboard.widget.monitorSystemHealth.anomalies",
			Status:         monitorHealthStatusForAnomalies(anomalyCount),
			DescriptionKey: "dashboard.widget.monitorSystemHealth.anomaliesDescription",
			RouteLocation:  monitorcontract.ServerStatusOverviewMenuPath,
		},
		{
			Key:            "observability",
			LabelKey:       "dashboard.widget.monitorSystemHealth.observability",
			Status:         monitorObservabilityStatus(response),
			DescriptionKey: "dashboard.widget.monitorSystemHealth.observabilityDescription",
			RouteLocation:  monitorcontract.ServerStatusOverviewMenuPath,
		},
	}

	summaryStatus := monitorHealthSummaryStatus(items)
	state := dashboard.WidgetStateNormal
	priority := dashboard.WidgetPriorityNormal
	if summaryStatus == dashboard.HealthStatusDegraded {
		state = dashboard.WidgetStateWarning
		priority = dashboard.WidgetPriorityWarning
	}

	return dashboard.WidgetPayload{
		"summary": dashboard.HealthSummaryItem{
			Status:   summaryStatus,
			LabelKey: "dashboard.widget.monitorSystemHealth.summary",
		},
		"items":             items,
		"abnormal_services": anomalyCount,
		"state":             string(state),
		"priority":          string(priority),
	}
}

func monitorHostResourceStatus(response generated.ServerStatusResponse) dashboard.HealthStatus {
	if monitorHasResourceAnomaly(response.Anomalies) {
		return dashboard.HealthStatusDegraded
	}
	if response.Runtime.HostMemoryTotalBytes <= 0 && response.Runtime.DiskUsage.TotalBytes <= 0 && len(response.Trend.Points) == 0 {
		return dashboard.HealthStatusUnknown
	}
	return dashboard.HealthStatusHealthy
}

func monitorHasResourceAnomaly(anomalies []generated.ServerStatusAnomaly) bool {
	for _, anomaly := range anomalies {
		switch monitorcontract.AnomalyKey(anomaly.AnomalyKey) {
		case monitorcontract.ResourceCPUPressure,
			monitorcontract.ResourceMemoryPressure,
			monitorcontract.ResourceDiskPressure,
			monitorcontract.SystemLoadPressure:
			return true
		}
	}
	return false
}

func monitorModuleHealthStatus(summary generated.ServerStatusSummary) dashboard.HealthStatus {
	if summary.TotalModules == 0 {
		return dashboard.HealthStatusUnknown
	}
	if summary.HealthyModules < summary.TotalModules {
		return dashboard.HealthStatusDegraded
	}
	return dashboard.HealthStatusHealthy
}

func monitorObservabilityStatus(response generated.ServerStatusResponse) dashboard.HealthStatus {
	if len(response.Trend.Points) > 0 || monitorHasHostObservabilitySample(response.HostObservability) {
		return dashboard.HealthStatusHealthy
	}
	return dashboard.HealthStatusUnknown
}

func monitorHasHostObservabilitySample(observation generated.ServerStatusHostObservability) bool {
	return monitorHasProcessSample(observation.Process) ||
		monitorHasTCPSample(observation.Tcp) ||
		monitorHasNetworkSample(observation.Network) ||
		monitorHasDiskIOSample(observation.DiskIo)
}

func monitorHasProcessSample(process generated.ServerStatusHostProcess) bool {
	return process.OpenFileDescriptors != nil ||
		process.OsThreads != nil ||
		process.RssBytes != nil
}

func monitorHasTCPSample(tcp generated.ServerStatusHostTcp) bool {
	return tcp.Total != nil ||
		tcp.Established != nil ||
		tcp.CloseWait != nil ||
		tcp.TimeWait != nil
}

func monitorHasNetworkSample(network generated.ServerStatusHostNetwork) bool {
	return network.ReceivedBytesPerSecond != nil ||
		network.ReceivedPacketsPerSecond != nil ||
		network.SentBytesPerSecond != nil ||
		network.SentPacketsPerSecond != nil
}

func monitorHasDiskIOSample(diskIO generated.ServerStatusHostDiskIo) bool {
	return diskIO.ReadBytesPerSecond != nil ||
		diskIO.ReadIops != nil ||
		diskIO.ReadAverageLatencyMs != nil ||
		diskIO.WriteBytesPerSecond != nil ||
		diskIO.WriteIops != nil ||
		diskIO.WriteAverageLatencyMs != nil
}

func monitorHealthSummaryStatus(items []dashboard.HealthItem) dashboard.HealthStatus {
	status := dashboard.HealthStatusHealthy
	for _, item := range items {
		if item.Status == dashboard.HealthStatusDegraded {
			return dashboard.HealthStatusDegraded
		}
		if item.Status == dashboard.HealthStatusUnknown {
			status = dashboard.HealthStatusUnknown
		}
	}
	return status
}

func monitorHealthStatusForAnomalies(count int) dashboard.HealthStatus {
	if count > 0 {
		return dashboard.HealthStatusDegraded
	}
	return dashboard.HealthStatusHealthy
}
