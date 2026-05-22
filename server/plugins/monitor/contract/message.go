package contract

// MenuMessageKey identifies a stable monitor-plugin menu title message key.
type MenuMessageKey string

// String returns the canonical menu message key value.
func (k MenuMessageKey) String() string {
	return string(k)
}

const (
	// MonitorSectionTitle identifies the localized title for the monitor navigation group.
	MonitorSectionTitle MenuMessageKey = "monitor.sectionTitle"
	// ServerStatusMenuTitle identifies the localized title for the server-status menu.
	ServerStatusMenuTitle MenuMessageKey = "menu.monitor.server_status.title"
	// ServerStatusOverviewMenuTitle identifies the localized title for the server-status overview menu.
	ServerStatusOverviewMenuTitle MenuMessageKey = "menu.monitor.server_status.overview.title"
	// ServerStatusRuntimeMenuTitle identifies the localized title for the server-status runtime menu.
	ServerStatusRuntimeMenuTitle MenuMessageKey = "menu.monitor.server_status.runtime.title"
	// ServerStatusDependenciesMenuTitle identifies the localized title for the server-status dependencies menu.
	ServerStatusDependenciesMenuTitle MenuMessageKey = "menu.monitor.server_status.dependencies.title"
)
