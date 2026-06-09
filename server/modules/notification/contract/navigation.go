// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package contract

// NavigationKind identifies a stable notification navigation target contract.
type NavigationKind string

// String returns the canonical navigation kind value.
func (k NavigationKind) String() string {
	return string(k)
}

const (
	// NavigationAuditIncident targets an audit incident detail.
	NavigationAuditIncident NavigationKind = "AUDIT_INCIDENT"
	// NavigationAuditLog targets an audit log detail.
	NavigationAuditLog NavigationKind = "AUDIT_LOG"
	// NavigationSchedulerRun targets a scheduled task run detail.
	NavigationSchedulerRun NavigationKind = "SCHEDULER_RUN"
	// NavigationSystemConfigItem is reserved for a system config item.
	NavigationSystemConfigItem NavigationKind = "SYSTEM_CONFIG_ITEM"
	// NavigationModuleRuntimeItem is reserved for a module runtime detail.
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
