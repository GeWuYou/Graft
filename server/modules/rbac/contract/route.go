package contract

// JoinRoute 拼接路由组路径和路由片段，并返回组合后的路径。
func JoinRoute(group, fragment string) string {
	return group + fragment
}

const (
	// RoleListMenuPath 标识角色管理页面的规范菜单路径。
	RoleListMenuPath = "/security/roles"
	// PermissionListMenuPath 标识权限管理页面的规范菜单路径。
	PermissionListMenuPath = "/security/permissions"

	// RolesGroup 标识角色管理路由组。
	RolesGroup = "/roles"
	// RoleCollection 标识角色集合接口在角色路由组下的路径片段。
	RoleCollection = ""
	// RoleSavedViewsRoute 标识角色列表私有保存筛选集合接口的路径片段。
	RoleSavedViewsRoute = "/saved-views"
	// RoleSavedViewRoute 标识单个角色列表私有保存筛选接口的路径片段。
	RoleSavedViewRoute = "/saved-views/:viewId"
	// RoleDetailRoute 标识单个角色详情接口的路径片段。
	RoleDetailRoute = "/:id"
	// RoleUpdateRoute 标识角色更新接口的路径片段。
	RoleUpdateRoute = "/:id/update"
	// RoleCloneRoute 标识从现有角色复制为自定义角色的路径片段。
	RoleCloneRoute = "/:id/clone"
	// RoleStatusRoute 标识角色状态更新接口的路径片段。
	RoleStatusRoute = "/:id/status"
	// RoleDeleteRoute 标识角色软删除接口的路径片段。
	RoleDeleteRoute = "/:id/delete"
	// RolePermissionReplaceRoute 标识覆盖角色权限绑定接口的路径片段。
	RolePermissionReplaceRoute = "/:id/permissions/replace"
	// RolePermissionAddRoute 标识追加角色权限绑定接口的路径片段。
	RolePermissionAddRoute = "/:id/permissions/add"
	// RolePermissionRemoveRoute 标识移除角色权限绑定接口的路径片段。
	RolePermissionRemoveRoute = "/:id/permissions/remove"
	// RolePermissionBindingRoute 标识角色权限绑定快照接口的路径片段。
	RolePermissionBindingRoute = "/:id/permissions"

	// PermissionsGroup 标识权限管理路由组。
	PermissionsGroup = "/permissions"
	// PermissionCollection 标识权限集合接口在权限路由组下的路径片段。
	PermissionCollection = ""
	// PermissionSavedViewsRoute 标识权限列表私有保存筛选集合接口的路径片段。
	PermissionSavedViewsRoute = "/saved-views"
	// PermissionSavedViewRoute 标识单个权限列表私有保存筛选接口的路径片段。
	PermissionSavedViewRoute = "/saved-views/:viewId"
	// PermissionDetailRoute 标识单个权限详情接口的路径片段。
	PermissionDetailRoute = "/:id"

	// UsersGroup 标识由 rbac 模块拥有的用户角色绑定路由组。
	UsersGroup = "/users"
	// UserRoleBindingRoute 标识用户角色绑定快照接口的路径片段。
	UserRoleBindingRoute = "/:id/roles"
	// UserRoleReplaceRoute 标识覆盖用户角色绑定接口的路径片段。
	UserRoleReplaceRoute = "/:id/roles/replace"
	// UserRoleAddRoute 标识追加用户角色绑定接口的路径片段。
	UserRoleAddRoute = "/:id/roles/add"
	// UserRoleRemoveRoute 标识移除用户角色绑定接口的路径片段。
	UserRoleRemoveRoute = "/:id/roles/remove"
	// BatchUserRoleReplaceRoute 标识批量覆盖用户角色绑定接口的路径片段。
	BatchUserRoleReplaceRoute = "/roles/replace"
	// BatchUserRoleAddRoute 标识批量追加用户角色绑定接口的路径片段。
	BatchUserRoleAddRoute = "/roles/add"
	// BatchUserRoleRemoveRoute 标识批量移除用户角色绑定接口的路径片段。
	BatchUserRoleRemoveRoute = "/roles/remove"
)
