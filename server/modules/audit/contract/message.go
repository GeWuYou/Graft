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
	// AuditRootMenuTitle 是审计根菜单的本地化标题键。
	AuditRootMenuTitle MenuMessageKey = "menu.audit.title"
	// AuditLogMenuTitle 是审计日志菜单的本地化标题键。
	AuditLogMenuTitle MenuMessageKey = "menu.audit.logs.title"

	// AuditTargetLabelUser 是内置用户目标的本地化标签键。
	AuditTargetLabelUser TargetLabelMessageKey = "audit.target.user"
	// AuditTargetLabelRole 是内置角色目标的本地化标签键。
	AuditTargetLabelRole TargetLabelMessageKey = "audit.target.role"
	// AuditTargetLabelPermission 是内置权限目标的本地化标签键。
	AuditTargetLabelPermission TargetLabelMessageKey = "audit.target.permission"
	// AuditTargetLabelAudit 是内置审计目标的本地化标签键。
	AuditTargetLabelAudit TargetLabelMessageKey = "audit.target.audit"
	// AuditTargetLabelServerStatus 是内置服务状态目标的本地化标签键。
	AuditTargetLabelServerStatus TargetLabelMessageKey = "audit.target.serverStatus"
	// AuditTargetLabelAuth 是内置认证目标的本地化标签键。
	AuditTargetLabelAuth TargetLabelMessageKey = "audit.target.auth"
)
