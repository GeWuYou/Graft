package auth

import (
	"context"
	"errors"
	"fmt"

	"graft/server/internal/container"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	authent "graft/server/modules/auth/ent"
	"graft/server/modules/auth/storeent"
)

// Module 是 auth 模块的认证与会话生命周期运行时入口。
type Module struct{ client *authent.Client }

// NewModule 创建 auth 模块实例；客户端由模块规格的 Builder 注入，缺失客户端时 Register 会拒绝启动。
func NewModule(client ...*authent.Client) *Module {
	module := &Module{}
	if len(client) > 0 {
		module.client = client[0]
	}
	return module
}

// Register 声明 auth 模块拥有的 `/auth/*` 运行时路由。
func (p *Module) Register(ctx *module.Context) error {
	if p.client == nil {
		return errors.New("auth ent client is unavailable")
	}
	authService, flow, err := p.newRuntime(ctx)
	if err != nil {
		return err
	}
	return p.registerCapabilitiesAndRoutes(ctx, authService, flow)
}

func (p *Module) newRuntime(ctx *module.Context) (*authService, authFlowBridge, error) {
	identity, err := resolveService[moduleapi.UserIdentityProvider](ctx, (*moduleapi.UserIdentityProvider)(nil), "user identity provider")
	if err != nil {
		return nil, authFlowBridge{}, err
	}
	bootstrap, err := resolveService[moduleapi.UserBootstrapProvider](ctx, (*moduleapi.UserBootstrapProvider)(nil), "user bootstrap provider")
	if err != nil {
		return nil, authFlowBridge{}, err
	}
	if p.client == nil {
		return nil, authFlowBridge{}, errors.New("auth ent client is unavailable")
	}
	credentials, err := storeent.NewCredentialStore(p.client, identity)
	if err != nil {
		return nil, authFlowBridge{}, err
	}
	sessions, err := storeent.NewSessionStore(p.client)
	if err != nil {
		return nil, authFlowBridge{}, err
	}
	personalTokens, err := storeent.NewPersonalAccessTokenStore(p.client)
	if err != nil {
		return nil, authFlowBridge{}, err
	}
	authService, err := newAuthService(ctx.Config.Auth, credentials, sessions, identity, personalTokens)
	if err != nil {
		return nil, authFlowBridge{}, err
	}
	flow := authFlowBridge{auth: authService, bootstrap: bootstrap}
	return authService, flow, nil
}

func (p *Module) registerCapabilitiesAndRoutes(ctx *module.Context, authService *authService, flow authFlowBridge) error {
	if err := ctx.Services.RegisterSingleton((*moduleapi.AuthService)(nil), func(container.Resolver) (any, error) {
		return authService, nil
	}); err != nil {
		return fmt.Errorf("register auth service: %w", err)
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.AuthSessionService)(nil), func(container.Resolver) (any, error) {
		return authService, nil
	}); err != nil {
		return fmt.Errorf("register auth session service: %w", err)
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.AuthFlowService)(nil), func(container.Resolver) (any, error) {
		return flow, nil
	}); err != nil {
		return fmt.Errorf("register auth flow service: %w", err)
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.AuthCredentialManagementService)(nil), func(container.Resolver) (any, error) {
		return authService, nil
	}); err != nil {
		return fmt.Errorf("register auth credential management service: %w", err)
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.PersonalAccessTokenService)(nil), func(container.Resolver) (any, error) {
		return authService, nil
	}); err != nil {
		return fmt.Errorf("register personal access token service: %w", err)
	}

	return registerAuthRoutes(ctx, moduleID, authService, authService, flow)
}

// Boot 在所有模块完成稳定 RBAC 引导能力注册后，创建 auth 所拥有的默认凭据。
func (p *Module) Boot(ctx *module.Context) error {
	identity, err := resolveService[moduleapi.UserIdentityProvider](ctx, (*moduleapi.UserIdentityProvider)(nil), "user identity provider")
	if err != nil {
		return err
	}
	rbacBootstrap, err := resolveService[moduleapi.RBACBootstrapService](ctx, (*moduleapi.RBACBootstrapService)(nil), "rbac bootstrap service")
	if err != nil {
		return err
	}
	credentials, err := storeent.NewCredentialStore(p.client, identity)
	if err != nil {
		return err
	}
	return ensureDefaultAdmin(ctx.LifecycleContext, ctx.I18n, credentials, identity, rbacBootstrap, ctx.PermissionRegistry.Items())
}

// Shutdown 释放 auth 不拥有的资源；数据库客户端生命周期由模块运行时统一管理。
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
