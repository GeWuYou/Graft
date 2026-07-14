package contract

// JoinRoute combines a route group path with a route fragment.
func JoinRoute(group, fragment string) string {
	return group + fragment
}

const (
	// MonitorGroup identifies the monitor route group.
	MonitorGroup = "/monitor"

	// ServerStatusRoute identifies the server-status API route fragment.
	ServerStatusRoute = "/server-status"

	// MonitorMenuRoot identifies the canonical monitor bootstrap root path.
	MonitorMenuRoot = "/server"

	// OverviewRoute identifies the overview route fragment under server-status.
	OverviewRoute = "/overview"

	// ServiceStatusRoute identifies the service-status route fragment under server-status.
	ServiceStatusRoute = "/service-status"

	// DependenciesRoute identifies the dependencies route fragment under server-status.
	DependenciesRoute = "/dependencies"

	// RequestPerformanceRoute identifies the request-performance API route fragment.
	RequestPerformanceRoute = "/request-performance"

	// RequestPerformanceRangeQueryKey identifies the request-performance range query parameter.
	RequestPerformanceRangeQueryKey = "range"

	// ServerStatusMenuPath identifies the Observability UI route prefix.
	ServerStatusMenuPath = "/observability"

	// ServerStatusOverviewMenuPath identifies the canonical overview menu path.
	ServerStatusOverviewMenuPath = ServerStatusMenuPath + OverviewRoute

	// ServerStatusServiceStatusMenuPath identifies the canonical service-status menu path.
	ServerStatusServiceStatusMenuPath = ServerStatusMenuPath + ServiceStatusRoute

	// ServerStatusDependenciesMenuPath identifies the canonical dependencies menu path.
	ServerStatusDependenciesMenuPath = ServerStatusMenuPath + DependenciesRoute

	// RequestPerformanceMenuPath identifies the canonical request-performance menu path.
	RequestPerformanceMenuPath = ServerStatusMenuPath + RequestPerformanceRoute
)
