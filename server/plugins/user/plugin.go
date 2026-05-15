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

	ctx.MenuRegistry.Register(menu.Item{
		Code:       "user.list",
		Title:      "用户管理",
		Path:       "/users",
		Icon:       "usergroup",
		Permission: "user.read",
		Plugin:     p.Name(),
	})

	if err := ctx.Services.RegisterSingleton((*pluginapi.UserService)(nil), func(resolver container.Resolver) (any, error) {
		return userService{users: ctx.Stores.Users()}, nil
	}); err != nil {
		return err
	}

	if err := ctx.Services.RegisterSingleton((*pluginapi.AuthService)(nil), func(resolver container.Resolver) (any, error) {
		tokenManager, err := newAccessTokenManager(ctx.Config.Auth)
		if err != nil {
			return nil, err
		}

		return authService{
			users:  ctx.Stores.Users(),
			tokens: tokenManager,
		}, nil
	}); err != nil {
		return err
	}

	group := ctx.Router.Group("/users")
	group.Use(httpx.RequirePermission(ctx.I18n, ctx.Services, "user.read"))
	group.GET("/:id", func(ginCtx *gin.Context) {
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
	users  store.UserRepository
	tokens *accessTokenManager
}

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
