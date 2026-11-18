package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	httpheader "graft/server/internal/contract/httpheader"
	messagecontract "graft/server/internal/contract/message"
	authopenapi "graft/server/internal/contract/openapi/auth"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/moduleapi"
	authcontract "graft/server/modules/auth/contract"
)

func (r authRouteRegistrar) registerLoginRoutes(authGroup *gin.RouterGroup) {
	authGroup.POST(authcontract.AuthLogin, r.wrapper.PostAuthLogin)
	authGroup.POST(authcontract.AuthRefresh, r.wrapper.PostAuthRefresh)
	authGroup.POST(authcontract.AuthLogout, r.wrapper.PostAuthLogout)
}

func (r authRouteRegistrar) handleLoginRoute(ginCtx *gin.Context, request authopenapi.PostAuthLoginJSONRequestBody) {
	normalizedUsername := strings.TrimSpace(request.Username)
	if normalizedUsername == "" {
		writeInvalidArgumentField(ginCtx, r.ctx.I18n, "username")
		return
	}
	if request.Password == "" {
		writeInvalidArgumentField(ginCtx, r.ctx.I18n, "password")
		return
	}

	result, err := r.authFlow.StartLogin(ginCtx.Request.Context(), normalizedUsername, request.Password)
	if err != nil {
		r.runtime().writeAuthRouteError(ginCtx, "login failed", err)
		return
	}

	payload, mapErr := toLoginResponse(result)
	if mapErr != nil {
		r.runtime().writeResponseMappingError(ginCtx, "map login response failed", mapErr, zap.Uint64("userID", result.User.ID))
		return
	}

	r.cookies.WriteRefreshCookie(ginCtx, result.RefreshToken, result.RefreshExpiry)
	httpx.WriteSuccess(ginCtx, http.StatusOK, payload)
}

func (r authRouteRegistrar) handleRefreshRoute(ginCtx *gin.Context) {
	refreshToken, err := r.cookies.ReadRefreshCookie(ginCtx)
	if err != nil {
		writeLocalizedContractError(ginCtx, r.ctx.I18n, http.StatusUnauthorized, messagecontract.AuthTokenMissing.String(), nil)
		return
	}

	result, err := r.authFlow.RefreshSession(ginCtx.Request.Context(), refreshToken)
	if err != nil {
		r.runtime().writeAuthRouteError(ginCtx, "refresh session failed", err)
		return
	}

	payload, mapErr := toLoginResponse(result)
	if mapErr != nil {
		r.runtime().writeResponseMappingError(ginCtx, "map refresh response failed", mapErr, zap.Uint64("userID", result.User.ID))
		return
	}

	r.cookies.WriteRefreshCookie(ginCtx, result.RefreshToken, result.RefreshExpiry)
	httpx.WriteSuccess(ginCtx, http.StatusOK, payload)
}

func (r authRouteRegistrar) handleLogoutRoute(ginCtx *gin.Context) {
	refreshToken, err := r.cookies.ReadRefreshCookie(ginCtx)
	if err != nil {
		writeLocalizedContractError(ginCtx, r.ctx.I18n, http.StatusUnauthorized, messagecontract.AuthTokenMissing.String(), nil)
		return
	}

	if err := r.authFlow.LogoutCurrentSession(ginCtx.Request.Context(), refreshToken); err != nil {
		r.runtime().writeAuthRouteError(ginCtx, "logout session failed", err)
		return
	}

	r.cookies.ClearRefreshCookie(ginCtx)
	httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
}

func (r authRouteRegistrar) registerCurrentUserSessionRoutes(authGroup *gin.RouterGroup) {
	authGroup.POST(authcontract.AuthSessionsRevokeAll, r.guards.authenticated, r.guards.restrictedSession, r.wrapper.PostAuthSessionsRevokeAll)
	authGroup.POST(authcontract.AuthSessionsRevokeOthers, r.guards.authenticated, r.guards.restrictedSession, r.wrapper.PostAuthSessionsRevokeOthers)
	authGroup.GET(authcontract.AuthSessions, r.guards.authenticated, r.guards.restrictedSession, r.wrapper.GetAuthSessions)
	authGroup.POST(authcontract.AuthSessionRevoke, r.guards.authenticated, r.guards.restrictedSession, r.wrapper.PostAuthSessionRevoke)
}

func (r authRouteRegistrar) handleSessionsRevokeAllRoute(ginCtx *gin.Context) {
	if err := r.authFlow.RevokeAllCurrentUserSessions(ginCtx.Request.Context()); err != nil {
		r.runtime().writeAuthRouteError(ginCtx, "revoke all refresh sessions failed", err)
		return
	}

	r.cookies.ClearRefreshCookie(ginCtx)
	httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
}

func (r authRouteRegistrar) handleSessionsRevokeOthersRoute(ginCtx *gin.Context) {
	if err := r.authFlow.RevokeOtherCurrentUserSessions(ginCtx.Request.Context()); err != nil {
		r.runtime().writeAuthRouteError(ginCtx, "revoke other user refresh sessions failed", err)
		return
	}

	r.cookies.ClearRefreshCookie(ginCtx)
	httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
}

func (r authRouteRegistrar) handleSessionsListRoute(ginCtx *gin.Context) {
	params, err := bindGeneratedAuthSessionsParams(ginCtx)
	if err != nil {
		writeInvalidArgumentField(ginCtx, r.ctx.I18n, "limit")
		return
	}
	sessions, err := r.authFlow.ListCurrentUserSessions(ginCtx.Request.Context(), generatedSessionListLimit(params))
	if err != nil {
		r.runtime().writeAuthRouteError(ginCtx, "list current user refresh sessions failed", err)
		return
	}

	httpx.WriteSuccess(ginCtx, http.StatusOK, toSessionSummaries(sessions))
}

func (r authRouteRegistrar) handleSessionRevokeRoute(ginCtx *gin.Context) {
	_, sessionID, ok := bindGeneratedAuthSessionRevokeParams(ginCtx, r.ctx.I18n)
	if !ok {
		return
	}
	handleSessionRevocation(
		ginCtx,
		func(ctx context.Context) error {
			return r.authFlow.RevokeCurrentUserSession(ctx, sessionID)
		},
		func(err error) {
			r.runtime().writeAuthRouteError(ginCtx, "revoke current user refresh session failed", err, zap.String("sessionID", sessionID))
		},
		r.cookies,
		func(claims *moduleapi.AccessTokenClaims) bool {
			return claims.SessionID == sessionID
		},
	)
}

func (r authRouteRegistrar) registerPersonalAccessTokenRoutes(authGroup *gin.RouterGroup) {
	authGroup.GET(authcontract.AuthPersonalAccessTokens, r.guards.authenticated, r.guards.restrictedSession, r.wrapper.GetAuthPersonalAccessTokens)
	authGroup.POST(authcontract.AuthPersonalAccessTokens, r.guards.authenticated, r.guards.restrictedSession, r.wrapper.PostAuthPersonalAccessTokens)
	authGroup.POST(authcontract.AuthPersonalAccessTokenRevoke, r.guards.authenticated, r.guards.restrictedSession, r.wrapper.PostAuthPersonalAccessTokenRevoke)
}

func (r authRouteRegistrar) handlePersonalAccessTokenListRoute(ginCtx *gin.Context) {
	items, err := r.personalTokens.ListCurrentUserPersonalAccessTokens(ginCtx.Request.Context(), 0)
	if err != nil {
		r.writePersonalAccessTokenRouteError(ginCtx, "list personal access tokens failed", err)
		return
	}

	response, mapErr := toPersonalAccessTokenSummaries(items)
	if mapErr != nil {
		r.runtime().writeResponseMappingError(ginCtx, "map personal access token list response failed", mapErr)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, response)
}

func (r authRouteRegistrar) handlePersonalAccessTokenCreateRoute(ginCtx *gin.Context, request authopenapi.PostAuthPersonalAccessTokensJSONRequestBody) {
	issued, err := r.personalTokens.CreateCurrentUserPersonalAccessToken(ginCtx.Request.Context(), moduleapi.PersonalAccessTokenCreateInput{
		Name:      request.Name,
		Scopes:    request.Scopes,
		ExpiresAt: request.ExpiresAt,
	})
	if err != nil {
		r.writePersonalAccessTokenRouteError(ginCtx, "create personal access token failed", err)
		return
	}

	response, mapErr := toPersonalAccessTokenIssued(issued)
	if mapErr != nil {
		r.runtime().writeResponseMappingError(ginCtx, "map personal access token issue response failed", mapErr)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusCreated, response)
}

func (r authRouteRegistrar) handlePersonalAccessTokenRevokeRoute(ginCtx *gin.Context) {
	_, tokenID, ok := bindGeneratedAuthPersonalAccessTokenRevokeParams(ginCtx, r.ctx.I18n)
	if !ok {
		return
	}
	if err := r.personalTokens.RevokeCurrentUserPersonalAccessToken(ginCtx.Request.Context(), tokenID); err != nil {
		r.writePersonalAccessTokenRouteError(ginCtx, "revoke personal access token failed", err)
		return
	}
	httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
}

func (r authRouteRegistrar) registerBootstrapAndPasswordRoutes(authGroup *gin.RouterGroup) {
	r.registerBootstrapRoute(authGroup)
	r.registerChangePasswordRoute(authGroup)
	r.registerCompleteRequiredPasswordChangeRoute(authGroup)
}

func (r authRouteRegistrar) registerBootstrapRoute(authGroup *gin.RouterGroup) {
	authGroup.GET(authcontract.AuthBootstrap, r.guards.authenticated, r.guards.restrictedSession, r.wrapper.GetAuthBootstrap)
}

func (r authRouteRegistrar) handleBootstrapRoute(ginCtx *gin.Context) {
	payload, err := r.authFlow.ReadBootstrapPayload(ginCtx.Request.Context(), ginCtx.Request)
	if err != nil {
		r.runtime().writeAuthRouteError(ginCtx, "read bootstrap payload failed", err)
		return
	}

	response, mapErr := toBootstrapResponse(payload)
	if mapErr != nil {
		r.runtime().writeResponseMappingError(ginCtx, "map bootstrap response failed", mapErr, zap.Uint64("userID", payload.User.ID))
		return
	}

	httpx.WriteSuccess(ginCtx, http.StatusOK, response)
}

func (r authRouteRegistrar) registerChangePasswordRoute(authGroup *gin.RouterGroup) {
	authGroup.POST(authcontract.AuthChangePassword, r.guards.authenticated, r.guards.restrictedSession, r.wrapper.PostAuthChangePassword)
}

func (r authRouteRegistrar) handleChangePasswordRoute(ginCtx *gin.Context, request authopenapi.PostAuthChangePasswordJSONRequestBody) {
	if strings.TrimSpace(request.CurrentPassword) == "" {
		writeInvalidArgumentField(ginCtx, r.ctx.I18n, "current_password")
		return
	}
	if strings.TrimSpace(request.NewPassword) == "" {
		writeInvalidArgumentField(ginCtx, r.ctx.I18n, "new_password")
		return
	}

	if err := r.authFlow.ChangeCurrentUserPassword(ginCtx.Request.Context(), request.CurrentPassword, request.NewPassword); err != nil {
		r.runtime().writeAuthRouteError(ginCtx, "change current user password failed", err)
		return
	}

	httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
}

func (r authRouteRegistrar) registerCompleteRequiredPasswordChangeRoute(authGroup *gin.RouterGroup) {
	authGroup.POST(authcontract.AuthCompleteRequiredPasswordChange, r.guards.authenticated, r.guards.requiredPasswordChange, r.wrapper.PostAuthCompleteRequiredPasswordChange)
}

func (r authRouteRegistrar) handleCompleteRequiredPasswordChangeRoute(ginCtx *gin.Context, request authopenapi.PostAuthCompleteRequiredPasswordChangeJSONRequestBody) {
	if strings.TrimSpace(request.NewPassword) == "" {
		writeInvalidArgumentField(ginCtx, r.ctx.I18n, "new_password")
		return
	}

	if err := r.authFlow.CompleteRequiredPasswordChange(ginCtx.Request.Context(), request.NewPassword); err != nil {
		r.runtime().writeAuthRouteError(ginCtx, "complete required password change failed", err)
		return
	}

	httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
}

func handleSessionRevocation(
	ginCtx *gin.Context,
	revoke func(context.Context) error,
	writeRouteError func(error),
	cookies CookieManager,
	shouldClearCookie func(*moduleapi.AccessTokenClaims) bool,
) {
	if err := revoke(ginCtx.Request.Context()); err != nil {
		writeRouteError(err)
		return
	}

	clearRefreshCookieWhen(ginCtx, cookies, shouldClearCookie)
	httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
}

func bindGeneratedAuthLoginParams(ginCtx *gin.Context) authopenapi.PostAuthLoginParams {
	locale, requestID := bindGeneratedAuthHeaders(ginCtx)
	return authopenapi.PostAuthLoginParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}
}

func bindGeneratedAuthChangePasswordParams(ginCtx *gin.Context) authopenapi.PostAuthChangePasswordParams {
	locale, requestID := bindGeneratedAuthHeaders(ginCtx)
	return authopenapi.PostAuthChangePasswordParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}
}

func bindGeneratedAuthCompleteRequiredPasswordChangeParams(
	ginCtx *gin.Context,
) authopenapi.PostAuthCompleteRequiredPasswordChangeParams {
	locale, requestID := bindGeneratedAuthHeaders(ginCtx)
	return authopenapi.PostAuthCompleteRequiredPasswordChangeParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}
}

func bindGeneratedAuthBootstrapParams(ginCtx *gin.Context) authopenapi.GetAuthBootstrapParams {
	locale, requestID := bindGeneratedAuthHeaders(ginCtx)
	return authopenapi.GetAuthBootstrapParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}
}

func bindGeneratedAuthRefreshParams(ginCtx *gin.Context) authopenapi.PostAuthRefreshParams {
	locale, requestID := bindGeneratedAuthHeaders(ginCtx)
	return authopenapi.PostAuthRefreshParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}
}

func bindGeneratedAuthLogoutParams(ginCtx *gin.Context) authopenapi.PostAuthLogoutParams {
	locale, requestID := bindGeneratedAuthHeaders(ginCtx)
	return authopenapi.PostAuthLogoutParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}
}

func bindGeneratedAuthPersonalAccessTokenListParams(ginCtx *gin.Context) authopenapi.GetAuthPersonalAccessTokensParams {
	locale, requestID := bindGeneratedAuthHeaders(ginCtx)
	return authopenapi.GetAuthPersonalAccessTokensParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}
}

func bindGeneratedAuthPersonalAccessTokenCreateParams(ginCtx *gin.Context) authopenapi.PostAuthPersonalAccessTokensParams {
	locale, requestID := bindGeneratedAuthHeaders(ginCtx)
	return authopenapi.PostAuthPersonalAccessTokensParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}
}

func bindGeneratedAuthPersonalAccessTokenRevokeParams(
	ginCtx *gin.Context,
	localizer *i18n.Service,
) (authopenapi.PostAuthPersonalAccessTokenRevokeParams, uint64, bool) {
	locale, requestID := bindGeneratedAuthHeaders(ginCtx)
	tokenID, ok := readPersonalAccessTokenIDParam(ginCtx, localizer)
	if !ok {
		return authopenapi.PostAuthPersonalAccessTokenRevokeParams{}, 0, false
	}

	return authopenapi.PostAuthPersonalAccessTokenRevokeParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}, tokenID, true
}

func bindGeneratedAuthSessionsParams(ginCtx *gin.Context) (authopenapi.GetAuthSessionsParams, error) {
	locale, requestID := bindGeneratedAuthHeaders(ginCtx)
	limit, err := parseSessionListLimit(ginCtx.Query("limit"))
	if err != nil {
		return authopenapi.GetAuthSessionsParams{}, err
	}

	params := authopenapi.GetAuthSessionsParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}
	if limit > 0 {
		params.Limit = &limit
	}

	return params, nil
}

func bindGeneratedAuthSessionsRevokeAllParams(ginCtx *gin.Context) authopenapi.PostAuthSessionsRevokeAllParams {
	locale, requestID := bindGeneratedAuthHeaders(ginCtx)
	return authopenapi.PostAuthSessionsRevokeAllParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}
}

func bindGeneratedAuthSessionsRevokeOthersParams(ginCtx *gin.Context) authopenapi.PostAuthSessionsRevokeOthersParams {
	locale, requestID := bindGeneratedAuthHeaders(ginCtx)
	return authopenapi.PostAuthSessionsRevokeOthersParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}
}

func bindGeneratedAuthSessionRevokeParams(
	ginCtx *gin.Context,
	localizer *i18n.Service,
) (authopenapi.PostAuthSessionRevokeParams, string, bool) {
	locale, requestID := bindGeneratedAuthHeaders(ginCtx)
	sessionID, ok := readSessionIDParam(ginCtx, localizer)
	if !ok {
		return authopenapi.PostAuthSessionRevokeParams{}, "", false
	}

	return authopenapi.PostAuthSessionRevokeParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}, sessionID, true
}

func generatedSessionListLimit(params authopenapi.GetAuthSessionsParams) int {
	if params.Limit == nil {
		return 0
	}
	return *params.Limit
}

func bindGeneratedAuthHeaders(ginCtx *gin.Context) (*string, *string) {
	locale := authHeaderPointer(ginCtx.GetHeader(string(httpheader.Locale)))
	requestID := authHeaderPointer(ginCtx.GetHeader(httpx.RequestIDHeader))
	return locale, requestID
}

func authHeaderPointer(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}
