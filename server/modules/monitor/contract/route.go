package contract

// JoinRoute 将路由分组路径与片段拼接为规范 API 路径。
func JoinRoute(group, fragment string) string {
	return group + fragment
}

const (
	// MonitorGroup 标识 monitor API 的路由分组。
	MonitorGroup = "/monitor"

	// ServerStatusRoute 标识服务器状态 API 路由片段。
	ServerStatusRoute = "/server-status"

	// MonitorMenuRoot 标识 monitor bootstrap 使用的规范根路径。
	MonitorMenuRoot = "/server"

	// OverviewRoute 标识服务器状态下的概览路由片段。
	OverviewRoute = "/overview"

	// ServiceStatusRoute 标识服务器状态下的服务状态路由片段。
	ServiceStatusRoute = "/service-status"

	// DependenciesRoute 标识服务器状态下的依赖路由片段。
	DependenciesRoute = "/dependencies"

	// RequestPerformanceRoute 标识请求性能 API 路由片段。
	RequestPerformanceRoute = "/request-performance"

	// RequestPerformanceRangeQueryKey 标识请求性能范围查询参数。
	RequestPerformanceRangeQueryKey = "range"

	// ServerStatusMenuPath 标识 Observability UI 的服务器状态路径前缀。
	ServerStatusMenuPath = "/observability"

	// ServerStatusOverviewMenuPath 标识规范概览菜单路径。
	ServerStatusOverviewMenuPath = ServerStatusMenuPath + OverviewRoute

	// ServerStatusServiceStatusMenuPath 标识规范服务状态菜单路径。
	ServerStatusServiceStatusMenuPath = ServerStatusMenuPath + ServiceStatusRoute

	// ServerStatusDependenciesMenuPath 标识规范依赖菜单路径。
	ServerStatusDependenciesMenuPath = ServerStatusMenuPath + DependenciesRoute

	// RequestPerformanceMenuPath 标识规范请求性能菜单路径。
	RequestPerformanceMenuPath = ServerStatusMenuPath + RequestPerformanceRoute
)
