package auth

import (
	"context"
	"errors"
	"fmt"

	"graft/server/internal/container"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
)

// Module 是 auth 模块的认证与会话生命周期运行时入口。
type Module struct{}

// NewModule 创建 auth 模块最小骨架实例。
func NewModule() *Module {
	return &Module{}
}

// Register 声明 auth 模块拥有的 `/auth/*` 运行时路由。
func (p *Module) Register(ctx *module.Context) error {
	provider, err := resolveService[moduleapi.AuthCapabilityProvider](ctx, (*moduleapi.AuthCapabilityProvider)(nil), "auth capability provider")
	if err != nil {
		return err
	}
	capabilities := provider.AuthCapabilities()
	if capabilities.Auth == nil || capabilities.Sessions == nil || capabilities.Flow == nil || capabilities.Credentials == nil {
		return errors.New("auth capability provider returned incomplete capabilities")
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.AuthService)(nil), func(container.Resolver) (any, error) {
		return capabilities.Auth, nil
	}); err != nil {
		return fmt.Errorf("register auth service: %w", err)
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.AuthSessionService)(nil), func(container.Resolver) (any, error) {
		return capabilities.Sessions, nil
	}); err != nil {
		return fmt.Errorf("register auth session service: %w", err)
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.AuthFlowService)(nil), func(container.Resolver) (any, error) {
		return capabilities.Flow, nil
	}); err != nil {
		return fmt.Errorf("register auth flow service: %w", err)
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.AuthCredentialManagementService)(nil), func(container.Resolver) (any, error) {
		return capabilities.Credentials, nil
	}); err != nil {
		return fmt.Errorf("register auth credential management service: %w", err)
	}

	return registerAuthRoutes(ctx, moduleID, capabilities.Auth, capabilities.Flow)
}

// Boot 当前没有额外运行时行为需要启动。
func (p *Module) Boot(_ *module.Context) error {
	return nil
}

// Shutdown 当前没有额外资源需要释放。
func (p *Module) Shutdown(_ *module.Context) error {
	return nil
}

func resolveService[T any](ctx *module.Context, key any, label string) (T, error) {
	var zero T
	if ctx == nil || ctx.Services == nil {
		return zero, errors.New("module services are unavailable")
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

func currentRequestAuth(ctx context.Context) (moduleapi.RequestAuthContext, bool) {
	return moduleapi.RequestAuthContextFromContext(ctx)
}
