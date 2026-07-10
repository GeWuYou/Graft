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

// newRouteGuards 创建包含身份认证、密码修改限制及用户权限校验处理器的路由守卫集合。
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
	}
}

// newRequiredPasswordChangeGuard creates middleware that allows requests only for
// sessions restricted to the password-change flow.
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

// loadRestrictedPasswordChangeState determines whether the current会话受限于密码修改。
// 成功时返回限制状态和 true；认证失败或内部错误时终止请求并返回 false、false。
func? Wait instruction only docstring, no signature. Need omit func. Also Chinese sentence starts name. Could use "loadRestrictedPasswordChangeState 判断当前会话是否受到密码修改限制。" valid. Return docs maybe comments not tags Go. 
// loadRestrictedPasswordChangeState 判断当前会话是否受到密码修改限制。
// 成功时返回限制状态和 true。认证缺失或服务不可用时终止请求，并返回 false、false。
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
