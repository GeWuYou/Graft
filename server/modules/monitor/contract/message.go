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
	// AuditEvidenceUnavailableTitle 标识不可用 audit 证据链接的标题。
	AuditEvidenceUnavailableTitle MessageKey = "monitor.evidence.auditUnavailable.title"
	// AnomalyDependencyDegraded 标识依赖降级异常摘要。
	AnomalyDependencyDegraded MessageKey = "monitor.serverStatus.anomaly.dependencyDegraded"
	// AnomalyDependencyUnknown 标识依赖状态未知异常摘要。
	AnomalyDependencyUnknown MessageKey = "monitor.serverStatus.anomaly.dependencyUnknown"
	// AnomalyModuleDependencyMissing 标识模块依赖缺失异常摘要。
	AnomalyModuleDependencyMissing MessageKey = "monitor.serverStatus.anomaly.moduleDependencyMissing"
	// AnomalyResourceCPUPressure 标识 CPU 压力异常摘要。
	AnomalyResourceCPUPressure MessageKey = "monitor.serverStatus.anomaly.resourceCpuPressure"
	// AnomalyResourceMemoryPressure 标识内存压力异常摘要。
	AnomalyResourceMemoryPressure MessageKey = "monitor.serverStatus.anomaly.resourceMemoryPressure"
	// AnomalyResourceDiskPressure 标识磁盘压力异常摘要。
	AnomalyResourceDiskPressure MessageKey = "monitor.serverStatus.anomaly.resourceDiskPressure"
	// AnomalyRuntimeGoroutinePressure 标识 goroutine 压力异常摘要。
	AnomalyRuntimeGoroutinePressure MessageKey = "monitor.serverStatus.anomaly.runtimeGoroutinePressure"
	// AnomalyRuntimeHeapPressure 标识堆内存压力异常摘要。
	AnomalyRuntimeHeapPressure MessageKey = "monitor.serverStatus.anomaly.runtimeHeapPressure"
	// AnomalySystemLoadPressure 标识系统负载压力异常摘要。
	AnomalySystemLoadPressure MessageKey = "monitor.serverStatus.anomaly.systemLoadPressure"
)
