package user

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/httpx"
	"graft/server/internal/pluginapi"
	usercontract "graft/server/plugins/user/contract"
)

func (r authRouteRegistrar) registerLoginRoutes(authGroup *gin.RouterGroup) {
	authGroup.POST(usercontract.AuthLogin, func(ginCtx *gin.Context) {
		var request loginRequest
		if err := ginCtx.ShouldBindJSON(&request); err != nil {
			writeInvalidArgumentField(ginCtx, r.ctx.I18n, "body")
			return
		}
		normalizedUsername := strings.TrimSpace(request.Username)
		if normalizedUsername == "" {
			writeInvalidArgumentField(ginCtx, r.ctx.I18n, "username")
			return
		}
		if request.Password == "" {
			writeInvalidArgumentField(ginCtx, r.ctx.I18n, "password")
			return
		}

		result, err := r.authSvc.LoginWithRefresh(ginCtx.Request.Context(), normalizedUsername, request.Password)
		if err != nil {
			r.runtime().writeAuthRouteError(ginCtx, "login failed", err)
			return
		}

		r.authSvc.cookies.writeRefreshCookie(ginCtx, result.RefreshToken, result.RefreshExpiry)
		httpx.WriteSuccess(ginCtx, http.StatusOK, loginResponse{
			AccessToken:        result.AccessToken,
			ExpiresAt:          result.AccessExpiry,
			MustChangePassword: result.MustChangePassword,
			User:               result.User,
		})
	})
	authGroup.POST(usercontract.AuthRefresh, func(ginCtx *gin.Context) {
		refreshToken, err := r.authSvc.cookies.readRefreshCookie(ginCtx)
		if err != nil {
			writeLocalizedContractError(ginCtx, r.ctx.I18n, http.StatusUnauthorized, messagecontract.AuthTokenMissing, nil)
			return
		}

		result, err := r.authSvc.RefreshWithRotation(ginCtx.Request.Context(), refreshToken)
		if err != nil {
			r.runtime().writeAuthRouteError(ginCtx, "refresh session failed", err)
			return
		}

		r.authSvc.cookies.writeRefreshCookie(ginCtx, result.RefreshToken, result.RefreshExpiry)
		httpx.WriteSuccess(ginCtx, http.StatusOK, loginResponse{
			AccessToken:        result.AccessToken,
			ExpiresAt:          result.AccessExpiry,
			MustChangePassword: result.MustChangePassword,
			User:               result.User,
		})
	})
	authGroup.POST(usercontract.AuthLogout, func(ginCtx *gin.Context) {
		refreshToken, err := r.authSvc.cookies.readRefreshCookie(ginCtx)
		if err != nil {
			writeLocalizedContractError(ginCtx, r.ctx.I18n, http.StatusUnauthorized, messagecontract.AuthTokenMissing, nil)
			return
		}

		if err := r.authSvc.LogoutCurrentSession(ginCtx.Request.Context(), refreshToken); err != nil {
			r.runtime().writeAuthRouteError(ginCtx, "logout session failed", err)
			return
		}

		r.authSvc.cookies.clearRefreshCookie(ginCtx)
		httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
	})
}

func (r authRouteRegistrar) registerCurrentUserSessionRoutes(authGroup *gin.RouterGroup) {
	authGroup.POST(usercontract.AuthSessionsRevokeAll, r.guards.authenticated, r.guards.restrictedSession, func(ginCtx *gin.Context) {
		if err := r.authSvc.RevokeAllCurrentUserSessions(ginCtx.Request.Context()); err != nil {
			r.runtime().writeAuthRouteError(ginCtx, "revoke all refresh sessions failed", err)
			return
		}

		r.authSvc.cookies.clearRefreshCookie(ginCtx)
		httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
	})
	authGroup.POST(usercontract.AuthSessionsRevokeOthers, r.guards.authenticated, r.guards.restrictedSession, func(ginCtx *gin.Context) {
		if err := r.authSvc.RevokeOtherCurrentUserSessions(ginCtx.Request.Context()); err != nil {
			r.runtime().writeAuthRouteError(ginCtx, "revoke other user refresh sessions failed", err)
			return
		}

		httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
	})
	authGroup.GET(usercontract.AuthSessions, r.guards.authenticated, r.guards.restrictedSession, func(ginCtx *gin.Context) {
		listOptions, err := parseSessionListOptions(ginCtx.Query("limit"))
		if err != nil {
			writeInvalidArgumentField(ginCtx, r.ctx.I18n, "limit")
			return
		}

		sessions, err := r.authSvc.ListCurrentUserSessions(ginCtx.Request.Context(), listOptions)
		if err != nil {
			r.runtime().writeAuthRouteError(ginCtx, "list current user refresh sessions failed", err)
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, sessions)
	})
	authGroup.POST(usercontract.AuthSessionRevoke, r.guards.authenticated, r.guards.restrictedSession, func(ginCtx *gin.Context) {
		sessionID, ok := readSessionIDParam(ginCtx, r.ctx.I18n)
		if !ok {
			return
		}

		handleSessionRevocation(
			ginCtx,
			func(ctx context.Context) error {
				return r.authSvc.RevokeCurrentUserSession(ctx, sessionID)
			},
			func(err error) {
				r.runtime().writeAuthRouteError(ginCtx, "revoke current user refresh session failed", err, zap.String("sessionID", sessionID))
			},
			r.authSvc,
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
	authGroup.GET(usercontract.AuthBootstrap, r.guards.authenticated, r.guards.restrictedSession, func(ginCtx *gin.Context) {
		payload, err := r.bootstrapSvc.Read(ginCtx.Request.Context(), ginCtx.Request)
		if err != nil {
			if errors.Is(err, pluginapi.ErrUnauthenticated) {
				writeLocalizedContractError(ginCtx, r.ctx.I18n, http.StatusUnauthorized, messagecontract.AuthTokenMissing, nil)
				return
			}

			r.runtime().logger.Error("read bootstrap payload failed",
				zap.String("plugin", r.pluginName),
				zap.Error(err),
			)
			writeLocalizedContractError(ginCtx, r.ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError, nil)
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, payload)
	})
}

func (r authRouteRegistrar) registerChangePasswordRoute(authGroup *gin.RouterGroup) {
	authGroup.POST(usercontract.AuthChangePassword, r.guards.authenticated, r.guards.restrictedSession, func(ginCtx *gin.Context) {
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

		if err := r.authSvc.ChangeCurrentUserPassword(ginCtx.Request.Context(), request.CurrentPassword, request.NewPassword); err != nil {
			r.runtime().writeAuthRouteError(ginCtx, "change current user password failed", err)
			return
		}

		httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
	})
}

func (r authRouteRegistrar) registerCompleteRequiredPasswordChangeRoute(authGroup *gin.RouterGroup) {
	authGroup.POST(usercontract.AuthCompleteRequiredPasswordChange, r.guards.authenticated, r.guards.requiredPasswordChange, func(ginCtx *gin.Context) {
		var request completeRequiredPasswordChangeRequest
		if err := ginCtx.ShouldBindJSON(&request); err != nil {
			writeInvalidArgumentField(ginCtx, r.ctx.I18n, "body")
			return
		}
		if strings.TrimSpace(request.NewPassword) == "" {
			writeInvalidArgumentField(ginCtx, r.ctx.I18n, "new_password")
			return
		}

		if err := r.authSvc.CompleteRequiredPasswordChange(ginCtx.Request.Context(), request.NewPassword); err != nil {
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
	authSvc *authService,
	shouldClearCookie func(*pluginapi.AccessTokenClaims) bool,
) {
	if err := revoke(ginCtx.Request.Context()); err != nil {
		writeRouteError(err)
		return
	}

	clearRefreshCookieWhen(ginCtx, authSvc, shouldClearCookie)
	httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
}
