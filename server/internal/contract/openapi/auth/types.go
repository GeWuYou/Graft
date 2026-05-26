package authopenapi

// ServerInterface is the minimal generated handler contract for guarded auth/session migration.
type ServerInterface interface {
	// PostAuthLogin 处理用户名密码登录请求，并对齐生成契约的登录入参边界。
	PostAuthLogin(params PostAuthLoginParams, body PostAuthLoginJSONRequestBody)
	// PostAuthRefresh 处理当前 refresh session 的续期入口。
	PostAuthRefresh(params PostAuthRefreshParams)
	// PostAuthLogout 处理当前会话登出并清理 refresh session。
	PostAuthLogout(params PostAuthLogoutParams)
	// GetAuthBootstrap 读取当前主体的 bootstrap 认证与导航上下文。
	GetAuthBootstrap(params GetAuthBootstrapParams)
	// GetAuthSessions 读取当前主体可见的活动会话列表。
	GetAuthSessions(params GetAuthSessionsParams)
	// PostAuthSessionsRevokeAll 撤销当前主体的全部会话。
	PostAuthSessionsRevokeAll(params PostAuthSessionsRevokeAllParams)
	// PostAuthSessionsRevokeOthers 撤销除当前会话外的其它会话。
	PostAuthSessionsRevokeOthers(params PostAuthSessionsRevokeOthersParams)
	// PostAuthSessionRevoke 撤销指定 session ID 对应的单个会话。
	PostAuthSessionRevoke(params PostAuthSessionRevokeParams)
	// PostAuthChangePassword 处理当前主体的改密请求。
	PostAuthChangePassword(params PostAuthChangePasswordParams, body PostAuthChangePasswordJSONRequestBody)
	// PostAuthCompleteRequiredPasswordChange 完成受限登录流中的强制改密步骤。
	PostAuthCompleteRequiredPasswordChange(
		params PostAuthCompleteRequiredPasswordChangeParams,
		body PostAuthCompleteRequiredPasswordChangeJSONRequestBody,
	)
}
