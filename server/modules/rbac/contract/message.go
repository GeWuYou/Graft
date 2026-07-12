package contract

// MessageKey identifies a stable rbac module message key.
type MessageKey string

// String returns the canonical menu message key value.
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
