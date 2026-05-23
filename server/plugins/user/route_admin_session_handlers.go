package user

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"graft/server/internal/httpx"
	"graft/server/internal/pluginapi"
	usercontract "graft/server/plugins/user/contract"
)

func (r userRouteRegistrar) registerAdminSessionRoutes(group *gin.RouterGroup) {
	r.registerAdminSessionReadRoute(group)
	r.registerAdminSessionRevokeRoutes(group)
}

func (r userRouteRegistrar) registerAdminSessionReadRoute(group *gin.RouterGroup) {
	group.GET(usercontract.UserSessions, r.guards.userSessionRead, r.guards.restrictedSession, func(ginCtx *gin.Context) {
		rawID, ok := readUserIDParam(ginCtx, r.ctx.I18n)
		if !ok {
			return
		}

		summary, err := r.userSvc.GetUserByID(ginCtx.Request.Context(), rawID)
		if err != nil {
			r.runtime().writeUserLookupError(ginCtx, rawID, "get user by id before listing sessions failed", err)
			return
		}

		listOptions, err := parseSessionListOptions(ginCtx.Query("limit"))
		if err != nil {
			writeInvalidArgumentField(ginCtx, r.ctx.I18n, "limit")
			return
		}

		sessions, err := r.authSvc.ListUserSessions(ginCtx.Request.Context(), summary.ID, listOptions)
		if err != nil {
			r.runtime().writeAuthRouteError(ginCtx, "list user refresh sessions failed", err, zap.Uint64("userID", rawID))
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, sessions)
	})
}

func (r userRouteRegistrar) registerAdminSessionRevokeRoutes(group *gin.RouterGroup) {
	r.registerAdminRevokeSingleSessionRoute(group)
	r.registerAdminRevokeAllSessionsRoute(group)
}

func (r userRouteRegistrar) registerAdminRevokeSingleSessionRoute(group *gin.RouterGroup) {
	group.POST(usercontract.UserSessionByIDRevoke, r.guards.userSessionRevoke, r.guards.restrictedSession, func(ginCtx *gin.Context) {
		rawID, ok := readUserIDParam(ginCtx, r.ctx.I18n)
		if !ok {
			return
		}

		sessionID, ok := readSessionIDParam(ginCtx, r.ctx.I18n)
		if !ok {
			return
		}

		summary, err := r.userSvc.GetUserByID(ginCtx.Request.Context(), rawID)
		if err != nil {
			r.runtime().writeUserLookupError(ginCtx, rawID, "get user by id before revoking session failed", err)
			return
		}

		if err := r.authSvc.RevokeUserSession(ginCtx.Request.Context(), summary.ID, sessionID); err != nil {
			r.runtime().writeAuthRouteError(
				ginCtx,
				"admin revoke user refresh session failed",
				err,
				zap.Uint64("userID", rawID),
				zap.String("sessionID", sessionID),
			)
			return
		}

		clearRefreshCookieWhen(ginCtx, r.authSvc, func(claims *pluginapi.AccessTokenClaims) bool {
			return claims.UserID == rawID && claims.SessionID == sessionID
		})
		httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
	})
}

func (r userRouteRegistrar) registerAdminRevokeAllSessionsRoute(group *gin.RouterGroup) {
	group.POST(usercontract.UserSessionsRevokeAll, r.guards.userSessionRevoke, r.guards.restrictedSession, func(ginCtx *gin.Context) {
		rawID, ok := readUserIDParam(ginCtx, r.ctx.I18n)
		if !ok {
			return
		}

		if err := r.authSvc.RevokeAllUserSessions(ginCtx.Request.Context(), rawID); err != nil {
			r.runtime().writeAuthRouteError(ginCtx, "admin revoke user refresh sessions failed", err, zap.Uint64("userID", rawID))
			return
		}

		clearRefreshCookieWhen(ginCtx, r.authSvc, func(claims *pluginapi.AccessTokenClaims) bool {
			return claims.UserID == rawID
		})
		httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
	})
}
