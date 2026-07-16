// Package contract 定义 user 模块稳定的路由、权限和消息契约。
package contract

// PermissionCode 标识 user 模块稳定的权限契约。
type PermissionCode string

// String 返回权限码的 wire-format 字符串。
func (c PermissionCode) String() string {
	return string(c)
}

const (
	// UserReadPermission 标识用户管理数据的读取权限。
	UserReadPermission PermissionCode = "user.read"

	// UserCreatePermission 标识用户管理数据的创建权限。
	UserCreatePermission PermissionCode = "user.create"

	// UserUpdatePermission 标识用户管理数据的更新权限。
	UserUpdatePermission PermissionCode = "user.update"

	// UserDisablePermission 标识用户禁用和删除权限。
	UserDisablePermission PermissionCode = "user.disable"

	// UserSessionReadPermission 标识刷新会话状态的读取权限。
	UserSessionReadPermission PermissionCode = "user.session.read"

	// UserSessionRevokePermission 标识刷新会话撤销权限。
	UserSessionRevokePermission PermissionCode = "user.session.revoke"

	// UserRead 是 user 模块消费者使用的用户读取规范权限码。
	UserRead PermissionCode = UserReadPermission

	// UserCreate 是 user 模块消费者使用的用户创建规范权限码。
	UserCreate PermissionCode = UserCreatePermission

	// UserUpdate 是 user 模块消费者使用的用户更新规范权限码。
	UserUpdate PermissionCode = UserUpdatePermission

	// UserDisable 是 user 模块消费者使用的用户禁用规范权限码。
	UserDisable PermissionCode = UserDisablePermission

	// UserSessionRead 是 user 模块消费者使用的会话读取规范权限码。
	UserSessionRead PermissionCode = UserSessionReadPermission

	// UserSessionRevoke 是 user 模块消费者使用的会话撤销规范权限码。
	UserSessionRevoke PermissionCode = UserSessionRevokePermission
)
