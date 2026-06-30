package project

import (
	"errors"

	"graft/server/internal/module"
)

// Module owns project registry, lifecycle, and managed-create contract surfaces.
type Module struct {
	service *Service
}

// NewModule creates the project module instance.
func NewModule(service *Service) *Module {
	return &Module{service: service}
}

// Register wires project module messages, permissions, menu, config definitions, and routes.
func (m *Module) Register(ctx *module.Context) error {
	if m == nil || m.service == nil {
		return errors.New("project module service is unavailable")
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
func (m *Module) Shutdown(_ *module.Context) error {
	return nil
}
