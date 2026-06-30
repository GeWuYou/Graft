package project

import (
	"errors"

	"graft/server/internal/module"
)

// Module owns batch-2 project registry, import, refresh, and readonly configuration routes.
type Module struct {
	service *Service
}

// NewModule creates the project module instance.
func NewModule(service *Service) *Module {
	return &Module{service: service}
}

// Register wires batch-2 project routes.
func (m *Module) Register(ctx *module.Context) error {
	if m == nil || m.service == nil {
		return errors.New("project module service is unavailable")
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
