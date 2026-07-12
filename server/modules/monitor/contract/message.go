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
	// ServerStatusOverviewMenuTitle identifies the localized title for the monitor overview menu.
	ServerStatusOverviewMenuTitle MessageKey = "menu.monitor.overview.title"
	// ServerStatusRuntimeMenuTitle identifies the localized title for the monitor runtime menu.
	ServerStatusRuntimeMenuTitle MessageKey = "menu.monitor.runtime.title"
	// ServerStatusDependenciesMenuTitle identifies the localized title for the monitor dependencies menu.
	ServerStatusDependenciesMenuTitle MessageKey = "menu.monitor.dependencies.title"
	// AuditEvidenceUnavailableTitle identifies unavailable audit evidence link titles.
	AuditEvidenceUnavailableTitle MessageKey = "monitor.evidence.auditUnavailable.title"
)
