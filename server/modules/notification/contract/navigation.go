package contract

// NavigationKind 标识通知稳定的导航目标契约。
//
// 规范权威归属：server/modules/notification/contract。
// 导航值由本包作为唯一权威维护，直到本包明确标记替换或移除。
type NavigationKind string

// String 返回规范化的导航目标类型值。
func (k NavigationKind) String() string {
	return string(k)
}

const (
	// NavigationAuditIncident 指向 audit 异常详情；生命周期为 stable。
	NavigationAuditIncident NavigationKind = "AUDIT_INCIDENT"
	// NavigationAuditLog 指向 audit 日志详情；生命周期为 stable。
	NavigationAuditLog NavigationKind = "AUDIT_LOG"
	// NavigationSchedulerRun 指向定时任务运行详情；生命周期为 stable。
	NavigationSchedulerRun NavigationKind = "SCHEDULER_RUN"
	// NavigationSystemConfigItem 预留给 system-config 配置项；生命周期为 experimental。
	NavigationSystemConfigItem NavigationKind = "SYSTEM_CONFIG_ITEM"
	// NavigationModuleRuntimeItem 预留给模块运行时详情；生命周期为 experimental。
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
