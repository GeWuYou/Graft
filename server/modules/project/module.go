package project

import (
	"errors"
	"fmt"

	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	"graft/server/internal/realtime"
	"graft/server/internal/realtimeauth"
)

// Module 拥有项目注册表、生命周期和受管创建契约的运行时装配面。
type Module struct {
	service *Service
}

// NewModule 创建并返回项目模块实例。
func NewModule(service *Service) *Module {
	return &Module{service: service}
}

// Register 按消息、权限、菜单、配置、路由和实时主题的声明顺序装配项目模块。
// 该阶段只注册契约，不启动后台流或其它长生命周期资源。
//
//nolint:cyclop
func (m *Module) Register(ctx *module.Context) error {
	if m == nil || m.service == nil {
		return errors.New("project module service is unavailable")
	}
	taskRegistrar, err := m.configureService(ctx)
	if err != nil {
		return err
	}
	if err := registerProjectTaskExecutors(taskRegistrar, m.service); err != nil {
		return err
	}
	if err := m.service.registerRealtimeTopics(); err != nil {
		return err
	}
	if err := registerPermissions(ctx.PermissionRegistry, moduleID); err != nil {
		return err
	}
	if err := registerMenu(ctx.MenuRegistry, moduleID); err != nil {
		return err
	}
	if err := registerConfig(ctx.ConfigRegistry); err != nil {
		return err
	}
	return registerRoutes(ctx, moduleID, m.service)
}

//nolint:cyclop // Explicit module dependency resolution keeps DI wiring auditable.
func (m *Module) configureService(ctx *module.Context) (moduleapi.TaskRuntimeRegistrar, error) {
	runtimeReader, err := module.ResolveService[moduleapi.ContainerProjectRuntimeReader](ctx.Services, (*moduleapi.ContainerProjectRuntimeReader)(nil))
	if err != nil {
		return nil, fmt.Errorf("resolve container project runtime reader: %w", err)
	}
	resourceReader, err := module.ResolveService[moduleapi.ContainerProjectResourceReader](ctx.Services, (*moduleapi.ContainerProjectResourceReader)(nil))
	if err != nil {
		return nil, fmt.Errorf("resolve container project resource reader: %w", err)
	}
	taskService, err := module.ResolveService[moduleapi.TaskService](ctx.Services, (*moduleapi.TaskService)(nil))
	if err != nil {
		return nil, fmt.Errorf("resolve task service: %w", err)
	}
	taskRegistrar, err := module.ResolveService[moduleapi.TaskRuntimeRegistrar](ctx.Services, (*moduleapi.TaskRuntimeRegistrar)(nil))
	if err != nil {
		return nil, fmt.Errorf("resolve task runtime registrar: %w", err)
	}
	logReader, err := module.ResolveService[moduleapi.ContainerProjectLogReader](ctx.Services, (*moduleapi.ContainerProjectLogReader)(nil))
	if err != nil {
		return nil, fmt.Errorf("resolve container project log reader: %w", err)
	}
	configResolver, err := module.ResolveService[moduleapi.SystemConfigResolver](ctx.Services, (*moduleapi.SystemConfigResolver)(nil))
	if err != nil {
		return nil, fmt.Errorf("resolve system config resolver: %w", err)
	}
	savedViews, err := module.ResolveService[moduleapi.SavedViewService](ctx.Services, (*moduleapi.SavedViewService)(nil))
	if err != nil {
		return nil, fmt.Errorf("resolve saved view service: %w", err)
	}
	runtimeTargets, err := resolveProjectRuntimeTargetReader(ctx)
	if err != nil {
		return nil, err
	}
	authorizer, err := resolveAuthorizer(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve authorizer: %w", err)
	}
	realtimeDeps, err := resolveProjectRealtime(ctx)
	if err != nil {
		return nil, err
	}
	m.service.SetRuntimeReader(runtimeReader)
	m.service.SetResourceReader(resourceReader)
	m.service.SetTaskService(taskService)
	if ctx.AppLogger != nil {
		m.service.SetAppLogger(ctx.AppLogger.Named("modules.project.lifecycle"))
	}
	m.service.SetLogReader(logReader)
	m.service.SetSystemConfigResolver(configResolver)
	m.service.SetSavedViewService(savedViews)
	m.service.SetRuntimeTargetReader(runtimeTargets)
	m.service.SetAuthorizer(authorizer)
	m.service.SetRealtime(realtimeDeps.tickets, realtimeDeps.hub, realtimeDeps.issuers)
	m.service.SetAuditPublisher(ctx.EventPublisher, ctx.Logger, moduleID)
	return taskRegistrar, nil
}

// resolveProjectRuntimeTargetReader 从模块服务注册表中解析项目运行时目标读取器。
// 如果解析失败，则返回包装后的错误。
func resolveProjectRuntimeTargetReader(ctx *module.Context) (moduleapi.ComposeRuntimeTargetReader, error) {
	reader, err := module.ResolveService[moduleapi.ComposeRuntimeTargetReader](ctx.Services, (*moduleapi.ComposeRuntimeTargetReader)(nil))
	if err != nil {
		return nil, fmt.Errorf("resolve runtime target reader: %w", err)
	}
	return reader, nil
}

type projectRealtimeDependencies struct {
	tickets realtimeauth.Service
	hub     realtime.Hub
	issuers realtime.TopicIssuerRegistry
}

// resolveProjectRealtime 解析项目实时通信所需的依赖。
// resolveProjectRealtime 解析实时票据服务、通信中心和主题发布者注册表；任一依赖解析失败时返回错误。
func resolveProjectRealtime(ctx *module.Context) (projectRealtimeDependencies, error) {
	tickets, err := module.ResolveService[realtimeauth.Service](ctx.Services, (*realtimeauth.Service)(nil))
	if err != nil {
		return projectRealtimeDependencies{}, fmt.Errorf("resolve realtime ticket service: %w", err)
	}
	hub, err := module.ResolveService[realtime.Hub](ctx.Services, (*realtime.Hub)(nil))
	if err != nil {
		return projectRealtimeDependencies{}, fmt.Errorf("resolve realtime hub: %w", err)
	}
	issuers, err := module.ResolveService[realtime.TopicIssuerRegistry](ctx.Services, (*realtime.TopicIssuerRegistry)(nil))
	if err != nil {
		return projectRealtimeDependencies{}, fmt.Errorf("resolve realtime topic issuer registry: %w", err)
	}
	return projectRealtimeDependencies{tickets: tickets, hub: hub, issuers: issuers}, nil
}

// Boot 启动项目模块的运行时工作；当前后台实时流按订阅懒启动，因此这里无需创建 goroutine。
func (m *Module) Boot(ctx *module.Context) error {
	if m == nil || m.service == nil || ctx == nil {
		return nil
	}
	if err := m.service.BackfillRuntimeTargets(ctx.LifecycleContext); err != nil {
		return err
	}
	return nil
}

// Shutdown 释放项目模块运行时资源；实时流由 Service.Close 在统一关闭路径中等待退出。
func (m *Module) Shutdown(ctx *module.Context) error {
	if m == nil || m.service == nil || ctx == nil || ctx.LifecycleContext == nil {
		return nil
	}
	return m.service.Close(ctx.LifecycleContext)
}
