package authopenapi

// ServerInterface is the minimal generated handler contract for guarded auth/session migration.
type ServerInterface interface {
	PostAuthLogin(params PostAuthLoginParams, body PostAuthLoginJSONRequestBody)
	PostAuthRefresh(params PostAuthRefreshParams)
	PostAuthLogout(params PostAuthLogoutParams)
	GetAuthBootstrap(params GetAuthBootstrapParams)
	GetAuthSessions(params GetAuthSessionsParams)
	PostAuthSessionsRevokeAll(params PostAuthSessionsRevokeAllParams)
	PostAuthSessionsRevokeOthers(params PostAuthSessionsRevokeOthersParams)
	PostAuthSessionRevoke(params PostAuthSessionRevokeParams)
	PostAuthChangePassword(params PostAuthChangePasswordParams, body PostAuthChangePasswordJSONRequestBody)
	PostAuthCompleteRequiredPasswordChange(
		params PostAuthCompleteRequiredPasswordChangeParams,
		body PostAuthCompleteRequiredPasswordChangeJSONRequestBody,
	)
}
