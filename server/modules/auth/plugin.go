package auth

import (
	"context"
	"errors"
	"fmt"

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
	authService, err := resolveService[moduleapi.AuthService](ctx, (*moduleapi.AuthService)(nil), "auth service")
	if err != nil {
		return err
	}
	authFlow, err := resolveService[moduleapi.AuthFlowService](ctx, (*moduleapi.AuthFlowService)(nil), "auth flow service")
	if err != nil {
		return err
	}

	return registerAuthRoutes(ctx, moduleID, authService, authFlow)
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
