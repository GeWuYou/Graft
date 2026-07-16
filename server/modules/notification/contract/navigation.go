package contract

// NavigationKind 标识通知稳定的导航目标契约。
//
// Canonical owner: server/modules/notification/contract.
// 导航值由本包作为唯一权威维护，直到本包明确标记替换或移除。
type NavigationKind string

// String 返回规范化的导航目标类型值。
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

// ValidNavigationKind 判断 value 是否为已知的导航目标契约。
func ValidNavigationKind(value NavigationKind) bool {
	switch value {
	case NavigationAuditIncident, NavigationAuditLog, NavigationSchedulerRun, NavigationSystemConfigItem, NavigationModuleRuntimeItem:
		return true
	default:
		return false
	}
}
