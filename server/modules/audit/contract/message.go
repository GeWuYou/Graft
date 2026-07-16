package contract

// MenuMessageKey 标识审计模块菜单标题使用的稳定消息键。
type MenuMessageKey string

// TargetLabelMessageKey 标识内置审计目标类型使用的稳定本地化标签键。
type TargetLabelMessageKey string

// String 返回规范化的菜单消息键值。
func (k MenuMessageKey) String() string {
	return string(k)
}

// String 返回规范化的目标标签消息键值。
func (k TargetLabelMessageKey) String() string {
	return string(k)
}

const (
	// AuditRootMenuTitle identifies the localized title for the audit root menu.
	AuditRootMenuTitle MenuMessageKey = "menu.audit.title"
	// AuditLogMenuTitle identifies the localized title for the audit-log menu.
	AuditLogMenuTitle MenuMessageKey = "menu.audit.logs.title"

	// AuditTargetLabelUser identifies the localized label for built-in user targets.
	AuditTargetLabelUser TargetLabelMessageKey = "audit.target.user"
	// AuditTargetLabelRole identifies the localized label for built-in role targets.
	AuditTargetLabelRole TargetLabelMessageKey = "audit.target.role"
	// AuditTargetLabelPermission identifies the localized label for built-in permission targets.
	AuditTargetLabelPermission TargetLabelMessageKey = "audit.target.permission"
	// AuditTargetLabelAudit identifies the localized label for built-in audit targets.
	AuditTargetLabelAudit TargetLabelMessageKey = "audit.target.audit"
	// AuditTargetLabelServerStatus identifies the localized label for built-in server-status targets.
	AuditTargetLabelServerStatus TargetLabelMessageKey = "audit.target.serverStatus"
	// AuditTargetLabelAuth identifies the localized label for built-in authentication targets.
	AuditTargetLabelAuth TargetLabelMessageKey = "audit.target.auth"
)
