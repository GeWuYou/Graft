package contract

// JoinRoute 拼接路由组路径和路由片段，统一保持 API 路径的斜杠边界。
func JoinRoute(group, fragment string) string {
	return group + fragment
}

//nolint:gosec // 路由片段属于 API 契约，不是凭据或外部输入。
const (
	// UserListMenuPath 标识用户管理页面的规范菜单路径。
	UserListMenuPath = "/security/users"

	// UsersGroup 标识用户管理路由组。
	UsersGroup = "/users"

	// UserCollection 标识用户集合接口在用户路由组下的路径片段。
	UserCollection = ""

	// UserByID 标识单个用户查询接口的路径片段。
	UserByID = "/:id"

	// UserUpdateRoute 标识单个用户更新接口的路径片段。
	UserUpdateRoute = "/:id/update"

	// UserStatusRoute 标识单个用户状态更新接口的路径片段。
	UserStatusRoute = "/:id/status"

	// UserResetPasswordRoute 标识单个用户密码重置接口的路径片段。
	UserResetPasswordRoute = "/:id/reset-password"

	// UserDeleteRoute 标识单个用户软删除接口的路径片段。
	UserDeleteRoute = "/:id/delete"

	// UserSessions 标识管理员查看指定用户会话列表的路径片段。
	UserSessions = "/:id/sessions"

	// UserSessionsRevokeAll 标识管理员撤销指定用户全部会话的路径片段。
	UserSessionsRevokeAll = "/:id/sessions/revoke-all"

	// UserSessionByIDRevoke 标识管理员撤销指定用户单个会话的路径片段。
	UserSessionByIDRevoke = "/:id/sessions/:sessionID/revoke"
)
