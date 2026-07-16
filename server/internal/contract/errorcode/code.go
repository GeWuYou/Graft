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
	// AuthCurrentPasswordInvalid identifies current-password validation failures.
	AuthCurrentPasswordInvalid Code = "AUTH_CURRENT_PASSWORD_INVALID"

	// AuthForbidden identifies permission-denied failures for authenticated callers.
	AuthForbidden Code = "AUTH_FORBIDDEN"

	// AuthInvalidCredentials identifies login credential validation failures.
	AuthInvalidCredentials Code = "AUTH_INVALID_CREDENTIALS"

	// AuthInvalidRefreshSession identifies invalid or expired refresh-session failures.
	AuthInvalidRefreshSession Code = "AUTH_INVALID_REFRESH_SESSION"

	// AuthMissingActor identifies missing authenticated principal failures.
	AuthMissingActor Code = "AUTH_MISSING_ACTOR"

	// AuthMissingPermission identifies missing-required-permission failures.
	AuthMissingPermission Code = "AUTH_MISSING_PERMISSION"

	// AuthPasswordPolicyViolation identifies password-policy validation failures.
	AuthPasswordPolicyViolation Code = "AUTH_PASSWORD_POLICY_VIOLATION"

	// AuthPasswordReuseForbidden identifies disallowed password-reuse failures.
	AuthPasswordReuseForbidden Code = "AUTH_PASSWORD_REUSE_FORBIDDEN"

	// AuthSessionNotFound identifies missing or inactive session failures.
	AuthSessionNotFound Code = "AUTH_SESSION_NOT_FOUND"

	// AuthTokenExpired identifies expired access-token failures.
	AuthTokenExpired Code = "AUTH_TOKEN_EXPIRED"

	// AuthTokenInvalid identifies malformed or invalid access-token failures.
	AuthTokenInvalid Code = "AUTH_TOKEN_INVALID"

	// AuthTokenMissing identifies missing access-token failures.
	AuthTokenMissing Code = "AUTH_TOKEN_MISSING"

	// CommonInternalError identifies internal server failures surfaced through the unified envelope.
	CommonInternalError Code = "COMMON_INTERNAL_ERROR"

	// CommonInvalidArgument identifies invalid request parameter failures.
	CommonInvalidArgument Code = "COMMON_INVALID_ARGUMENT"

	// CommonNotFound identifies resources hidden because they do not exist or are outside the caller scope.
	CommonNotFound Code = "COMMON_NOT_FOUND"

	// RbacCannotRemoveOwnAdminRole identifies self-lockout prevention failures for builtin admin role replacement.
	RbacCannotRemoveOwnAdminRole Code = "RBAC_CANNOT_REMOVE_OWN_ADMIN_ROLE"

	// RbacBuiltinAdminPermissionsImmutable identifies builtin admin permission mutation failures.
	RbacBuiltinAdminPermissionsImmutable Code = "RBAC_BUILTIN_ADMIN_PERMISSIONS_IMMUTABLE"

	// OK identifies the stable success response code.
	OK Code = "OK"

	// UserNotFound identifies missing-user failures surfaced by auth-adjacent flows.
	UserNotFound Code = "USER_NOT_FOUND"

	// UserProtectedDefaultAdminImmutable identifies protected default-admin mutations blocked by server authority.
	UserProtectedDefaultAdminImmutable Code = "USER_PROTECTED_DEFAULT_ADMIN_IMMUTABLE"

	// RoleNotFound identifies missing-role failures surfaced by RBAC management flows.
	RoleNotFound Code = "ROLE_NOT_FOUND"

	// PermissionNotFound identifies missing-permission failures surfaced by RBAC management flows.
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
