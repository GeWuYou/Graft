package project

import (
	"errors"
	"fmt"

	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
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
func (m *Module) Register(ctx *module.Context) error {
	if m == nil || m.service == nil {
		return errors.New("project module service is unavailable")
	}
	runtimeReader, err := module.ResolveService[moduleapi.ContainerProjectRuntimeReader](ctx.Services, (*moduleapi.ContainerProjectRuntimeReader)(nil))
	if err != nil {
		return fmt.Errorf("resolve container project runtime reader: %w", err)
	}
	configResolver, err := module.ResolveService[moduleapi.SystemConfigResolver](ctx.Services, (*moduleapi.SystemConfigResolver)(nil))
	if err != nil {
		return fmt.Errorf("resolve system config resolver: %w", err)
	}
	m.service.SetRuntimeReader(runtimeReader)
	m.service.SetSystemConfigResolver(configResolver)
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
func (m *Module) Shutdown(_ *module.Context) error {
	return nil
}
