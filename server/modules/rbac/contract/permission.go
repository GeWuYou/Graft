package contract

// PermissionCode identifies a stable rbac module permission contract.
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
	// RoleStatusUpdatePermission identifies lifecycle status updates for roles.
	RoleStatusUpdatePermission PermissionCode = "role.status.update"
	// RoleDeletePermission identifies destructive role deletion access.
	RoleDeletePermission PermissionCode = "role.delete"
	// RolePermissionAssignPermission identifies write access to role-permission bindings.
	RolePermissionAssignPermission PermissionCode = "role.permission.assign"
	// PermissionReadPermission identifies read access to permission-management data.
	PermissionReadPermission PermissionCode = "permission.read"
	// UserRoleReadPermission identifies read access to user-role binding snapshots.
	UserRoleReadPermission PermissionCode = "user.role.read"
	// UserRoleAssignPermission identifies write access to user-role bindings.
	UserRoleAssignPermission PermissionCode = "user.role.assign"

	// RoleRead is the canonical permission used by rbac module consumers.
	RoleRead PermissionCode = RoleReadPermission
	// RoleCreate is the canonical permission used by rbac module consumers.
	RoleCreate PermissionCode = RoleCreatePermission
	// RoleUpdate is the canonical permission used by rbac module consumers.
	RoleUpdate PermissionCode = RoleUpdatePermission
	// RoleStatusUpdate is the canonical permission used by rbac module consumers.
	RoleStatusUpdate PermissionCode = RoleStatusUpdatePermission
	// RoleDelete is the canonical permission used by rbac module consumers.
	RoleDelete PermissionCode = RoleDeletePermission
	// RolePermissionAssign is the canonical permission used by rbac module consumers.
	RolePermissionAssign PermissionCode = RolePermissionAssignPermission
	// PermissionRead is the canonical permission used by rbac module consumers.
	PermissionRead PermissionCode = PermissionReadPermission
	// UserRoleRead is the canonical permission used by rbac module consumers.
	UserRoleRead PermissionCode = UserRoleReadPermission
	// UserRoleAssign is the canonical permission used by rbac module consumers.
	UserRoleAssign PermissionCode = UserRoleAssignPermission
)
