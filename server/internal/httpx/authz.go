package httpx

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"graft/server/internal/i18n"
	"graft/server/internal/pluginapi"
)

const bearerPrefix = "Bearer "

// RequirePermission 以真实请求鉴权上下文保护路由。
//
// 该中间件只负责从请求中提取访问令牌、解析当前主体并调用授权器，不直接
// 依赖任何具体插件实现。缺少登录态返回 401，认证成功但权限不足返回 403。
func RequirePermission(
	localizer *i18n.Service,
	authService pluginapi.AuthService,
	authorizer pluginapi.Authorizer,
	code string,
) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		EnsureRequestID(ctx)

		if authService == nil {
			AbortLocalizedError(ctx, localizer, http.StatusInternalServerError, "common.internal_error", nil)
			return
		}

		requestAuth, requestCtx, handled := authenticateRequest(ctx, localizer, authService)
		if handled {
			return
		}
		if authorizeRequest(requestCtx, ctx, localizer, authorizer, code, requestAuth) {
			return
		}

		ctx.Request = ctx.Request.WithContext(requestCtx)
		ctx.Next()
	}
}

func authenticateRequest(
	ctx *gin.Context,
	localizer *i18n.Service,
	authService pluginapi.AuthService,
) (pluginapi.RequestAuthContext, context.Context, bool) {
	requestToken, ok := extractBearerToken(ctx.Request)
	if !ok {
		AbortLocalizedError(ctx, localizer, http.StatusUnauthorized, "auth.token_missing", nil)
		return pluginapi.RequestAuthContext{}, nil, true
	}

	claims, err := authService.ParseAccessToken(ctx.Request.Context(), requestToken)
	if err != nil {
		writeAccessTokenError(ctx, localizer, err)
		return pluginapi.RequestAuthContext{}, nil, true
	}

	requestAuth := pluginapi.RequestAuthContext{Claims: claims}
	requestCtx := pluginapi.WithRequestAuthContext(ctx.Request.Context(), requestAuth)
	user, err := authService.CurrentUser(requestCtx)
	if err != nil {
		writeCurrentUserError(ctx, localizer, err)
		return pluginapi.RequestAuthContext{}, nil, true
	}

	requestAuth.User = user
	requestCtx = pluginapi.WithRequestAuthContext(ctx.Request.Context(), requestAuth)
	return requestAuth, requestCtx, false
}

func authorizeRequest(
	requestCtx context.Context,
	ctx *gin.Context,
	localizer *i18n.Service,
	authorizer pluginapi.Authorizer,
	code string,
	requestAuth pluginapi.RequestAuthContext,
) bool {
	if strings.TrimSpace(code) == "" {
		return false
	}

	if authorizer == nil {
		AbortLocalizedError(ctx, localizer, http.StatusInternalServerError, "common.internal_error", nil)
		return true
	}
	if err := authorizer.Authorize(requestCtx, requestAuth, code); err != nil {
		writeAuthorizationError(ctx, localizer, code, err)
		return true
	}

	return false
}

func writeAccessTokenError(ctx *gin.Context, localizer *i18n.Service, err error) {
	switch {
	case errors.Is(err, pluginapi.ErrExpiredAccessToken):
		AbortLocalizedError(ctx, localizer, http.StatusUnauthorized, "auth.token_expired", nil)
	case errors.Is(err, pluginapi.ErrInvalidAccessToken):
		AbortLocalizedError(ctx, localizer, http.StatusUnauthorized, "auth.token_invalid", nil)
	default:
		AbortLocalizedError(ctx, localizer, http.StatusInternalServerError, "common.internal_error", nil)
	}
}

func writeCurrentUserError(ctx *gin.Context, localizer *i18n.Service, err error) {
	switch {
	case errors.Is(err, pluginapi.ErrInvalidAccessToken):
		AbortLocalizedError(ctx, localizer, http.StatusUnauthorized, "auth.token_invalid", nil)
	case errors.Is(err, pluginapi.ErrUnauthenticated):
		AbortLocalizedError(ctx, localizer, http.StatusUnauthorized, "auth.token_missing", nil)
	default:
		AbortLocalizedError(ctx, localizer, http.StatusInternalServerError, "common.internal_error", nil)
	}
}

func writeAuthorizationError(ctx *gin.Context, localizer *i18n.Service, code string, err error) {
	switch {
	case errors.Is(err, pluginapi.ErrPermissionDenied):
		AbortLocalizedError(ctx, localizer, http.StatusForbidden, "auth.forbidden", map[string]any{
			"permission": code,
		})
	case errors.Is(err, pluginapi.ErrInvalidAccessToken):
		AbortLocalizedError(ctx, localizer, http.StatusUnauthorized, "auth.token_invalid", nil)
	case errors.Is(err, pluginapi.ErrUnauthenticated):
		AbortLocalizedError(ctx, localizer, http.StatusUnauthorized, "auth.token_missing", nil)
	default:
		AbortLocalizedError(ctx, localizer, http.StatusInternalServerError, "common.internal_error", nil)
	}
}

func extractBearerToken(request *http.Request) (string, bool) {
	if request == nil {
		return "", false
	}

	header := strings.TrimSpace(request.Header.Get("Authorization"))
	if header == "" {
		return "", false
	}
	if !strings.HasPrefix(strings.ToLower(header), strings.ToLower(bearerPrefix)) {
		return "", false
	}

	token := strings.TrimSpace(header[len(bearerPrefix):])
	if token == "" {
		return "", false
	}

	return token, true
}
