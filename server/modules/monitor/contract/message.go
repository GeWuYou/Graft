package contract

// MessageKey identifies a stable monitor module message key.
type MessageKey string

// String returns the canonical menu message key value.
func (k MessageKey) String() string {
	return string(k)
}

const (
	// MonitorSectionTitle identifies the localized title for the monitor navigation group.
	MonitorSectionTitle MessageKey = "monitor.sectionTitle"
	// ServerStatusMenuTitle identifies the localized title for the server management root menu.
	ServerStatusMenuTitle MessageKey = "menu.server.title"
	// ServerStatusOverviewMenuTitle identifies the localized title for the server overview menu.
	ServerStatusOverviewMenuTitle MessageKey = "menu.server.overview.title"
	// ServerStatusRuntimeMenuTitle identifies the localized title for the server runtime menu.
	ServerStatusRuntimeMenuTitle MessageKey = "menu.server.runtime.title"
	// ServerStatusDependenciesMenuTitle identifies the localized title for the server dependencies menu.
	ServerStatusDependenciesMenuTitle MessageKey = "menu.server.dependencies.title"
	// AuditEvidenceUnavailableTitle identifies unavailable audit evidence link titles.
	AuditEvidenceUnavailableTitle MessageKey = "monitor.evidence.auditUnavailable.title"
)
