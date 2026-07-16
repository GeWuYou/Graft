package contract

// PermissionCode 标识 RBAC 模块稳定的权限契约。
type PermissionCode string

// String 返回权限码的 wire-format 字符串。
func (c PermissionCode) String() string {
	return string(c)
}

const (
	// RoleReadPermission 标识角色管理数据的读取权限。
	RoleReadPermission PermissionCode = "role.read"
	// RoleCreatePermission 标识角色管理数据的创建权限。
	RoleCreatePermission PermissionCode = "role.create"
	// RoleUpdatePermission 标识角色管理数据的更新权限。
	RoleUpdatePermission PermissionCode = "role.update"
	// RoleStatusUpdatePermission 标识角色生命周期状态更新权限。
	RoleStatusUpdatePermission PermissionCode = "role.status.update"
	// RoleDeletePermission 标识角色删除等破坏性操作权限。
	RoleDeletePermission PermissionCode = "role.delete"
	// RolePermissionAssignPermission 标识角色权限绑定写入权限。
	RolePermissionAssignPermission PermissionCode = "role.permission.assign"
	// PermissionReadPermission 标识权限管理数据的读取权限。
	PermissionReadPermission PermissionCode = "permission.read"
	// UserRoleReadPermission 标识用户角色绑定快照的读取权限。
	UserRoleReadPermission PermissionCode = "user.role.read"
	// UserRoleAssignPermission 标识用户角色绑定写入权限。
	UserRoleAssignPermission PermissionCode = "user.role.assign"

	// RoleRead 是 rbac 模块消费者使用的角色读取规范权限码。
	RoleRead PermissionCode = RoleReadPermission
	// RoleCreate 是 rbac 模块消费者使用的角色创建规范权限码。
	RoleCreate PermissionCode = RoleCreatePermission
	// RoleUpdate 是 rbac 模块消费者使用的角色更新规范权限码。
	RoleUpdate PermissionCode = RoleUpdatePermission
	// RoleStatusUpdate 是 rbac 模块消费者使用的角色状态更新规范权限码。
	RoleStatusUpdate PermissionCode = RoleStatusUpdatePermission
	// RoleDelete 是 rbac 模块消费者使用的角色删除规范权限码。
	RoleDelete PermissionCode = RoleDeletePermission
	// RolePermissionAssign 是 rbac 模块消费者使用的角色权限绑定规范权限码。
	RolePermissionAssign PermissionCode = RolePermissionAssignPermission
	// PermissionRead 是 rbac 模块消费者使用的权限读取规范权限码。
	PermissionRead PermissionCode = PermissionReadPermission
	// UserRoleRead 是 rbac 模块消费者使用的用户角色读取规范权限码。
	UserRoleRead PermissionCode = UserRoleReadPermission
	// UserRoleAssign 是 rbac 模块消费者使用的用户角色绑定规范权限码。
	UserRoleAssign PermissionCode = UserRoleAssignPermission
)
