// Package message defines stable localized message-key contracts shared by the server runtime.
package message

// Key identifies a stable localized message contract key.
type Key string

// String returns the canonical message key value.
func (k Key) String() string {
	return string(k)
}

//nolint:gosec // Canonical message-key literals are contract values, not credentials.
const (
	// AuthCurrentPasswordInvalid identifies current-password validation failures.
	AuthCurrentPasswordInvalid Key = "auth.current_password_invalid"

	// AuthForbidden identifies permission-denied failures for authenticated callers.
	AuthForbidden Key = "auth.forbidden"

	// AuthInvalidCredentials identifies login credential validation failures.
	AuthInvalidCredentials Key = "auth.invalid_credentials"

	// AuthInvalidRefreshSession identifies invalid or expired refresh-session failures.
	AuthInvalidRefreshSession Key = "auth.invalid_refresh_session"

	// AuthMissingActor identifies missing authenticated principal failures.
	AuthMissingActor Key = "auth.missing_actor"

	// AuthMissingPermission identifies missing-required-permission failures.
	AuthMissingPermission Key = "auth.missing_permission"

	// AuthPasswordPolicyViolation identifies password-policy validation failures.
	AuthPasswordPolicyViolation Key = "auth.password_policy_violation"

	// AuthPasswordReuseForbidden identifies disallowed password-reuse failures.
	AuthPasswordReuseForbidden Key = "auth.password_reuse_forbidden"

	// AuthSessionNotFound identifies missing or inactive session failures.
	AuthSessionNotFound Key = "auth.session_not_found"

	// AuthTokenExpired identifies expired access-token failures.
	AuthTokenExpired Key = "auth.token_expired"

	// AuthTokenInvalid identifies malformed or invalid access-token failures.
	AuthTokenInvalid Key = "auth.token_invalid"

	// AuthTokenMissing identifies missing access-token failures.
	AuthTokenMissing Key = "auth.token_missing"

	// CommonInternalError identifies internal server failures surfaced through the unified envelope.
	CommonInternalError Key = "common.internal_error"

	// CommonInvalidArgument identifies invalid request parameter failures.
	CommonInvalidArgument Key = "common.invalid_argument"

	// CommonConjunction identifies the shared conjunction label used by runtime UI copy.
	CommonConjunction Key = "common.conjunction"

	// CommonCopyright identifies the shared copyright footer label used by runtime UI copy.
	CommonCopyright Key = "common.copyright"

	// RbacCannotRemoveOwnAdminRole identifies self-lockout prevention failures for builtin admin role replacement.
	RbacCannotRemoveOwnAdminRole Key = "rbac.cannot_remove_own_admin_role"

	// UserNotFound identifies missing-user failures surfaced by auth-adjacent flows.
	UserNotFound Key = "user.not_found"

	// RoleNotFound identifies missing-role failures surfaced by RBAC management flows.
	RoleNotFound Key = "role.not_found"
)
