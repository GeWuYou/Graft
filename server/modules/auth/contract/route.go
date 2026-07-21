package contract

// JoinRoute 将路由组路径与模块路由片段拼接为完整路径。
func JoinRoute(group, fragment string) string {
	return group + fragment
}

//nolint:gosec // canonical 路由片段属于 API 契约，不是凭据。
const (
	// AuthGroup 是 auth 模块的路由组路径。
	AuthGroup = "/auth"

	// AuthLogin 是登录接口的路由片段。
	AuthLogin = "/login"

	// AuthRefresh 是刷新会话接口的路由片段。
	AuthRefresh = "/refresh"

	// AuthLogout 是退出登录接口的路由片段。
	AuthLogout = "/logout"

	// AuthSessionsRevokeAll 是当前用户吊销全部会话接口的路由片段。
	AuthSessionsRevokeAll = "/sessions/revoke-all"

	// AuthSessionsRevokeOthers 是当前用户吊销其它会话接口的路由片段。
	AuthSessionsRevokeOthers = "/sessions/revoke-others"

	// AuthSessions 是当前用户会话列表接口的路由片段。
	AuthSessions = "/sessions"

	// AuthSessionRevoke 是当前用户吊销单个会话接口的路由片段。
	AuthSessionRevoke = "/sessions/:sessionID/revoke"

	// AuthBootstrap 是认证引导接口的路由片段。
	AuthBootstrap = "/bootstrap"

	// AuthChangePassword 是当前用户自助改密接口的路由片段。
	AuthChangePassword = "/change-password"

	// AuthCompleteRequiredPasswordChange 是受限会话完成强制改密接口的路由片段。
	AuthCompleteRequiredPasswordChange = "/complete-required-password-change"

	// AuthPersonalAccessTokens 是当前用户管理个人 API Token 的路由片段。
	AuthPersonalAccessTokens = "/personal-access-tokens"

	// AuthPersonalAccessTokenRevoke 是当前用户撤销一个个人 API Token 的路由片段。
	AuthPersonalAccessTokenRevoke = "/personal-access-tokens/:tokenID/revoke"
)
