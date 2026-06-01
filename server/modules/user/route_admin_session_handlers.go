package user

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	useropenapi "graft/server/internal/contract/openapi/user"
	"graft/server/internal/httpx"
	"graft/server/internal/pluginapi"
	usercontract "graft/server/modules/user/contract"
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
		userReadGeneratedHandler{}.GetUserSessions(rawID, bindGeneratedUserSessionsParams(ginCtx, listOptions))

		sessions, err := r.authSvc.ListUserSessions(ginCtx.Request.Context(), summary.ID, listOptions)
		if err != nil {
			r.runtime().writeAuthRouteError(ginCtx, "list user refresh sessions failed", err, zap.Uint64("userID", rawID))
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, toGeneratedSessionSummaries(sessions))
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
		userWriteGeneratedHandler{}.PostUserSessionRevoke(rawID, sessionID, bindGeneratedUserSessionRevokeParams(ginCtx))

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
		userWriteGeneratedHandler{}.PostUserSessionsRevokeAll(rawID, bindGeneratedUserSessionsRevokeAllParams(ginCtx))

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

func (h userReadGeneratedHandler) GetUserSessions(id uint64, params useropenapi.GetUserSessionsParams) {
	_ = h
	_ = id
	_ = params
}

func (h userWriteGeneratedHandler) PostUserSessionsRevokeAll(
	id uint64,
	params useropenapi.PostUserSessionsRevokeAllParams,
) {
	_ = h
	_ = id
	_ = params
}

func (h userWriteGeneratedHandler) PostUserSessionRevoke(
	id uint64,
	sessionID string,
	params useropenapi.PostUserSessionRevokeParams,
) {
	_ = h
	_ = id
	_ = sessionID
	_ = params
}

func bindGeneratedUserSessionsParams(
	ginCtx *gin.Context,
	options sessionListOptions,
) useropenapi.GetUserSessionsParams {
	locale, requestID := bindGeneratedHeaders(ginCtx)
	params := useropenapi.GetUserSessionsParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}
	if options.Limit > 0 {
		params.Limit = &options.Limit
	}

	return params
}

func bindGeneratedUserSessionsRevokeAllParams(ginCtx *gin.Context) useropenapi.PostUserSessionsRevokeAllParams {
	locale, requestID := bindGeneratedHeaders(ginCtx)
	return useropenapi.PostUserSessionsRevokeAllParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}
}

func bindGeneratedUserSessionRevokeParams(ginCtx *gin.Context) useropenapi.PostUserSessionRevokeParams {
	locale, requestID := bindGeneratedHeaders(ginCtx)
	return useropenapi.PostUserSessionRevokeParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}
}
