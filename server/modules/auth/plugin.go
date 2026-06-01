package auth

import (
	"context"
	"errors"
	"fmt"

	"graft/server/internal/module"
	"graft/server/internal/pluginapi"
)

// Plugin 是 auth 插件的认证与会话生命周期运行时入口。
type Plugin struct{}

// NewPlugin 创建 auth 插件最小骨架实例。
func NewPlugin() *Plugin {
	return &Plugin{}
}

// Register 声明 auth 插件拥有的 `/auth/*` 运行时路由。
func (p *Plugin) Register(ctx *module.Context) error {
	authService, err := resolveService[pluginapi.AuthService](ctx, (*pluginapi.AuthService)(nil), "auth service")
	if err != nil {
		return err
	}
	authFlow, err := resolveService[pluginapi.AuthFlowService](ctx, (*pluginapi.AuthFlowService)(nil), "auth flow service")
	if err != nil {
		return err
	}

	return registerAuthRoutes(ctx, moduleID, authService, authFlow)
}

// Boot 当前没有额外运行时行为需要启动。
func (p *Plugin) Boot(_ *module.Context) error {
	return nil
}

// Shutdown 当前没有额外资源需要释放。
func (p *Plugin) Shutdown(_ *module.Context) error {
	return nil
}

func resolveService[T any](ctx *module.Context, key any, label string) (T, error) {
	var zero T
	if ctx == nil || ctx.Services == nil {
		return zero, errors.New("plugin services are unavailable")
	}

	resolved, err := ctx.Services.Resolve(key)
	if err != nil {
		return zero, fmt.Errorf("resolve %s: %w", label, err)
	}

	service, ok := resolved.(T)
	if !ok {
		return zero, fmt.Errorf("resolve %s: unexpected type %T", label, resolved)
	}

	return service, nil
}

func currentRequestAuth(ctx context.Context) (pluginapi.RequestAuthContext, bool) {
	return pluginapi.RequestAuthContextFromContext(ctx)
}
