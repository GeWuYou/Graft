package contract

const (
	// NetworkGroup 是平台网络 HTTP API 的路由组。
	NetworkGroup = "/platform/network"
	// OutboundNetworkRoute 是出站网络策略的路径片段。
	OutboundNetworkRoute = "/outbound"
	// OutboundNetworkResetRoute 恢复出站网络策略的模块默认值。
	OutboundNetworkResetRoute = "/outbound/reset"
	// OutboundNetworkDiagnosticRoute 执行固定注册的出站网络诊断。
	OutboundNetworkDiagnosticRoute = "/outbound/diagnostics/:targetId"
	// OutboundNetworkDiagnosticHistoryRoute 返回固定注册目标的有限诊断历史。
	OutboundNetworkDiagnosticHistoryRoute = "/outbound/diagnostics/:targetId/history"
	// NetworkMenuPath 是平台网络页面的稳定前端路由。
	NetworkMenuPath = "/platform/network"
)
