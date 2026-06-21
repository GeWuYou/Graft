package user

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/moduleapi"
	usercontract "graft/server/modules/user/contract"
)

func newRouteGuards(
	localizer *i18n.Service,
	services registeredServices,
	authorizer moduleapi.Authorizer,
	publisher httpx.SecurityAuditPublisher,
) routeGuards {
	return routeGuards{
		authenticated:          httpx.RequirePermission(localizer, services.auth, nil, "", publisher),
		requiredPasswordChange: newRequiredPasswordChangeGuard(localizer, services.authFlow),
		userRead:               httpx.RequirePermission(localizer, services.auth, authorizer, usercontract.UserReadPermission.String(), publisher),
		userCreate:             httpx.RequirePermission(localizer, services.auth, authorizer, usercontract.UserCreatePermission.String(), publisher),
		userUpdate:             httpx.RequirePermission(localizer, services.auth, authorizer, usercontract.UserUpdatePermission.String(), publisher),
		userDisable:            httpx.RequirePermission(localizer, services.auth, authorizer, usercontract.UserDisablePermission.String(), publisher),
		userSessionRead:        httpx.RequirePermission(localizer, services.auth, authorizer, usercontract.UserSessionReadPermission.String(), publisher),
		userSessionRevoke:      httpx.RequirePermission(localizer, services.auth, authorizer, usercontract.UserSessionRevokePermission.String(), publisher),
		authRepo:               services.authRepo,
		passwords:              services.passwords,
		policy:                 services.policy,
	}
}

func newRequiredPasswordChangeGuard(localizer *i18n.Service, authFlow moduleapi.AuthFlowService) gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		restricted, ok := loadRestrictedPasswordChangeState(ginCtx, localizer, authFlow)
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

func newRestrictedSessionGuard(localizer *i18n.Service, authFlow moduleapi.AuthFlowService, apiBasePath string) gin.HandlerFunc {
	allowedPaths := []string{
		usercontract.JoinRoute(apiBasePath, usercontract.AuthBootstrap),
		usercontract.JoinRoute(apiBasePath, usercontract.AuthCompleteRequiredPasswordChange),
	}

	return func(ginCtx *gin.Context) {
		restricted, ok := loadRestrictedPasswordChangeState(ginCtx, localizer, authFlow)
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
	authFlow moduleapi.AuthFlowService,
) (bool, bool) {
	if authFlow == nil {
		abortLocalizedContractError(ginCtx, localizer, http.StatusInternalServerError, messagecontract.CommonInternalError, nil)
		return false, false
	}

	restricted, err := authFlow.IsRestrictedPasswordChangeSession(ginCtx.Request.Context())
	if err != nil {
		if errors.Is(err, moduleapi.ErrUnauthenticated) {
			abortLocalizedContractError(ginCtx, localizer, http.StatusUnauthorized, messagecontract.AuthTokenMissing, nil)
			return false, false
		}

		abortLocalizedContractError(ginCtx, localizer, http.StatusInternalServerError, messagecontract.CommonInternalError, nil)
		return false, false
	}

	return restricted, true
}
