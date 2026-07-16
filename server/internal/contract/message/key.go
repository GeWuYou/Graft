// Package message 定义服务端运行时共享的稳定本地化消息键契约。
package message

// Key 标识稳定的本地化消息契约键。
type Key string

// String 返回规范消息键字符串。
func (k Key) String() string {
	return string(k)
}

//nolint:gosec // Canonical message-key literals are contract values, not credentials.
const (
	// AuthCurrentPasswordInvalid 表示当前密码校验失败。
	AuthCurrentPasswordInvalid Key = "auth.current_password_invalid"

	// AuthForbidden 表示已认证调用方因权限不足而被拒绝。
	AuthForbidden Key = "auth.forbidden"

	// AuthInvalidCredentials 表示登录凭据校验失败。
	AuthInvalidCredentials Key = "auth.invalid_credentials"

	// AuthInvalidRefreshSession 表示刷新会话无效或已过期。
	AuthInvalidRefreshSession Key = "auth.invalid_refresh_session"

	// AuthMissingActor 表示请求缺少已认证主体。
	AuthMissingActor Key = "auth.missing_actor"

	// AuthMissingPermission 表示调用方缺少所需权限。
	AuthMissingPermission Key = "auth.missing_permission"

	// AuthPasswordPolicyViolation 表示密码策略校验失败。
	AuthPasswordPolicyViolation Key = "auth.password_policy_violation"

	// AuthPasswordReuseForbidden 表示密码复用被策略禁止。
	AuthPasswordReuseForbidden Key = "auth.password_reuse_forbidden"

	// AuthSessionNotFound 表示会话不存在或已失效。
	AuthSessionNotFound Key = "auth.session_not_found"

	// AuthTokenExpired 表示访问令牌已过期。
	AuthTokenExpired Key = "auth.token_expired"

	// AuthTokenInvalid 表示访问令牌格式错误或无效。
	AuthTokenInvalid Key = "auth.token_invalid"

	// AuthTokenMissing 表示请求缺少访问令牌。
	AuthTokenMissing Key = "auth.token_missing"

	// CommonInternalError 表示通过统一响应封装返回的服务端内部错误。
	CommonInternalError Key = "common.internal_error"

	// CommonInvalidArgument 表示请求参数无效。
	CommonInvalidArgument Key = "common.invalid_argument"

	// CommonNotFound 表示资源不存在或不在调用方可见范围内。
	CommonNotFound Key = "common.not_found"

	// CommonConjunction 表示运行时界面文案共用的连接词。
	CommonConjunction Key = "common.conjunction"

	// CommonCopyright 表示运行时界面文案共用的版权页脚标签。
	CommonCopyright Key = "common.copyright"

	// RbacCannotRemoveOwnAdminRole 表示替换内置管理员角色时阻止自我锁定。
	RbacCannotRemoveOwnAdminRole Key = "rbac.cannot_remove_own_admin_role"

	// RbacBuiltinAdminPermissionsImmutable 表示内置管理员角色权限不可修改。
	RbacBuiltinAdminPermissionsImmutable Key = "rbac.builtin_admin_permissions_immutable"

	// UserNotFound 表示认证相关流程中找不到用户。
	UserNotFound Key = "user.not_found"

	// UserProtectedDefaultAdminImmutable 表示服务端阻止修改受保护的默认管理员。
	UserProtectedDefaultAdminImmutable Key = "user.protected_default_admin_immutable"

	// RoleNotFound 表示 RBAC 管理流程中找不到角色。
	RoleNotFound Key = "role.not_found"

	// PermissionNotFound 表示 RBAC 管理流程中找不到权限。
	PermissionNotFound Key = "permission.not_found"
)
