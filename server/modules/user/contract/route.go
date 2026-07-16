package contract

// JoinRoute 拼接路由组路径和路由片段，统一保持 API 路径的斜杠边界。
func JoinRoute(group, fragment string) string {
	return group + fragment
}

//nolint:gosec // 路由片段属于 API 契约，不是凭据或外部输入。
const (
	// UserListMenuPath identifies the canonical user-management UI menu path.
	UserListMenuPath = "/security/users"

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
