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

	// PermissionsGroup identifies the permission-management route group.
	PermissionsGroup = "/permissions"
	// PermissionCollection identifies the collection endpoint route fragment on the permissions group.
	PermissionCollection = ""
)
