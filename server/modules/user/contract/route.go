// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package contract

import authcontract "graft/server/modules/auth/contract"

// JoinRoute combines a route group path with a route fragment.
func JoinRoute(group, fragment string) string {
	return group + fragment
}

//nolint:gosec // Canonical route fragments are API contracts, not credentials.
const (
	// AuthGroup keeps the legacy compatibility alias for the canonical auth route group.
	AuthGroup = authcontract.AuthGroup

	// AuthLogin keeps the legacy compatibility alias for the canonical auth login route fragment.
	AuthLogin = authcontract.AuthLogin

	// AuthRefresh keeps the legacy compatibility alias for the canonical auth refresh route fragment.
	AuthRefresh = authcontract.AuthRefresh

	// AuthLogout keeps the legacy compatibility alias for the canonical auth logout route fragment.
	AuthLogout = authcontract.AuthLogout

	// AuthSessionsRevokeAll keeps the legacy compatibility alias for the canonical revoke-all route fragment.
	AuthSessionsRevokeAll = authcontract.AuthSessionsRevokeAll

	// AuthSessionsRevokeOthers keeps the legacy compatibility alias for the canonical revoke-others route fragment.
	AuthSessionsRevokeOthers = authcontract.AuthSessionsRevokeOthers

	// AuthSessions keeps the legacy compatibility alias for the canonical session-list route fragment.
	AuthSessions = authcontract.AuthSessions

	// AuthSessionRevoke keeps the legacy compatibility alias for the canonical per-session revoke route fragment.
	AuthSessionRevoke = authcontract.AuthSessionRevoke

	// AuthBootstrap keeps the legacy compatibility alias for the canonical bootstrap route fragment.
	AuthBootstrap = authcontract.AuthBootstrap

	// AuthChangePassword keeps the legacy compatibility alias for the canonical password-change route fragment.
	AuthChangePassword = authcontract.AuthChangePassword

	// AuthCompleteRequiredPasswordChange keeps the legacy compatibility alias for the canonical required-password-change route fragment.
	AuthCompleteRequiredPasswordChange = authcontract.AuthCompleteRequiredPasswordChange

	// UsersGroup identifies the user-management route group.
	UsersGroup = "/users"

	// UserCollection identifies the collection endpoint route fragment on the users group.
	UserCollection = ""

	// UserByID identifies the single-user lookup route fragment.
	UserByID = "/:id"

	// UserUpdateRoute identifies the single-user update route fragment.
	UserUpdateRoute = "/:id/update"

	// UserStatusRoute identifies the single-user status update route fragment.
	UserStatusRoute = "/:id/status"

	// UserResetPasswordRoute identifies the single-user password reset route fragment.
	UserResetPasswordRoute = "/:id/reset-password"

	// UserDeleteRoute identifies the single-user soft-delete route fragment.
	UserDeleteRoute = "/:id/delete"

	// UserSessions identifies the admin user-session list route fragment.
	UserSessions = "/:id/sessions"

	// UserSessionsRevokeAll identifies the admin user revoke-all route fragment.
	UserSessionsRevokeAll = "/:id/sessions/revoke-all"

	// UserSessionByIDRevoke identifies the admin user per-session revoke route fragment.
	UserSessionByIDRevoke = "/:id/sessions/:sessionID/revoke"
)
