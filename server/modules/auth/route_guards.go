package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/module"
	"graft/server/internal/pluginapi"
	authcontract "graft/server/modules/auth/contract"
)

func newRouteGuards(
	ctx *module.Context,
	authService pluginapi.AuthService,
	authFlow pluginapi.AuthFlowService,
	apiBasePath string,
) routeGuards {
	publisher := httpx.NewSecurityAuditPublisher(ctx.EventBus, ctx.Logger, moduleID)
	return routeGuards{
		authenticated:          httpx.RequirePermission(ctx.I18n, authService, nil, "", publisher),
		requiredPasswordChange: newRequiredPasswordChangeGuard(ctx.I18n, authFlow),
		restrictedSession:      newRestrictedSessionGuard(ctx.I18n, authFlow, apiBasePath),
	}
}

func newRequiredPasswordChangeGuard(localizer *i18n.Service, authFlow pluginapi.AuthFlowService) gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		restricted, ok := loadRestrictedPasswordChangeState(ginCtx, localizer, authFlow)
		if !ok {
			return
		}
		if !restricted {
			abortLocalizedContractError(ginCtx, localizer, http.StatusForbidden, messagecontract.AuthForbidden.String(), nil)
			return
		}

		ginCtx.Next()
	}
}

func newRestrictedSessionGuard(localizer *i18n.Service, authFlow pluginapi.AuthFlowService, apiBasePath string) gin.HandlerFunc {
	allowedPaths := []string{
		authcontract.JoinRoute(apiBasePath, authcontract.AuthBootstrap),
		authcontract.JoinRoute(apiBasePath, authcontract.AuthCompleteRequiredPasswordChange),
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

		abortLocalizedContractError(ginCtx, localizer, http.StatusForbidden, messagecontract.AuthForbidden.String(), nil)
	}
}

func loadRestrictedPasswordChangeState(
	ginCtx *gin.Context,
	localizer *i18n.Service,
	authFlow pluginapi.AuthFlowService,
) (bool, bool) {
	if authFlow == nil {
		abortLocalizedContractError(ginCtx, localizer, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
		return false, false
	}

	restricted, err := authFlow.IsRestrictedPasswordChangeSession(ginCtx.Request.Context())
	if err != nil {
		if errors.Is(err, pluginapi.ErrUnauthenticated) {
			abortLocalizedContractError(ginCtx, localizer, http.StatusUnauthorized, messagecontract.AuthTokenMissing.String(), nil)
			return false, false
		}

		abortLocalizedContractError(ginCtx, localizer, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
		return false, false
	}

	return restricted, true
}
