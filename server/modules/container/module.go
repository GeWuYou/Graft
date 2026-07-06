package container

import (
	"errors"

	containerdi "graft/server/internal/container"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
)

// Module declares the container management module foundation.
type Module struct {
	service *service
}

// NewModule creates a container management module instance.
func NewModule() *Module {
	return &Module{}
}

// Register declares container menu, permissions, messages, config definitions, and routes.
func (m *Module) Register(ctx *module.Context) error {
	if m == nil {
		return errors.New("container module is unavailable")
	}
	if err := registerMessages(ctx.I18n); err != nil {
		return err
	}
	if err := registerPermissions(ctx.PermissionRegistry, moduleID); err != nil {
		return err
	}
	if err := registerMenu(ctx.MenuRegistry, moduleID); err != nil {
		return err
	}
	if err := registerConfig(ctx.I18n, ctx.ConfigRegistry); err != nil {
		return err
	}
	service, err := newContainerService(ctx, moduleID)
	if err != nil {
		return err
	}
	if err := service.registerRealtimeTopics(); err != nil {
		return err
	}
	if err := registerModuleServices(ctx, service); err != nil {
		return err
	}
	m.service = service
	return registerRoutes(ctx, moduleID, service)
}

// Boot starts the module-owned stats collector when realtime publishing is available.
func (m *Module) Boot(ctx *module.Context) error {
	if m == nil || m.service == nil {
		return nil
	}
	if ctx == nil || ctx.LifecycleContext == nil {
		return errors.New("container lifecycle context is required")
	}
	if err := m.service.startStatsCollector(ctx.LifecycleContext); err != nil {
		return err
	}
	if err := m.service.startRuntimeEventManager(ctx.LifecycleContext); err != nil {
		if stopErr := m.service.stopStatsCollector(ctx.LifecycleContext); stopErr != nil {
			return errors.Join(err, stopErr)
		}
		return err
	}
	return nil
}

// Shutdown stops module-owned background work and releases the runtime client.
func (m *Module) Shutdown(ctx *module.Context) error {
	if m == nil || m.service == nil {
		return nil
	}
	if ctx == nil || ctx.LifecycleContext == nil {
		return errors.New("container shutdown lifecycle context is required")
	}
	return m.service.Close(ctx.LifecycleContext)
}

// registerModuleServices 向模块服务注册器登记容器项目运行时读取器的单例实现。
// 它要求模块上下文、服务注册器和运行时服务都可用。
func registerModuleServices(ctx *module.Context, service *service) error {
	if ctx == nil || ctx.Services == nil {
		return errors.New("container service registry is unavailable")
	}
	if service == nil {
		return errors.New("container runtime reader is unavailable")
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.ContainerProjectRuntimeReader)(nil), func(_ containerdi.Resolver) (any, error) {
		return containerProjectRuntimeReader{service: service}, nil
	}); err != nil {
		return err
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.ContainerProjectResourceReader)(nil), func(_ containerdi.Resolver) (any, error) {
		return containerProjectRuntimeReader{service: service}, nil
	}); err != nil {
		return err
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.ContainerProjectLogReader)(nil), func(_ containerdi.Resolver) (any, error) {
		return containerProjectRuntimeReader{service: service}, nil
	}); err != nil {
		return err
	}
	return nil
}
