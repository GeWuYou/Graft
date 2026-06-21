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

	// RuntimeRoute identifies the runtime route fragment under server-status.
	RuntimeRoute = "/runtime"

	// DependenciesRoute identifies the dependencies route fragment under server-status.
	DependenciesRoute = "/dependencies"

	// ServerStatusMenuPath identifies the second-level server management menu path.
	ServerStatusMenuPath = MonitorMenuRoot

	// ServerStatusOverviewMenuPath identifies the canonical overview menu path.
	ServerStatusOverviewMenuPath = ServerStatusMenuPath + OverviewRoute

	// ServerStatusRuntimeMenuPath identifies the canonical runtime menu path.
	ServerStatusRuntimeMenuPath = ServerStatusMenuPath + RuntimeRoute

	// ServerStatusDependenciesMenuPath identifies the canonical dependencies menu path.
	ServerStatusDependenciesMenuPath = ServerStatusMenuPath + DependenciesRoute
)
