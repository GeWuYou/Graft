package project

import (
	"errors"
	"fmt"

	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	"graft/server/internal/realtime"
	"graft/server/internal/realtimeauth"
)

// Module owns project registry, lifecycle, and managed-create contract surfaces.
type Module struct {
	service *Service
}

// NewModule 创建并返回项目模块实例。
func NewModule(service *Service) *Module {
	return &Module{service: service}
}

// Register wires project module messages, permissions, menu, config definitions, and routes.
//
//nolint:cyclop
func (m *Module) Register(ctx *module.Context) error {
	if m == nil || m.service == nil {
		return errors.New("project module service is unavailable")
	}
	runtimeReader, err := module.ResolveService[moduleapi.ContainerProjectRuntimeReader](ctx.Services, (*moduleapi.ContainerProjectRuntimeReader)(nil))
	if err != nil {
		return fmt.Errorf("resolve container project runtime reader: %w", err)
	}
	resourceReader, err := module.ResolveService[moduleapi.ContainerProjectResourceReader](ctx.Services, (*moduleapi.ContainerProjectResourceReader)(nil))
	if err != nil {
		return fmt.Errorf("resolve container project resource reader: %w", err)
	}
	logReader, err := module.ResolveService[moduleapi.ContainerProjectLogReader](ctx.Services, (*moduleapi.ContainerProjectLogReader)(nil))
	if err != nil {
		return fmt.Errorf("resolve container project log reader: %w", err)
	}
	configResolver, err := module.ResolveService[moduleapi.SystemConfigResolver](ctx.Services, (*moduleapi.SystemConfigResolver)(nil))
	if err != nil {
		return fmt.Errorf("resolve system config resolver: %w", err)
	}
	authorizer, err := resolveAuthorizer(ctx)
	if err != nil {
		return fmt.Errorf("resolve authorizer: %w", err)
	}
	realtimeTickets, err := module.ResolveService[realtimeauth.Service](ctx.Services, (*realtimeauth.Service)(nil))
	if err != nil {
		return fmt.Errorf("resolve realtime ticket service: %w", err)
	}
	realtimeHub, err := module.ResolveService[realtime.Hub](ctx.Services, (*realtime.Hub)(nil))
	if err != nil {
		return fmt.Errorf("resolve realtime hub: %w", err)
	}
	topicIssuers, err := module.ResolveService[realtime.TopicIssuerRegistry](ctx.Services, (*realtime.TopicIssuerRegistry)(nil))
	if err != nil {
		return fmt.Errorf("resolve realtime topic issuer registry: %w", err)
	}
	m.service.SetRuntimeReader(runtimeReader)
	m.service.SetResourceReader(resourceReader)
	m.service.SetLogReader(logReader)
	m.service.SetSystemConfigResolver(configResolver)
	m.service.SetAuthorizer(authorizer)
	m.service.SetRealtime(realtimeTickets, realtimeHub, topicIssuers)
	m.service.SetAuditPublisher(ctx.EventBus, ctx.Logger, moduleID)
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

// Boot currently has no runtime-owned background work.
func (m *Module) Boot(_ *module.Context) error {
	return nil
}

// Shutdown currently has no runtime-owned resources to close.
func (m *Module) Shutdown(ctx *module.Context) error {
	if m == nil || m.service == nil || ctx == nil || ctx.LifecycleContext == nil {
		return nil
	}
	return m.service.Close(ctx.LifecycleContext)
}
