package contract

// MessageKey 标识 RBAC 模块稳定的消息键。
type MessageKey string

// String 返回 canonical 消息键值。
func (k MessageKey) String() string {
	return string(k)
}

const (
	// RoleListMenuTitle 标识角色列表菜单的本地化标题键。
	RoleListMenuTitle MessageKey = "menu.security.roles.title"
	// PermissionListMenuTitle 标识权限列表菜单的本地化标题键。
	PermissionListMenuTitle MessageKey = "menu.security.permissions.title"
	// AuditRolePermissionsAdded 标识追加角色权限绑定时使用的审计消息键。
	AuditRolePermissionsAdded MessageKey = "rbac.audit.rolePermissionsAdded"
	// AuditRolePermissionsRemoved 标识移除角色权限绑定时使用的审计消息键。
	AuditRolePermissionsRemoved MessageKey = "rbac.audit.rolePermissionsRemoved"
)
