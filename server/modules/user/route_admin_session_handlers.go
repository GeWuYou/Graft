package user

import (
	"graft/server/internal/contract/openapi/strictsupport"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"graft/server/internal/httpx"
	"graft/server/internal/moduleapi"
	usercontract "graft/server/modules/user/contract"
)

func (r userRouteRegistrar) registerAdminSessionRoutes(group *gin.RouterGroup) {
	r.registerAdminSessionReadRoute(group)
	r.registerAdminSessionRevokeRoutes(group)
}

func (r userRouteRegistrar) registerAdminSessionReadRoute(group *gin.RouterGroup) {
	group.GET(usercontract.UserSessions, r.guards.userSessionRead, r.guards.restrictedSession, strictsupport.Route(r.wrapper.GetUserSessions, r.handleUserSessions))
}

func (r userRouteRegistrar) handleUserSessions(ginCtx *gin.Context) {
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
	sessions, err := r.authSessions.ListSessionsByUserID(ginCtx.Request.Context(), summary.ID)
	if err != nil {
		r.runtime().writeAuthRouteError(ginCtx, "list user refresh sessions failed", err, zap.Uint64("userID", rawID))
		return
	}

	httpx.WriteSuccess(ginCtx, http.StatusOK, toGeneratedSessionSummariesFromCapability(sessions, listOptions))
}

func (r userRouteRegistrar) registerAdminSessionRevokeRoutes(group *gin.RouterGroup) {
	r.registerAdminRevokeSingleSessionRoute(group)
	r.registerAdminRevokeAllSessionsRoute(group)
}

func (r userRouteRegistrar) registerAdminRevokeSingleSessionRoute(group *gin.RouterGroup) {
	group.POST(usercontract.UserSessionByIDRevoke, r.guards.userSessionRevoke, r.guards.restrictedSession, strictsupport.Route(r.wrapper.PostUserSessionRevoke, r.handleUserSessionRevoke))
}

func (r userRouteRegistrar) handleUserSessionRevoke(ginCtx *gin.Context) {
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

	if _, err := r.authSessions.RevokeSessionByUserID(ginCtx.Request.Context(), summary.ID, sessionID); err != nil {
		r.runtime().writeAuthRouteError(
			ginCtx,
			"admin revoke user refresh session failed",
			err,
			zap.Uint64("userID", rawID),
			zap.String("sessionID", sessionID),
		)
		return
	}

	clearRefreshCookieWhen(ginCtx, r.cookies, func(claims *moduleapi.AccessTokenClaims) bool {
		return claims.UserID == rawID && claims.SessionID == sessionID
	})
	httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
}

func (r userRouteRegistrar) registerAdminRevokeAllSessionsRoute(group *gin.RouterGroup) {
	group.POST(usercontract.UserSessionsRevokeAll, r.guards.userSessionRevoke, r.guards.restrictedSession, strictsupport.Route(r.wrapper.PostUserSessionsRevokeAll, r.handleUserSessionsRevokeAll))
}

func (r userRouteRegistrar) handleUserSessionsRevokeAll(ginCtx *gin.Context) {
	rawID, ok := readUserIDParam(ginCtx, r.ctx.I18n)
	if !ok {
		return
	}
	if _, err := r.authSessions.RevokeSessionsByUserID(ginCtx.Request.Context(), rawID); err != nil {
		r.runtime().writeAuthRouteError(ginCtx, "admin revoke user refresh sessions failed", err, zap.Uint64("userID", rawID))
		return
	}

	clearRefreshCookieWhen(ginCtx, r.cookies, func(claims *moduleapi.AccessTokenClaims) bool {
		return claims.UserID == rawID
	})
	httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
}
