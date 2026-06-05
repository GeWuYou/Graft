package scheduler

import (
	"database/sql"
	"fmt"

	"graft/server/internal/container"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	schedulercore "graft/server/internal/scheduler"
)

const (
	moduleID = "scheduler"
)

// Module 是当前 MVP 阶段的最小调度模块。
//
// 该模块只负责在所有模块完成 Register 后，把 `cron registry` 中已声明的
// 任务装配到运行时调度器，并在 Boot / Shutdown 阶段统一完成“运行启动、
// 收敛关闭”。若 Boot 阶段任务装配或启动失败，模块不会进入可运行状态；
// Shutdown 会把运行时停止错误上抛给调用方，便于宿主决定是否继续整体退出流程。
type Module struct {
	runtime         schedulercore.Runtime
	routeAuth       *deferredAuthService
	routeAuthorizer *deferredAuthorizer
}

// NewModule 创建最小调度模块。
func NewModule() *Module {
	return &Module{}
}

// Register 声明 scheduler 模块对后续 API 路由可消费的运行时能力。
func (p *Module) Register(ctx *module.Context) error {
	if ctx == nil || ctx.Services == nil {
		return fmt.Errorf("scheduler register context is required")
	}

	if err := registerMessages(ctx.I18n); err != nil {
		return err
	}
	registerSchedulerPermissions(ctx.PermissionRegistry, moduleID)
	registerSchedulerMenu(ctx.MenuRegistry, moduleID)
	if err := registerSchedulerRuntimeService(ctx, p); err != nil {
		return err
	}
	p.routeAuth = newDeferredAuthService()
	p.routeAuthorizer = newDeferredAuthorizer()
	return registerSchedulerRoutesWithSecurity(ctx, moduleID, p.routeAuth, p.routeAuthorizer)
}

func registerSchedulerRuntimeService(ctx *module.Context, p *Module) error {
	return ctx.Services.RegisterSingleton((*schedulercore.Runtime)(nil), func(resolver container.Resolver) (any, error) {
		db, err := module.ResolveService[*sql.DB](resolver, (*sql.DB)(nil))
		if err != nil {
			return nil, err
		}

		repo, err := schedulercore.NewSQLRunRepository(db)
		if err != nil {
			return nil, err
		}
		taskRepo, err := schedulercore.NewSQLTaskRepository(db)
		if err != nil {
			return nil, err
		}

		if p.runtime != nil {
			return p.runtime, nil
		}
		runtime := schedulercore.New(ctx.Logger, repo)
		runtime.SetTaskRepository(taskRepo)
		return runtime, nil
	})
}

func resolveAuthService(ctx *module.Context) (moduleapi.AuthService, error) {
	resolved, err := ctx.Services.Resolve((*moduleapi.AuthService)(nil))
	if err != nil {
		return nil, fmt.Errorf("resolve auth service: %w", err)
	}
	authService, ok := resolved.(moduleapi.AuthService)
	if !ok || authService == nil {
		return nil, fmt.Errorf("resolve auth service: unexpected type %T", resolved)
	}
	return authService, nil
}

func resolveAuthorizer(ctx *module.Context) (moduleapi.Authorizer, error) {
	resolved, err := ctx.Services.Resolve((*moduleapi.Authorizer)(nil))
	if err != nil {
		return nil, fmt.Errorf("resolve route authorizer: %w", err)
	}
	authorizer, ok := resolved.(moduleapi.Authorizer)
	if !ok || authorizer == nil {
		return nil, fmt.Errorf("resolve route authorizer: unexpected type %T", resolved)
	}
	return authorizer, nil
}

func (p *Module) resolveRuntime(ctx *module.Context) (schedulercore.Runtime, error) {
	if p.runtime != nil {
		return p.runtime, nil
	}
	if ctx == nil || ctx.Services == nil {
		return nil, fmt.Errorf("scheduler services are required")
	}

	resolved, err := ctx.Services.Resolve((*schedulercore.Runtime)(nil))
	if err != nil {
		return nil, err
	}
	runtime, ok := resolved.(schedulercore.Runtime)
	if !ok || runtime == nil {
		return nil, fmt.Errorf("scheduler runtime service has unexpected type %T", resolved)
	}

	return runtime, nil
}

func (p *Module) ensureRuntimeService(ctx *module.Context) error {
	if p.runtime != nil {
		return nil
	}
	if ctx == nil || ctx.Services == nil {
		return fmt.Errorf("scheduler services are required")
	}
	_, err := ctx.Services.Resolve((*schedulercore.Runtime)(nil))
	if err != nil {
		return err
	}

	return nil
}

// Boot 在所有模块 Register 完成后装配并启动最小调度器。
func (p *Module) Boot(ctx *module.Context) error {
	if ctx == nil || ctx.CronRegistry == nil {
		return fmt.Errorf("scheduler boot context is required")
	}

	if err := p.bindRouteSecurityServices(ctx); err != nil {
		return err
	}

	if err := p.ensureRuntimeService(ctx); err != nil {
		return fmt.Errorf("resolve scheduler runtime: %w", err)
	}

	runtime, err := p.resolveRuntime(ctx)
	if err != nil {
		return fmt.Errorf("resolve scheduler runtime: %w", err)
	}

	if err := runtime.SeedBuiltinJobs(ctx.LifecycleContext, ctx.CronRegistry.Items()); err != nil {
		return fmt.Errorf("seed scheduler builtin jobs: %w", err)
	}

	if err := runtime.Start(ctx.LifecycleContext); err != nil {
		return fmt.Errorf("start scheduler runtime: %w", err)
	}

	p.runtime = runtime
	return nil
}

func (p *Module) bindRouteSecurityServices(ctx *module.Context) error {
	if p.routeAuth == nil || p.routeAuthorizer == nil {
		return nil
	}

	authService, err := resolveAuthService(ctx)
	if err != nil {
		return err
	}
	if err := p.routeAuth.SetTarget(authService); err != nil {
		return fmt.Errorf("bind scheduler route auth service: %w", err)
	}

	authorizer, err := resolveAuthorizer(ctx)
	if err != nil {
		return err
	}
	if err := p.routeAuthorizer.SetTarget(authorizer); err != nil {
		return fmt.Errorf("bind scheduler route authorizer: %w", err)
	}

	return nil
}

// Shutdown 停止当前调度器并等待在途任务收敛。
func (p *Module) Shutdown(ctx *module.Context) error {
	if p.runtime == nil {
		return nil
	}

	if ctx == nil || ctx.LifecycleContext == nil {
		return fmt.Errorf("scheduler shutdown lifecycle context is required")
	}

	return p.runtime.Stop(ctx.LifecycleContext)
}
