package contract

// MessageKey 标识 monitor 模块稳定的本地化消息键。
type MessageKey string

// String 返回可在线路契约中使用的规范消息键值。
func (k MessageKey) String() string {
	return string(k)
}

const (
	// MonitorSectionTitle 标识 monitor 导航分组的本地化标题。
	MonitorSectionTitle MessageKey = "monitor.sectionTitle"
	// ServerStatusOverviewMenuTitle 标识 monitor 概览菜单的本地化标题。
	ServerStatusOverviewMenuTitle MessageKey = "menu.monitor.overview.title"
	// ServerStatusServiceStatusMenuTitle 标识服务状态菜单的本地化标题。
	ServerStatusServiceStatusMenuTitle MessageKey = "menu.monitor.serviceStatus.title"
	// ServerStatusDependenciesMenuTitle 标识 monitor 依赖菜单的本地化标题。
	ServerStatusDependenciesMenuTitle MessageKey = "menu.monitor.dependencies.title"
	// RequestPerformanceMenuTitle 标识请求性能菜单的本地化标题。
	RequestPerformanceMenuTitle MessageKey = "menu.monitor.requestPerformance.title"
	// AuditEvidenceUnavailableTitle 标识不可用 audit 证据链接的标题。
	AuditEvidenceUnavailableTitle MessageKey = "monitor.evidence.auditUnavailable.title"
)
