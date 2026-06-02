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
	// RoleDetailRoute identifies the single-role detail endpoint route fragment.
	RoleDetailRoute = "/:id"
	// RoleUpdateRoute identifies the role update endpoint route fragment.
	RoleUpdateRoute = "/:id/update"
	// RoleStatusRoute identifies the role status endpoint route fragment.
	RoleStatusRoute = "/:id/status"
	// RoleDeleteRoute identifies the role soft-delete endpoint route fragment.
	RoleDeleteRoute = "/:id/delete"
	// RolePermissionReplaceRoute identifies the role-permission replace endpoint route fragment.
	RolePermissionReplaceRoute = "/:id/permissions/replace"
	// RolePermissionAddRoute identifies the role-permission add endpoint route fragment.
	RolePermissionAddRoute = "/:id/permissions/add"
	// RolePermissionRemoveRoute identifies the role-permission remove endpoint route fragment.
	RolePermissionRemoveRoute = "/:id/permissions/remove"
	// RolePermissionBindingRoute identifies the role-permission binding snapshot endpoint route fragment.
	RolePermissionBindingRoute = "/:id/permissions"

	// PermissionsGroup identifies the permission-management route group.
	PermissionsGroup = "/permissions"
	// PermissionCollection identifies the collection endpoint route fragment on the permissions group.
	PermissionCollection = ""
	// PermissionDetailRoute identifies the single-permission detail endpoint route fragment.
	PermissionDetailRoute = "/:id"

	// UsersGroup identifies the user-role assignment route group owned by the rbac module.
	UsersGroup = "/users"
	// UserRoleBindingRoute identifies the user-role binding snapshot endpoint route fragment.
	UserRoleBindingRoute = "/:id/roles"
	// UserRoleReplaceRoute identifies the user-role replace endpoint route fragment.
	UserRoleReplaceRoute = "/:id/roles/replace"
	// UserRoleAddRoute identifies the user-role add endpoint route fragment.
	UserRoleAddRoute = "/:id/roles/add"
	// UserRoleRemoveRoute identifies the user-role remove endpoint route fragment.
	UserRoleRemoveRoute = "/:id/roles/remove"
	// BatchUserRoleReplaceRoute identifies the batch user-role replace endpoint route fragment.
	BatchUserRoleReplaceRoute = "/roles/replace"
	// BatchUserRoleAddRoute identifies the batch user-role add endpoint route fragment.
	BatchUserRoleAddRoute = "/roles/add"
	// BatchUserRoleRemoveRoute identifies the batch user-role remove endpoint route fragment.
	BatchUserRoleRemoveRoute = "/roles/remove"
)
