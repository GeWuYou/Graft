package contract

// JoinRoute combines a route group path with a route fragment.
func JoinRoute(group, fragment string) string {
	return group + fragment
}

const (
	// MonitorGroup identifies the monitor route group.
	MonitorGroup = "/monitor"

	// ServerStatusRoute identifies the server-status route fragment.
	ServerStatusRoute = "/server-status"

	// OverviewRoute identifies the overview route fragment under server-status.
	OverviewRoute = "/overview"

	// RuntimeRoute identifies the runtime route fragment under server-status.
	RuntimeRoute = "/runtime"

	// DependenciesRoute identifies the dependencies route fragment under server-status.
	DependenciesRoute = "/dependencies"

	// ServerStatusMenuPath identifies the second-level server-status menu path.
	ServerStatusMenuPath = MonitorGroup + ServerStatusRoute

	// ServerStatusOverviewMenuPath identifies the third-level overview menu path.
	ServerStatusOverviewMenuPath = ServerStatusMenuPath + OverviewRoute

	// ServerStatusRuntimeMenuPath identifies the third-level runtime menu path.
	ServerStatusRuntimeMenuPath = ServerStatusMenuPath + RuntimeRoute

	// ServerStatusDependenciesMenuPath identifies the third-level dependencies menu path.
	ServerStatusDependenciesMenuPath = ServerStatusMenuPath + DependenciesRoute
)
