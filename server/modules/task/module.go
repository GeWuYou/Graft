package task

import (
	"errors"

	"graft/server/internal/container"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	taskstore "graft/server/modules/task/store"
)

// Module owns persisted Task Runtime facts. Worker, HTTP, and realtime behavior
// are introduced in later Task Runtime batches.
type Module struct {
	repository taskstore.Repository
	runtime    *Runtime
}

// NewModule creates the Task Runtime module around its module-owned repository.
func NewModule(repository taskstore.Repository) *Module {
	return &Module{repository: repository, runtime: NewRuntime(repository)}
}

// Register validates and publishes the Task Runtime capability for consumer
// modules to register Stage executors and submit TaskPlans.
func (m *Module) Register(ctx *module.Context) error {
	if m == nil || m.repository == nil {
		return errors.New("task module repository is unavailable")
	}
	if ctx == nil || ctx.Services == nil || m.runtime == nil {
		return errors.New("task module register context is unavailable")
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.TaskService)(nil), func(_ container.Resolver) (any, error) {
		return m.runtime, nil
	}); err != nil {
		return err
	}
	return ctx.Services.RegisterSingleton((*moduleapi.TaskRuntimeRegistrar)(nil), func(_ container.Resolver) (any, error) {
		return m.runtime, nil
	})
}

// Boot starts recovery and the module-owned in-process worker pool after all
// consumer modules have had a chance to register their Stage executors.
func (m *Module) Boot(ctx *module.Context) error {
	if m == nil || m.runtime == nil || ctx == nil || ctx.LifecycleContext == nil {
		return errors.New("task module boot context is unavailable")
	}
	return m.runtime.Start(ctx.LifecycleContext)
}

// Shutdown cooperatively stops active executors and owned worker goroutines.
func (m *Module) Shutdown(ctx *module.Context) error {
	if m == nil || m.runtime == nil {
		return nil
	}
	if ctx == nil || ctx.LifecycleContext == nil {
		return errors.New("task module shutdown context is unavailable")
	}
	return m.runtime.Stop(ctx.LifecycleContext)
}
