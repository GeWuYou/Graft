package user

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/pluginapi"
	usercontract "graft/server/plugins/user/contract"
)

func newRouteGuards(localizer *i18n.Service, authSvc *authService, authorizer pluginapi.Authorizer) routeGuards {
	return routeGuards{
		authenticated:          httpx.RequirePermission(localizer, authSvc, nil, ""),
		requiredPasswordChange: newRequiredPasswordChangeGuard(localizer, authSvc),
		userRead:               httpx.RequirePermission(localizer, authSvc, authorizer, usercontract.UserReadPermission.String()),
		userCreate:             httpx.RequirePermission(localizer, authSvc, authorizer, usercontract.UserCreatePermission.String()),
		userUpdate:             httpx.RequirePermission(localizer, authSvc, authorizer, usercontract.UserUpdatePermission.String()),
		userDisable:            httpx.RequirePermission(localizer, authSvc, authorizer, usercontract.UserDisablePermission.String()),
		userSessionRead:        httpx.RequirePermission(localizer, authSvc, authorizer, usercontract.UserSessionReadPermission.String()),
		userSessionRevoke:      httpx.RequirePermission(localizer, authSvc, authorizer, usercontract.UserSessionRevokePermission.String()),
	}
}

func newRequiredPasswordChangeGuard(localizer *i18n.Service, authSvc *authService) gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		restricted, ok := loadRestrictedPasswordChangeState(ginCtx, localizer, authSvc)
		if !ok {
			return
		}
		if !restricted {
			abortLocalizedContractError(ginCtx, localizer, http.StatusForbidden, messagecontract.AuthForbidden, nil)
			return
		}

		ginCtx.Next()
	}
}

func newRestrictedSessionGuard(localizer *i18n.Service, authSvc *authService, apiBasePath string) gin.HandlerFunc {
	allowedPaths := []string{
		usercontract.JoinRoute(apiBasePath, usercontract.AuthBootstrap),
		usercontract.JoinRoute(apiBasePath, usercontract.AuthCompleteRequiredPasswordChange),
	}

	return func(ginCtx *gin.Context) {
		restricted, ok := loadRestrictedPasswordChangeState(ginCtx, localizer, authSvc)
		if !ok {
			return
		}
		if !restricted {
			ginCtx.Next()
			return
		}

		for _, allowedPath := range allowedPaths {
			if ginCtx.FullPath() == allowedPath {
				ginCtx.Next()
				return
			}
		}

		abortLocalizedContractError(ginCtx, localizer, http.StatusForbidden, messagecontract.AuthForbidden, nil)
	}
}

func loadRestrictedPasswordChangeState(
	ginCtx *gin.Context,
	localizer *i18n.Service,
	authSvc *authService,
) (bool, bool) {
	if authSvc == nil {
		abortLocalizedContractError(ginCtx, localizer, http.StatusInternalServerError, messagecontract.CommonInternalError, nil)
		return false, false
	}

	restricted, err := authSvc.isRestrictedPasswordChangeSession(ginCtx.Request.Context())
	if err != nil {
		if errors.Is(err, pluginapi.ErrUnauthenticated) {
			abortLocalizedContractError(ginCtx, localizer, http.StatusUnauthorized, messagecontract.AuthTokenMissing, nil)
			return false, false
		}

		abortLocalizedContractError(ginCtx, localizer, http.StatusInternalServerError, messagecontract.CommonInternalError, nil)
		return false, false
	}

	return restricted, true
}
