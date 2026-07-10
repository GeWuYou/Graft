package task

import (
	"errors"

	"graft/server/internal/module"
	taskstore "graft/server/modules/task/store"
)

// Module owns persisted Task Runtime facts. Worker, HTTP, and realtime behavior
// are introduced in later Task Runtime batches.
type Module struct {
	repository taskstore.Repository
}

// NewModule creates the Task Runtime module around its module-owned repository.
func NewModule(repository taskstore.Repository) *Module {
	return &Module{repository: repository}
}

// Register validates that the Task Runtime persistence boundary is available.
func (m *Module) Register(_ *module.Context) error {
	if m == nil || m.repository == nil {
		return errors.New("task module repository is unavailable")
	}
	return nil
}

// Boot intentionally starts no background work in the persistence batch.
func (m *Module) Boot(_ *module.Context) error {
	return nil
}

// Shutdown has no owned runtime resources before the worker batch.
func (m *Module) Shutdown(_ *module.Context) error {
	return nil
}
