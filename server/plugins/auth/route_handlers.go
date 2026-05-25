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
	"graft/server/internal/pluginapi"
	authcontract "graft/server/plugins/auth/contract"
)

func (r authRouteRegistrar) registerLoginRoutes(authGroup *gin.RouterGroup) {
	authGroup.POST(authcontract.AuthLogin, func(ginCtx *gin.Context) {
		var request authopenapi.PostAuthLoginJSONRequestBody
		if err := ginCtx.ShouldBindJSON(&request); err != nil {
			writeInvalidArgumentField(ginCtx, r.ctx.I18n, "body")
			return
		}
		authGeneratedHandler{}.PostAuthLogin(bindGeneratedAuthLoginParams(ginCtx), request)
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

		r.cookies.WriteRefreshCookie(ginCtx, result.RefreshToken, result.RefreshExpiry)
		httpx.WriteSuccess(ginCtx, http.StatusOK, toLoginResponse(result))
	})
	authGroup.POST(authcontract.AuthRefresh, func(ginCtx *gin.Context) {
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

		r.cookies.WriteRefreshCookie(ginCtx, result.RefreshToken, result.RefreshExpiry)
		httpx.WriteSuccess(ginCtx, http.StatusOK, toLoginResponse(result))
	})
	authGroup.POST(authcontract.AuthLogout, func(ginCtx *gin.Context) {
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
	})
}

func (r authRouteRegistrar) registerCurrentUserSessionRoutes(authGroup *gin.RouterGroup) {
	authGroup.POST(authcontract.AuthSessionsRevokeAll, r.guards.authenticated, r.guards.restrictedSession, func(ginCtx *gin.Context) {
		if err := r.authFlow.RevokeAllCurrentUserSessions(ginCtx.Request.Context()); err != nil {
			r.runtime().writeAuthRouteError(ginCtx, "revoke all refresh sessions failed", err)
			return
		}

		r.cookies.ClearRefreshCookie(ginCtx)
		httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
	})
	authGroup.POST(authcontract.AuthSessionsRevokeOthers, r.guards.authenticated, r.guards.restrictedSession, func(ginCtx *gin.Context) {
		if err := r.authFlow.RevokeOtherCurrentUserSessions(ginCtx.Request.Context()); err != nil {
			r.runtime().writeAuthRouteError(ginCtx, "revoke other user refresh sessions failed", err)
			return
		}

		httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
	})
	authGroup.GET(authcontract.AuthSessions, r.guards.authenticated, r.guards.restrictedSession, func(ginCtx *gin.Context) {
		limit, err := parseSessionListLimit(ginCtx.Query("limit"))
		if err != nil {
			writeInvalidArgumentField(ginCtx, r.ctx.I18n, "limit")
			return
		}

		sessions, err := r.authFlow.ListCurrentUserSessions(ginCtx.Request.Context(), limit)
		if err != nil {
			r.runtime().writeAuthRouteError(ginCtx, "list current user refresh sessions failed", err)
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, toSessionSummaries(sessions))
	})
	authGroup.POST(authcontract.AuthSessionRevoke, r.guards.authenticated, r.guards.restrictedSession, func(ginCtx *gin.Context) {
		sessionID, ok := readSessionIDParam(ginCtx, r.ctx.I18n)
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
			func(claims *pluginapi.AccessTokenClaims) bool {
				return claims.SessionID == sessionID
			},
		)
	})
}

func (r authRouteRegistrar) registerBootstrapAndPasswordRoutes(authGroup *gin.RouterGroup) {
	r.registerBootstrapRoute(authGroup)
	r.registerChangePasswordRoute(authGroup)
	r.registerCompleteRequiredPasswordChangeRoute(authGroup)
}

func (r authRouteRegistrar) registerBootstrapRoute(authGroup *gin.RouterGroup) {
	authGroup.GET(authcontract.AuthBootstrap, r.guards.authenticated, r.guards.restrictedSession, func(ginCtx *gin.Context) {
		authGeneratedHandler{}.GetAuthBootstrap(bindGeneratedAuthBootstrapParams(ginCtx))

		payload, err := r.authFlow.ReadBootstrapPayload(ginCtx.Request.Context(), ginCtx.Request)
		if err != nil {
			r.runtime().writeAuthRouteError(ginCtx, "read bootstrap payload failed", err)
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, toBootstrapResponse(payload))
	})
}

func (r authRouteRegistrar) registerChangePasswordRoute(authGroup *gin.RouterGroup) {
	authGroup.POST(authcontract.AuthChangePassword, r.guards.authenticated, r.guards.restrictedSession, func(ginCtx *gin.Context) {
		var request changePasswordRequest
		if err := ginCtx.ShouldBindJSON(&request); err != nil {
			writeInvalidArgumentField(ginCtx, r.ctx.I18n, "body")
			return
		}
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
	})
}

func (r authRouteRegistrar) registerCompleteRequiredPasswordChangeRoute(authGroup *gin.RouterGroup) {
	authGroup.POST(authcontract.AuthCompleteRequiredPasswordChange, r.guards.authenticated, r.guards.requiredPasswordChange, func(ginCtx *gin.Context) {
		var request completeRequiredPasswordChangeRequest
		if err := ginCtx.ShouldBindJSON(&request); err != nil {
			writeInvalidArgumentField(ginCtx, r.ctx.I18n, "body")
			return
		}
		if strings.TrimSpace(request.NewPassword) == "" {
			writeInvalidArgumentField(ginCtx, r.ctx.I18n, "new_password")
			return
		}

		if err := r.authFlow.CompleteRequiredPasswordChange(ginCtx.Request.Context(), request.NewPassword); err != nil {
			r.runtime().writeAuthRouteError(ginCtx, "complete required password change failed", err)
			return
		}

		httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
	})
}

func handleSessionRevocation(
	ginCtx *gin.Context,
	revoke func(context.Context) error,
	writeRouteError func(error),
	cookies CookieManager,
	shouldClearCookie func(*pluginapi.AccessTokenClaims) bool,
) {
	if err := revoke(ginCtx.Request.Context()); err != nil {
		writeRouteError(err)
		return
	}

	clearRefreshCookieWhen(ginCtx, cookies, shouldClearCookie)
	httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
}

type authGeneratedHandler struct{}

func (h authGeneratedHandler) PostAuthLogin(
	params authopenapi.PostAuthLoginParams,
	body authopenapi.PostAuthLoginJSONRequestBody,
) {
	_ = h
	_ = params
	_ = body
}

func (h authGeneratedHandler) GetAuthBootstrap(params authopenapi.GetAuthBootstrapParams) {
	_ = h
	_ = params
}

func bindGeneratedAuthLoginParams(ginCtx *gin.Context) authopenapi.PostAuthLoginParams {
	locale, requestID := bindGeneratedAuthHeaders(ginCtx)
	return authopenapi.PostAuthLoginParams{
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
