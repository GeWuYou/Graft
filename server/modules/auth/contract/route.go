package contract

// JoinRoute combines a route group path with a route fragment.
func JoinRoute(group, fragment string) string {
	return group + fragment
}

//nolint:gosec // Canonical route fragments are API contracts, not credentials.
const (
	// AuthGroup identifies the auth module route group.
	AuthGroup = "/auth"

	// AuthLogin identifies the login endpoint route fragment.
	AuthLogin = "/login"

	// AuthRefresh identifies the refresh endpoint route fragment.
	AuthRefresh = "/refresh"

	// AuthLogout identifies the logout endpoint route fragment.
	AuthLogout = "/logout"

	// AuthSessionsRevokeAll identifies the current-user revoke-all endpoint route fragment.
	AuthSessionsRevokeAll = "/sessions/revoke-all"

	// AuthSessionsRevokeOthers identifies the current-user revoke-others endpoint route fragment.
	AuthSessionsRevokeOthers = "/sessions/revoke-others"

	// AuthSessions identifies the current-user session list endpoint route fragment.
	AuthSessions = "/sessions"

	// AuthSessionRevoke identifies the current-user per-session revoke endpoint route fragment.
	AuthSessionRevoke = "/sessions/:sessionID/revoke"

	// AuthBootstrap identifies the bootstrap endpoint route fragment.
	AuthBootstrap = "/bootstrap"

	// AuthChangePassword identifies the current-user password change endpoint route fragment.
	AuthChangePassword = "/change-password"

	// AuthCompleteRequiredPasswordChange identifies the restricted-session password completion route fragment.
	AuthCompleteRequiredPasswordChange = "/complete-required-password-change"
)
