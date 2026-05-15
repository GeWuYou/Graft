package httpx

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"graft/server/internal/container"
	"graft/server/internal/i18n"
	"graft/server/internal/pluginapi"
)

const bearerPrefix = "Bearer "

// RequirePermission 以真实请求鉴权上下文保护路由。
//
// 该中间件只负责从请求中提取访问令牌、解析当前主体并调用授权器，不直接
// 依赖任何具体插件实现。缺少登录态返回 401，认证成功但权限不足返回 403。
func RequirePermission(localizer *i18n.Service, resolver container.Resolver, code string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authService, err := resolveAuthService(resolver)
		if err != nil {
			AbortLocalizedError(ctx, localizer, http.StatusInternalServerError, "common.internal_error", nil)
			return
		}

		requestToken, ok := extractBearerToken(ctx.Request)
		if !ok {
			AbortLocalizedError(ctx, localizer, http.StatusUnauthorized, "auth.missing_actor", nil)
			return
		}

		claims, err := authService.ParseAccessToken(ctx.Request.Context(), requestToken)
		if err != nil {
			if errors.Is(err, pluginapi.ErrExpiredAccessToken) || errors.Is(err, pluginapi.ErrInvalidAccessToken) {
				AbortLocalizedError(ctx, localizer, http.StatusUnauthorized, "auth.missing_actor", nil)
				return
			}
			AbortLocalizedError(ctx, localizer, http.StatusInternalServerError, "common.internal_error", nil)
			return
		}

		requestAuth := pluginapi.RequestAuthContext{Claims: claims}
		requestCtx := pluginapi.WithRequestAuthContext(ctx.Request.Context(), requestAuth)
		user, err := authService.CurrentUser(requestCtx)
		if err != nil {
			if errors.Is(err, pluginapi.ErrUnauthenticated) || errors.Is(err, pluginapi.ErrInvalidAccessToken) {
				AbortLocalizedError(ctx, localizer, http.StatusUnauthorized, "auth.missing_actor", nil)
				return
			}
			AbortLocalizedError(ctx, localizer, http.StatusInternalServerError, "common.internal_error", nil)
			return
		}

		requestAuth.User = user
		requestCtx = pluginapi.WithRequestAuthContext(ctx.Request.Context(), requestAuth)

		if strings.TrimSpace(code) != "" {
			authorizer, err := resolveAuthorizer(resolver)
			if err != nil {
				AbortLocalizedError(ctx, localizer, http.StatusInternalServerError, "common.internal_error", nil)
				return
			}
			if err := authorizer.Authorize(requestCtx, requestAuth, code); err != nil {
				if errors.Is(err, pluginapi.ErrPermissionDenied) {
					AbortLocalizedError(ctx, localizer, http.StatusForbidden, "auth.missing_permission", map[string]any{
						"permission": code,
					})
					return
				}
				if errors.Is(err, pluginapi.ErrUnauthenticated) {
					AbortLocalizedError(ctx, localizer, http.StatusUnauthorized, "auth.missing_actor", nil)
					return
				}
				AbortLocalizedError(ctx, localizer, http.StatusInternalServerError, "common.internal_error", nil)
				return
			}
		}

		ctx.Request = ctx.Request.WithContext(requestCtx)
		ctx.Next()
	}
}

// resolveAuthService 解析认证中间件必需的稳定 AuthService 单例。
func resolveAuthService(resolver container.Resolver) (pluginapi.AuthService, error) {
	if resolver == nil {
		return nil, errors.New("resolver is required")
	}

	authAny, err := resolver.Resolve((*pluginapi.AuthService)(nil))
	if err != nil {
		return nil, err
	}

	authService, ok := authAny.(pluginapi.AuthService)
	if !ok {
		return nil, errors.New("resolved auth service has unexpected type")
	}

	return authService, nil
}

// resolveAuthorizer 仅在路由声明了权限码时解析稳定 Authorizer 单例。
func resolveAuthorizer(resolver container.Resolver) (pluginapi.Authorizer, error) {
	if resolver == nil {
		return nil, errors.New("resolver is required")
	}

	authorizerAny, err := resolver.Resolve((*pluginapi.Authorizer)(nil))
	if err != nil {
		return nil, err
	}

	authorizer, ok := authorizerAny.(pluginapi.Authorizer)
	if !ok {
		return nil, errors.New("resolved authorizer has unexpected type")
	}

	return authorizer, nil
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
