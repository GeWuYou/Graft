package task

import (
	"errors"

	"graft/server/internal/container"
	"graft/server/internal/httpx"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	"graft/server/internal/realtime"
	"graft/server/internal/realtimeauth"
	taskstore "graft/server/modules/task/store"
)

// Module 拥有 Task Runtime 的持久化事实，并统一承载 worker、HTTP 与 realtime 能力的模块生命周期。
type Module struct {
	repository taskstore.Repository
	runtime    *Runtime
}

// NewModule 基于模块自有仓储创建 Task Runtime 模块；仓储是状态机事实的持久化权威。
func NewModule(repository taskstore.Repository) *Module {
	return &Module{repository: repository, runtime: NewRuntime(repository)}
}

// Register 校验并发布 Task Runtime 能力，供消费模块注册阶段执行器和提交不可变 TaskPlan。
func (m *Module) Register(ctx *module.Context) error {
	if m == nil || m.repository == nil {
		return errors.New("task module repository is unavailable")
	}
	if ctx == nil || ctx.Services == nil || m.runtime == nil {
		return errors.New("task module register context is unavailable")
	}
	if err := m.registerServices(ctx); err != nil {
		return err
	}
	if err := m.configureRealtime(ctx); err != nil {
		return err
	}
	return registerTaskRoutes(ctx, m.runtime, httpx.NewSecurityAuditPublisher(ctx.EventBus, ctx.Logger, moduleID))
}

func (m *Module) registerServices(ctx *module.Context) error {
	if err := ctx.Services.RegisterSingleton((*moduleapi.TaskService)(nil), func(_ container.Resolver) (any, error) { return m.runtime, nil }); err != nil {
		return err
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.TaskQueryService)(nil), func(_ container.Resolver) (any, error) { return m.runtime, nil }); err != nil {
		return err
	}
	return ctx.Services.RegisterSingleton((*moduleapi.TaskRuntimeRegistrar)(nil), func(_ container.Resolver) (any, error) { return m.runtime, nil })
}

func (m *Module) configureRealtime(ctx *module.Context) error {
	tickets, err := module.ResolveService[realtimeauth.Service](ctx.Services, (*realtimeauth.Service)(nil))
	if err != nil {
		return err
	}
	hub, err := module.ResolveService[realtime.Hub](ctx.Services, (*realtime.Hub)(nil))
	if err != nil {
		return err
	}
	issuers, err := module.ResolveService[realtime.TopicIssuerRegistry](ctx.Services, (*realtime.TopicIssuerRegistry)(nil))
	if err != nil {
		return err
	}
	m.runtime.SetRealtime(tickets, hub, issuers)
	return m.runtime.RegisterRealtimeTopics()
}

// Boot 在消费模块完成阶段执行器注册后先执行崩溃恢复，再启动模块自有的进程内 worker 池。
func (m *Module) Boot(ctx *module.Context) error {
	if m == nil || m.runtime == nil || ctx == nil || ctx.LifecycleContext == nil {
		return errors.New("task module boot context is unavailable")
	}
	return m.runtime.Start(ctx.LifecycleContext)
}

// Shutdown 协作停止活跃执行器和模块自有 worker；在途外部副作用无法证明完成时由恢复流程标记为 unknown。
func (m *Module) Shutdown(ctx *module.Context) error {
	if m == nil || m.runtime == nil {
		return nil
	}
	if ctx == nil || ctx.LifecycleContext == nil {
		return errors.New("task module shutdown context is unavailable")
	}
	return m.runtime.Stop(ctx.LifecycleContext)
}
