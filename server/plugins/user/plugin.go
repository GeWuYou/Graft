// Package user 提供接入 MVP 运行时的首个示例业务插件。
package user

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"graft/server/internal/container"
	"graft/server/internal/httpx"
	"graft/server/internal/menu"
	"graft/server/internal/permission"
	"graft/server/internal/plugin"
	"graft/server/internal/pluginapi"
	"graft/server/internal/store"
)

// Plugin 是用于验证扩展路径的示例用户能力插件。
//
// 该插件展示业务能力如何在 Register 阶段声明边界，在 Boot/Shutdown 阶段保持显式生命周期。
type Plugin struct{}

// NewPlugin 创建示例用户插件。
func NewPlugin() *Plugin {
	return &Plugin{}
}

// Name 返回插件的稳定标识。
func (p *Plugin) Name() string {
	return "user"
}

// Version 返回当前示例插件版本。
func (p *Plugin) Version() string {
	return "0.1.0"
}

// DependsOn 返回当前插件的依赖列表。
func (p *Plugin) DependsOn() []string {
	return nil
}

// Register 声明用户插件需要的权限、菜单、路由和公开服务。
//
// 约束：
//   - 只注册跨插件可见的稳定接口，不暴露具体仓储或 ORM 实现。
//   - 只做声明式装配，不启动后台 goroutine 或持久占用额外资源。
//
// 失败语义：
//   - 任一注册步骤失败都会中止插件装配，并由上层运行时负责整体回滚。
func (p *Plugin) Register(ctx *plugin.Context) error {
	ctx.PermissionRegistry.Register(permission.Item{
		Code:        "user.read",
		Name:        "Read Users",
		Description: "Allows reading user management data.",
		Plugin:      p.Name(),
	})
	ctx.PermissionRegistry.Register(permission.Item{
		Code:        "user.session.revoke",
		Name:        "Revoke User Sessions",
		Description: "Allows revoking refresh sessions for a specified user.",
		Plugin:      p.Name(),
	})
	ctx.PermissionRegistry.Register(permission.Item{
		Code:        "user.session.read",
		Name:        "Read User Sessions",
		Description: "Allows reading active refresh sessions for a specified user.",
		Plugin:      p.Name(),
	})

	ctx.MenuRegistry.Register(menu.Item{
		Code:       "user.list",
		Title:      "用户管理",
		Path:       "/users",
		Icon:       "usergroup",
		Permission: "user.read",
		Plugin:     p.Name(),
	})

	userSvc := userService{users: ctx.Stores.Users()}
	if err := ctx.Services.RegisterSingleton((*pluginapi.UserService)(nil), func(resolver container.Resolver) (any, error) {
		return userSvc, nil
	}); err != nil {
		return err
	}

	authSvc, err := newAuthService(ctx.Config.Auth, ctx.Stores.Auth(), ctx.Stores.Users())
	if err != nil {
		return err
	}

	if err := ctx.Services.RegisterSingleton((*pluginapi.AuthService)(nil), func(resolver container.Resolver) (any, error) {
		return authSvc, nil
	}); err != nil {
		return err
	}

	// 登录与 refresh 入口保持在插件内，避免把 session/cookie 细节泄漏到 core。
	authGroup := ctx.Router.Group("/auth")
	authGroup.POST("/login", func(ginCtx *gin.Context) {
		var request loginRequest
		if err := ginCtx.ShouldBindJSON(&request); err != nil {
			httpx.WriteLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, "common.invalid_argument", map[string]any{
				"field": "body",
			})
			return
		}

		if strings.TrimSpace(request.Username) == "" {
			httpx.WriteLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, "common.invalid_argument", map[string]any{
				"field": "username",
			})
			return
		}
		if request.Password == "" {
			httpx.WriteLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, "common.invalid_argument", map[string]any{
				"field": "password",
			})
			return
		}

		result, err := authSvc.LoginWithRefresh(ginCtx.Request.Context(), request.Username, request.Password)
		if err != nil {
			status, messageKey := mapAuthError(err)
			if status == http.StatusInternalServerError {
				ctx.Logger.Error("login failed",
					zap.String("plugin", p.Name()),
					zap.String("username", request.Username),
					zap.Error(err),
				)
			}

			httpx.WriteLocalizedError(ginCtx, ctx.I18n, status, messageKey, nil)
			return
		}

		authSvc.cookies.writeRefreshCookie(ginCtx, result.RefreshToken, result.RefreshExpiry)
		ginCtx.JSON(http.StatusOK, loginResponse{
			AccessToken: result.AccessToken,
			ExpiresAt:   result.AccessExpiry,
			User:        result.User,
		})
	})
	authGroup.POST("/refresh", func(ginCtx *gin.Context) {
		refreshToken, err := authSvc.cookies.readRefreshCookie(ginCtx)
		if err != nil {
			httpx.WriteLocalizedError(ginCtx, ctx.I18n, http.StatusUnauthorized, "auth.invalid_refresh_session", nil)
			return
		}

		result, err := authSvc.RefreshWithRotation(ginCtx.Request.Context(), refreshToken)
		if err != nil {
			status, messageKey := mapAuthError(err)
			if status == http.StatusInternalServerError {
				ctx.Logger.Error("refresh session failed",
					zap.String("plugin", p.Name()),
					zap.Error(err),
				)
			}

			httpx.WriteLocalizedError(ginCtx, ctx.I18n, status, messageKey, nil)
			return
		}

		authSvc.cookies.writeRefreshCookie(ginCtx, result.RefreshToken, result.RefreshExpiry)
		ginCtx.JSON(http.StatusOK, loginResponse{
			AccessToken: result.AccessToken,
			ExpiresAt:   result.AccessExpiry,
			User:        result.User,
		})
	})
	authGroup.POST("/logout", func(ginCtx *gin.Context) {
		refreshToken, err := authSvc.cookies.readRefreshCookie(ginCtx)
		if err != nil {
			httpx.WriteLocalizedError(ginCtx, ctx.I18n, http.StatusUnauthorized, "auth.invalid_refresh_session", nil)
			return
		}

		if err := authSvc.LogoutCurrentSession(ginCtx.Request.Context(), refreshToken); err != nil {
			status, messageKey := mapAuthError(err)
			if status == http.StatusInternalServerError {
				ctx.Logger.Error("logout session failed",
					zap.String("plugin", p.Name()),
					zap.Error(err),
				)
			}

			httpx.WriteLocalizedError(ginCtx, ctx.I18n, status, messageKey, nil)
			return
		}

		authSvc.cookies.clearRefreshCookie(ginCtx)
		ginCtx.Status(http.StatusNoContent)
	})
	// 当前用户自助撤销入口复用同一套 request-auth 上下文，只吊销当前主体名下
	// 的全部 refresh sessions，不把更宽的管理员治理语义下沉到 core。
	authGroup.POST("/sessions/revoke-all", httpx.RequirePermission(ctx.I18n, ctx.Services, ""), func(ginCtx *gin.Context) {
		if err := authSvc.RevokeAllCurrentUserSessions(ginCtx.Request.Context()); err != nil {
			status, messageKey := mapAuthError(err)
			if status == http.StatusInternalServerError {
				ctx.Logger.Error("revoke all refresh sessions failed",
					zap.String("plugin", p.Name()),
					zap.Error(err),
				)
			}

			httpx.WriteLocalizedError(ginCtx, ctx.I18n, status, messageKey, nil)
			return
		}

		authSvc.cookies.clearRefreshCookie(ginCtx)
		ginCtx.Status(http.StatusNoContent)
	})
	// 当前用户可批量吊销除当前请求外的其它有效 session，适合保留当前登录态的
	// 多端清退场景；该治理动作继续留在 user 插件内，不扩散新的 core 契约。
	authGroup.POST("/sessions/revoke-others", httpx.RequirePermission(ctx.I18n, ctx.Services, ""), func(ginCtx *gin.Context) {
		if err := authSvc.RevokeOtherCurrentUserSessions(ginCtx.Request.Context()); err != nil {
			status, messageKey := mapAuthError(err)
			if status == http.StatusInternalServerError {
				ctx.Logger.Error("revoke other user refresh sessions failed",
					zap.String("plugin", p.Name()),
					zap.Error(err),
				)
			}

			httpx.WriteLocalizedError(ginCtx, ctx.I18n, status, messageKey, nil)
			return
		}

		ginCtx.Status(http.StatusNoContent)
	})
	// 当前用户会话列表只暴露最小有效 session 摘要，避免把历史轮换或底层存储
	// 细节泄漏到插件外部，同时为后续更细粒度治理保留清晰入口。
	authGroup.GET("/sessions", httpx.RequirePermission(ctx.I18n, ctx.Services, ""), func(ginCtx *gin.Context) {
		listOptions, err := parseSessionListOptions(ginCtx.Query("limit"))
		if err != nil {
			httpx.WriteLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, "common.invalid_argument", map[string]any{
				"field": "limit",
			})
			return
		}

		sessions, err := authSvc.ListCurrentUserSessions(ginCtx.Request.Context(), listOptions)
		if err != nil {
			status, messageKey := mapAuthError(err)
			if status == http.StatusInternalServerError {
				ctx.Logger.Error("list current user refresh sessions failed",
					zap.String("plugin", p.Name()),
					zap.Error(err),
				)
			}

			httpx.WriteLocalizedError(ginCtx, ctx.I18n, status, messageKey, nil)
			return
		}

		ginCtx.JSON(http.StatusOK, sessions)
	})
	// 当前用户可对自己的一条有效 session 做定向吊销，保持会话治理仍然落在
	// user 插件边界内，而不是把单条操作拆进 core 中间件或公共 auth 服务。
	authGroup.POST("/sessions/:sessionID/revoke", httpx.RequirePermission(ctx.I18n, ctx.Services, ""), func(ginCtx *gin.Context) {
		sessionID := strings.TrimSpace(ginCtx.Param("sessionID"))
		if sessionID == "" {
			httpx.WriteLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, "common.invalid_argument", map[string]any{
				"field": "sessionID",
			})
			return
		}

		if err := authSvc.RevokeCurrentUserSession(ginCtx.Request.Context(), sessionID); err != nil {
			status, messageKey := mapAuthError(err)
			if status == http.StatusInternalServerError {
				ctx.Logger.Error("revoke current user refresh session failed",
					zap.String("plugin", p.Name()),
					zap.String("sessionID", sessionID),
					zap.Error(err),
				)
			}

			httpx.WriteLocalizedError(ginCtx, ctx.I18n, status, messageKey, nil)
			return
		}

		if requestAuth, ok := pluginapi.RequestAuthContextFromContext(ginCtx.Request.Context()); ok &&
			requestAuth.Claims != nil &&
			requestAuth.Claims.SessionID == sessionID {
			authSvc.cookies.clearRefreshCookie(ginCtx)
		}

		ginCtx.Status(http.StatusNoContent)
	})

	group := ctx.Router.Group("/users")
	group.GET("/:id", httpx.RequirePermission(ctx.I18n, ctx.Services, "user.read"), func(ginCtx *gin.Context) {
		rawID, err := parseUserID(ginCtx.Param("id"))
		if err != nil {
			httpx.WriteLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, "common.invalid_argument", map[string]any{
				"field": "id",
			})
			return
		}

		svcAny, err := ctx.Services.Resolve((*pluginapi.UserService)(nil))
		if err != nil {
			ctx.Logger.Error("resolve user service failed",
				zap.String("plugin", p.Name()),
				zap.Error(err),
			)
			httpx.WriteLocalizedError(ginCtx, ctx.I18n, http.StatusInternalServerError, "common.internal_error", nil)
			return
		}

		// 这里解析跨插件公共接口而不是直接依赖具体实现，保证后续用户插件
		// 内部存储实现变更时，不会破坏其它插件的依赖边界。
		svc := svcAny.(pluginapi.UserService)
		summary, err := svc.GetUserByID(ginCtx.Request.Context(), rawID)
		if err != nil {
			status := http.StatusInternalServerError
			messageKey := "common.internal_error"
			if errors.Is(err, store.ErrUserNotFound) {
				status = http.StatusNotFound
				messageKey = "user.not_found"
			} else {
				ctx.Logger.Error("get user by id failed",
					zap.String("plugin", p.Name()),
					zap.Uint64("userID", rawID),
					zap.Error(err),
				)
			}
			httpx.WriteLocalizedError(ginCtx, ctx.I18n, status, messageKey, nil)
			return
		}

		ginCtx.JSON(http.StatusOK, summary)
	})
	// 管理员查看指定用户当前有效 session 时仍留在 user 插件边界内，使用显式
	// session 读取权限，避免把更敏感的登录态治理语义隐式并入普通 user.read。
	group.GET("/:id/sessions", httpx.RequirePermission(ctx.I18n, ctx.Services, "user.session.read"), func(ginCtx *gin.Context) {
		rawID, err := parseUserID(ginCtx.Param("id"))
		if err != nil {
			httpx.WriteLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, "common.invalid_argument", map[string]any{
				"field": "id",
			})
			return
		}

		summary, err := userSvc.GetUserByID(ginCtx.Request.Context(), rawID)
		if err != nil {
			status := http.StatusInternalServerError
			messageKey := "common.internal_error"
			if errors.Is(err, store.ErrUserNotFound) {
				status = http.StatusNotFound
				messageKey = "user.not_found"
			} else {
				ctx.Logger.Error("get user by id before listing sessions failed",
					zap.String("plugin", p.Name()),
					zap.Uint64("userID", rawID),
					zap.Error(err),
				)
			}

			httpx.WriteLocalizedError(ginCtx, ctx.I18n, status, messageKey, nil)
			return
		}

		listOptions, err := parseSessionListOptions(ginCtx.Query("limit"))
		if err != nil {
			httpx.WriteLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, "common.invalid_argument", map[string]any{
				"field": "limit",
			})
			return
		}

		sessions, err := authSvc.ListUserSessions(ginCtx.Request.Context(), summary.ID, listOptions)
		if err != nil {
			status, messageKey := mapAuthError(err)
			if status == http.StatusInternalServerError {
				ctx.Logger.Error("list user refresh sessions failed",
					zap.String("plugin", p.Name()),
					zap.Uint64("userID", rawID),
					zap.Error(err),
				)
			}

			httpx.WriteLocalizedError(ginCtx, ctx.I18n, status, messageKey, nil)
			return
		}

		ginCtx.JSON(http.StatusOK, sessions)
	})
	// 管理员按用户与 session 双重显式标识做定向吊销，避免单独暴露 sessionID
	// 时跨用户误操作，同时保持权限与业务边界都停留在 user 插件内部。
	group.POST("/:id/sessions/:sessionID/revoke", httpx.RequirePermission(ctx.I18n, ctx.Services, "user.session.revoke"), func(ginCtx *gin.Context) {
		rawID, err := parseUserID(ginCtx.Param("id"))
		if err != nil {
			httpx.WriteLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, "common.invalid_argument", map[string]any{
				"field": "id",
			})
			return
		}

		sessionID := strings.TrimSpace(ginCtx.Param("sessionID"))
		if sessionID == "" {
			httpx.WriteLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, "common.invalid_argument", map[string]any{
				"field": "sessionID",
			})
			return
		}

		summary, err := userSvc.GetUserByID(ginCtx.Request.Context(), rawID)
		if err != nil {
			status := http.StatusInternalServerError
			messageKey := "common.internal_error"
			if errors.Is(err, store.ErrUserNotFound) {
				status = http.StatusNotFound
				messageKey = "user.not_found"
			} else {
				ctx.Logger.Error("get user by id before revoking session failed",
					zap.String("plugin", p.Name()),
					zap.Uint64("userID", rawID),
					zap.Error(err),
				)
			}

			httpx.WriteLocalizedError(ginCtx, ctx.I18n, status, messageKey, nil)
			return
		}

		if err := authSvc.RevokeUserSession(ginCtx.Request.Context(), summary.ID, sessionID); err != nil {
			status, messageKey := mapAuthError(err)
			if status == http.StatusInternalServerError {
				ctx.Logger.Error("admin revoke user refresh session failed",
					zap.String("plugin", p.Name()),
					zap.Uint64("userID", rawID),
					zap.String("sessionID", sessionID),
					zap.Error(err),
				)
			}

			httpx.WriteLocalizedError(ginCtx, ctx.I18n, status, messageKey, nil)
			return
		}

		if requestAuth, ok := pluginapi.RequestAuthContextFromContext(ginCtx.Request.Context()); ok &&
			requestAuth.Claims != nil &&
			requestAuth.Claims.UserID == rawID &&
			requestAuth.Claims.SessionID == sessionID {
			authSvc.cookies.clearRefreshCookie(ginCtx)
		}

		ginCtx.Status(http.StatusNoContent)
	})
	// 管理员按用户 ID 批量吊销 refresh sessions 仍保持在 user 插件边界内，
	// 通过专用权限码和显式路由声明治理入口，不把该语义扩散到 core。
	group.POST("/:id/sessions/revoke-all", httpx.RequirePermission(ctx.I18n, ctx.Services, "user.session.revoke"), func(ginCtx *gin.Context) {
		rawID, err := parseUserID(ginCtx.Param("id"))
		if err != nil {
			httpx.WriteLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, "common.invalid_argument", map[string]any{
				"field": "id",
			})
			return
		}

		if err := authSvc.RevokeAllUserSessions(ginCtx.Request.Context(), rawID); err != nil {
			status, messageKey := mapAuthError(err)
			if status == http.StatusInternalServerError {
				ctx.Logger.Error("admin revoke user refresh sessions failed",
					zap.String("plugin", p.Name()),
					zap.Uint64("userID", rawID),
					zap.Error(err),
				)
			}

			httpx.WriteLocalizedError(ginCtx, ctx.I18n, status, messageKey, nil)
			return
		}

		if requestAuth, ok := pluginapi.RequestAuthContextFromContext(ginCtx.Request.Context()); ok &&
			requestAuth.Claims != nil &&
			requestAuth.Claims.UserID == rawID {
			authSvc.cookies.clearRefreshCookie(ginCtx)
		}

		ginCtx.Status(http.StatusNoContent)
	})

	return nil
}

// Boot 在注册完成后启动用户插件的运行时行为。
//
// 当前用户插件没有额外后台资源需要启动，因此保持空实现，便于后续能力扩展时继续沿用显式生命周期钩子。
func (p *Plugin) Boot(ctx *plugin.Context) error {
	return nil
}

// Shutdown 在应用停止时释放用户插件资源。
//
// 当前实现没有自主管理的外部资源，因此关闭阶段保持幂等空操作。
func (p *Plugin) Shutdown(ctx *plugin.Context) error {
	return nil
}

type userService struct {
	users store.UserRepository
}

type authService struct {
	auth          store.AuthRepository
	users         store.UserRepository
	passwords     passwordHasher
	tokens        *accessTokenManager
	refreshTokens *refreshTokenManager
	cookies       authCookieManager
}

const maxSessionListLimit = 100

// GetUserByID 通过稳定仓储契约读取用户，并收敛为跨插件 DTO。
func (s userService) GetUserByID(ctx context.Context, id uint64) (pluginapi.UserSummary, error) {
	record, err := s.users.GetByID(ctx, id)
	if err != nil {
		return pluginapi.UserSummary{}, err
	}

	return pluginapi.UserSummary{
		ID:       record.ID,
		Username: record.Username,
		Display:  record.Display,
	}, nil
}

// CurrentUser 根据请求上下文中已解析的访问令牌声明返回当前主体摘要。
//
// 该实现要求调用链先通过鉴权中间件写入稳定 claims，再按用户仓储读取跨
// 插件可见的最小用户资料，不把 token 解析细节泄漏给业务调用方。
func (s authService) CurrentUser(ctx context.Context) (*pluginapi.CurrentUser, error) {
	if s.users == nil {
		return nil, errors.New("user repository is unavailable")
	}

	requestAuth, ok := pluginapi.RequestAuthContextFromContext(ctx)
	if !ok || requestAuth.Claims == nil {
		return nil, pluginapi.ErrUnauthenticated
	}

	record, err := s.users.GetByID(ctx, requestAuth.Claims.UserID)
	if err != nil {
		if errors.Is(err, store.ErrUserNotFound) {
			return nil, pluginapi.ErrUnauthenticated
		}
		return nil, err
	}

	return &pluginapi.CurrentUser{
		ID:          record.ID,
		Username:    record.Username,
		DisplayName: record.Display,
	}, nil
}

// ParseAccessToken 校验 access token 并返回跨插件稳定 claims。
func (s authService) ParseAccessToken(ctx context.Context, token string) (*pluginapi.AccessTokenClaims, error) {
	if s.tokens == nil {
		return nil, errors.New("access token manager is unavailable")
	}

	claims, err := s.tokens.Parse(strings.TrimSpace(token))
	if err != nil {
		switch {
		case errors.Is(err, errExpiredAccessToken):
			return nil, pluginapi.ErrExpiredAccessToken
		case errors.Is(err, errInvalidAccessToken):
			return nil, pluginapi.ErrInvalidAccessToken
		default:
			return nil, err
		}
	}

	if err := s.validateAccessSession(ctx, claims); err != nil {
		if errors.Is(err, errAccessSessionFailed) {
			return nil, pluginapi.ErrInvalidAccessToken
		}
		return nil, err
	}

	return claims, nil
}

var _ pluginapi.AuthService = authService{}

// parseUserID 将路由参数转换为插件内部统一使用的正整数 ID。
func parseUserID(input string) (uint64, error) {
	id, err := strconv.ParseUint(input, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse user id %q: %w", input, err)
	}
	if id == 0 {
		return 0, errors.New("id must be greater than zero")
	}
	return id, nil
}

// parseSessionListOptions 将列表查询参数收敛为插件内最小会话列表约束。
//
// 当前只允许显式 limit，并把约束留在插件层，避免为了轻量分页提前扩展仓储
// 或跨插件契约。
func parseSessionListOptions(rawLimit string) (sessionListOptions, error) {
	rawLimit = strings.TrimSpace(rawLimit)
	if rawLimit == "" {
		return sessionListOptions{}, nil
	}

	limit, err := strconv.Atoi(rawLimit)
	if err != nil {
		return sessionListOptions{}, fmt.Errorf("parse session limit %q: %w", rawLimit, err)
	}
	if limit <= 0 || limit > maxSessionListLimit {
		return sessionListOptions{}, fmt.Errorf("session limit %d is out of range", limit)
	}

	return sessionListOptions{Limit: limit}, nil
}

func mapAuthError(err error) (int, string) {
	switch {
	case errors.Is(err, pluginapi.ErrUnauthenticated):
		return http.StatusUnauthorized, "auth.missing_actor"
	case errors.Is(err, errInvalidLoginCredentials):
		return http.StatusUnauthorized, "auth.invalid_credentials"
	case errors.Is(err, errSessionNotFound):
		return http.StatusNotFound, "auth.session_not_found"
	case errors.Is(err, errRefreshSessionFailed):
		return http.StatusUnauthorized, "auth.invalid_refresh_session"
	default:
		return http.StatusInternalServerError, "common.internal_error"
	}
}
