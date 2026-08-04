package contract

const (
	// NetworkGroup 是平台网络 HTTP API 的路由组。
	NetworkGroup = "/platform/network"
	// OutboundNetworkRoute 是出站网络策略的路径片段。
	OutboundNetworkRoute = "/outbound"
	// OutboundNetworkResetRoute 恢复出站网络策略的模块默认值。
	OutboundNetworkResetRoute = "/outbound/reset"
	// LegacyDiagnosticRoute 是 Phase 3 完成前旧 Network 页面使用的临时固定诊断适配路径。
	// COMPAT(owner=platform-network connectivity authority, cleanup=Phase 3 web consumer migrates to target-addressed connectivity APIs).
	LegacyDiagnosticRoute = "/diagnostics/:targetId"
	// LegacyDiagnosticHistoryRoute 是临时固定诊断历史适配路径。
	LegacyDiagnosticHistoryRoute = "/diagnostics/:targetId/history"
	// ConnectivityTargetsRoute 返回已注册连通性目标。
	ConnectivityTargetsRoute = "/connectivity/targets"
	// ConnectivityCustomTargetsRoute 管理经过 SSRF 校验的自定义 HTTP(S) 目标。
	ConnectivityCustomTargetsRoute = "/connectivity/custom-targets"
	// ConnectivityCustomTargetRoute 删除一个自定义目标。
	ConnectivityCustomTargetRoute = "/connectivity/custom-targets/:targetId"
	// ConnectivityLatestRoute 返回每个目标的最新健康检查。
	ConnectivityLatestRoute = "/connectivity/latest"
	// ConnectivityAggregateRoute 返回批量连通性聚合摘要。
	ConnectivityAggregateRoute = "/connectivity/aggregate"
	// ConnectivityRunRoute 为一个 target 执行诊断；页面身份始终是 target，而不是一次执行 ID。
	ConnectivityRunRoute = "/connectivity/:targetId/run"
	// ConnectivityBatchRunRoute 为当前所有已启用目标执行一次有界健康检查。
	ConnectivityBatchRunRoute = "/connectivity/run"
	// ConnectivityHistoryRoute 返回 target 的最近检查摘要。
	ConnectivityHistoryRoute = "/connectivity/:targetId/history"
	// ConnectivityReportRoute 返回 target 内某次检查的报告。
	ConnectivityReportRoute = "/connectivity/:targetId/reports/:checkId"
	// ConnectivityTraceRoute 返回与报告相同的已净化 Probe Trace 投影。
	ConnectivityTraceRoute = "/connectivity/:targetId/reports/:checkId/trace"
	// ConnectivityExportRoute 导出不含完整出口 IP 的报告 JSON。
	ConnectivityExportRoute = "/connectivity/:targetId/reports/:checkId/export"
	// NetworkMenuPath 是平台网络页面的稳定前端路由。
	NetworkMenuPath = "/platform/network"
)
