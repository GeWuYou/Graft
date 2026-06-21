package contract

// NavigationKind identifies a stable notification navigation target contract.
//
// Canonical owner: server/modules/notification/contract.
// Lifecycle: stable values remain authoritative until this package marks a replacement or removal.
type NavigationKind string

// String returns the canonical navigation kind value.
func (k NavigationKind) String() string {
	return string(k)
}

const (
	// NavigationAuditIncident targets an audit incident detail.
	// Lifecycle: stable.
	NavigationAuditIncident NavigationKind = "AUDIT_INCIDENT"
	// NavigationAuditLog targets an audit log detail.
	// Lifecycle: stable.
	NavigationAuditLog NavigationKind = "AUDIT_LOG"
	// NavigationSchedulerRun targets a scheduled task run detail.
	// Lifecycle: stable.
	NavigationSchedulerRun NavigationKind = "SCHEDULER_RUN"
	// NavigationSystemConfigItem is reserved for a system config item.
	// Lifecycle: experimental.
	NavigationSystemConfigItem NavigationKind = "SYSTEM_CONFIG_ITEM"
	// NavigationModuleRuntimeItem is reserved for a module runtime detail.
	// Lifecycle: experimental.
	NavigationModuleRuntimeItem NavigationKind = "MODULE_RUNTIME_ITEM"
)

// ValidNavigationKind reports whether value is a known navigation contract.
func ValidNavigationKind(value NavigationKind) bool {
	switch value {
	case NavigationAuditIncident, NavigationAuditLog, NavigationSchedulerRun, NavigationSystemConfigItem, NavigationModuleRuntimeItem:
		return true
	default:
		return false
	}
}
