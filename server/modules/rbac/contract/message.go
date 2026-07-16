package contract

// MessageKey 标识 RBAC 模块稳定的消息键。
type MessageKey string

// String 返回 canonical 消息键值。
func (k MessageKey) String() string {
	return string(k)
}

const (
	// RoleListMenuTitle identifies the localized title for the role list menu.
	RoleListMenuTitle MessageKey = "menu.security.roles.title"
	// PermissionListMenuTitle identifies the localized title for the permission list menu.
	PermissionListMenuTitle MessageKey = "menu.security.permissions.title"
	// AuditRolePermissionsAdded identifies role-permission append audit messages.
	AuditRolePermissionsAdded MessageKey = "rbac.audit.rolePermissionsAdded"
	// AuditRolePermissionsRemoved identifies role-permission removal audit messages.
	AuditRolePermissionsRemoved MessageKey = "rbac.audit.rolePermissionsRemoved"
)
