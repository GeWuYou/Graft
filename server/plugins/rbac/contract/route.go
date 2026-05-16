package contract

// JoinRoute combines a route group path with a route fragment.
func JoinRoute(group, fragment string) string {
	return group + fragment
}

const (
	// RolesGroup identifies the role-management route group.
	RolesGroup = "/roles"
	// RoleCollection identifies the collection endpoint route fragment on the roles group.
	RoleCollection = ""
	// RoleUpdateRoute identifies the role update endpoint route fragment.
	RoleUpdateRoute = "/:id/update"
	// RolePermissionAssignRoute identifies the role-permission assignment endpoint route fragment.
	RolePermissionAssignRoute = "/:id/permissions/assign"

	// PermissionsGroup identifies the permission-management route group.
	PermissionsGroup = "/permissions"
	// PermissionCollection identifies the collection endpoint route fragment on the permissions group.
	PermissionCollection = ""

	// UsersGroup identifies the user-role assignment route group owned by the rbac plugin.
	UsersGroup = "/users"
	// UserRoleAssignRoute identifies the user-role assignment endpoint route fragment.
	UserRoleAssignRoute = "/:id/roles/assign"
)
