// Package errorcode 定义服务端运行时共享的稳定 API 响应码契约。
package errorcode

import (
	"strings"

	messagecontract "graft/server/internal/contract/message"
)

// Code 标识稳定的 API 响应码契约。
type Code string

// String 返回规范响应码字符串。
func (c Code) String() string {
	return string(c)
}

//nolint:gosec // Canonical response-code literals are contract values, not credentials.
const (
	// AuthCurrentPasswordInvalid 表示当前密码校验失败。
	AuthCurrentPasswordInvalid Code = "AUTH_CURRENT_PASSWORD_INVALID"

	// AuthForbidden 表示已认证调用方因权限不足而被拒绝。
	AuthForbidden Code = "AUTH_FORBIDDEN"

	// AuthInvalidCredentials 表示登录凭据校验失败。
	AuthInvalidCredentials Code = "AUTH_INVALID_CREDENTIALS"

	// AuthInvalidRefreshSession 表示刷新会话无效或已过期。
	AuthInvalidRefreshSession Code = "AUTH_INVALID_REFRESH_SESSION"

	// AuthMissingActor 表示请求缺少已认证主体。
	AuthMissingActor Code = "AUTH_MISSING_ACTOR"

	// AuthMissingPermission 表示调用方缺少所需权限。
	AuthMissingPermission Code = "AUTH_MISSING_PERMISSION"

	// AuthPasswordPolicyViolation 表示密码策略校验失败。
	AuthPasswordPolicyViolation Code = "AUTH_PASSWORD_POLICY_VIOLATION"

	// AuthPasswordReuseForbidden 表示密码复用被策略禁止。
	AuthPasswordReuseForbidden Code = "AUTH_PASSWORD_REUSE_FORBIDDEN"

	// AuthSessionNotFound 表示会话不存在或已失效。
	AuthSessionNotFound Code = "AUTH_SESSION_NOT_FOUND"

	// AuthTokenExpired 表示访问令牌已过期。
	AuthTokenExpired Code = "AUTH_TOKEN_EXPIRED"

	// AuthTokenInvalid 表示访问令牌格式错误或无效。
	AuthTokenInvalid Code = "AUTH_TOKEN_INVALID"

	// AuthTokenMissing 表示请求缺少访问令牌。
	AuthTokenMissing Code = "AUTH_TOKEN_MISSING"

	// CommonInternalError 表示通过统一响应封装返回的服务端内部错误。
	CommonInternalError Code = "COMMON_INTERNAL_ERROR"

	// CommonInvalidArgument 表示请求参数无效。
	CommonInvalidArgument Code = "COMMON_INVALID_ARGUMENT"

	// CommonNotFound 表示资源不存在或不在调用方可见范围内。
	CommonNotFound Code = "COMMON_NOT_FOUND"

	// RbacCannotRemoveOwnAdminRole 表示替换内置管理员角色时阻止自我锁定。
	RbacCannotRemoveOwnAdminRole Code = "RBAC_CANNOT_REMOVE_OWN_ADMIN_ROLE"

	// RbacBuiltinAdminPermissionsImmutable 表示内置管理员权限不可修改。
	RbacBuiltinAdminPermissionsImmutable Code = "RBAC_BUILTIN_ADMIN_PERMISSIONS_IMMUTABLE"

	// OK 表示稳定的成功响应码。
	OK Code = "OK"

	// UserNotFound 表示认证相关流程中找不到用户。
	UserNotFound Code = "USER_NOT_FOUND"

	// UserProtectedDefaultAdminImmutable 表示服务端阻止修改受保护的默认管理员。
	UserProtectedDefaultAdminImmutable Code = "USER_PROTECTED_DEFAULT_ADMIN_IMMUTABLE"

	// RoleNotFound 表示 RBAC 管理流程中找不到角色。
	RoleNotFound Code = "ROLE_NOT_FOUND"

	// PermissionNotFound 表示 RBAC 管理流程中找不到权限。
	PermissionNotFound Code = "PERMISSION_NOT_FOUND"
)

var messageKeyCodes = map[messagecontract.Key]Code{
	messagecontract.AuthCurrentPasswordInvalid:           AuthCurrentPasswordInvalid,
	messagecontract.AuthForbidden:                        AuthForbidden,
	messagecontract.AuthInvalidCredentials:               AuthInvalidCredentials,
	messagecontract.AuthInvalidRefreshSession:            AuthInvalidRefreshSession,
	messagecontract.AuthMissingActor:                     AuthMissingActor,
	messagecontract.AuthMissingPermission:                AuthMissingPermission,
	messagecontract.AuthPasswordPolicyViolation:          AuthPasswordPolicyViolation,
	messagecontract.AuthPasswordReuseForbidden:           AuthPasswordReuseForbidden,
	messagecontract.AuthSessionNotFound:                  AuthSessionNotFound,
	messagecontract.AuthTokenExpired:                     AuthTokenExpired,
	messagecontract.AuthTokenInvalid:                     AuthTokenInvalid,
	messagecontract.AuthTokenMissing:                     AuthTokenMissing,
	messagecontract.CommonInternalError:                  CommonInternalError,
	messagecontract.CommonInvalidArgument:                CommonInvalidArgument,
	messagecontract.CommonNotFound:                       CommonNotFound,
	messagecontract.RbacCannotRemoveOwnAdminRole:         RbacCannotRemoveOwnAdminRole,
	messagecontract.RbacBuiltinAdminPermissionsImmutable: RbacBuiltinAdminPermissionsImmutable,
	messagecontract.PermissionNotFound:                   PermissionNotFound,
	messagecontract.RoleNotFound:                         RoleNotFound,
	messagecontract.UserNotFound:                         UserNotFound,
	messagecontract.UserProtectedDefaultAdminImmutable:   UserProtectedDefaultAdminImmutable,
}

// FromMessageKey 将稳定消息键解析为规范响应码；未登记的键按规范化后的键名生成回退值。
func FromMessageKey(key messagecontract.Key) Code {
	if code, ok := messageKeyCodes[key]; ok {
		return code
	}

	replacer := strings.NewReplacer(".", "_", "-", "_")
	return Code(strings.ToUpper(replacer.Replace(strings.TrimSpace(key.String()))))
}
