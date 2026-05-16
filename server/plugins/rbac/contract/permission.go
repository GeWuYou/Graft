package contract

// PermissionCode identifies a stable rbac-plugin permission contract.
type PermissionCode string

// String returns the wire-format permission code.
func (c PermissionCode) String() string {
	return string(c)
}

const (
	// RoleReadPermission identifies read access to role-management data.
	RoleReadPermission PermissionCode = "role.read"
	// RoleCreatePermission identifies create access to role-management data.
	RoleCreatePermission PermissionCode = "role.create"
	// RoleUpdatePermission identifies update access to role-management data.
	RoleUpdatePermission PermissionCode = "role.update"
	// RolePermissionAssignPermission identifies write access to role-permission bindings.
	RolePermissionAssignPermission PermissionCode = "role.permission.assign"
	// PermissionReadPermission identifies read access to permission-management data.
	PermissionReadPermission PermissionCode = "permission.read"
	// UserRoleAssignPermission identifies write access to user-role bindings.
	UserRoleAssignPermission PermissionCode = "user.role.assign"

	// RoleRead is the canonical permission used by rbac-plugin consumers.
	RoleRead PermissionCode = RoleReadPermission
	// RoleCreate is the canonical permission used by rbac-plugin consumers.
	RoleCreate PermissionCode = RoleCreatePermission
	// RoleUpdate is the canonical permission used by rbac-plugin consumers.
	RoleUpdate PermissionCode = RoleUpdatePermission
	// RolePermissionAssign is the canonical permission used by rbac-plugin consumers.
	RolePermissionAssign PermissionCode = RolePermissionAssignPermission
	// PermissionRead is the canonical permission used by rbac-plugin consumers.
	PermissionRead PermissionCode = PermissionReadPermission
	// UserRoleAssign is the canonical permission used by rbac-plugin consumers.
	UserRoleAssign PermissionCode = UserRoleAssignPermission
)
