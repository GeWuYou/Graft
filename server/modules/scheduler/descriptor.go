package scheduler

import "graft/server/internal/module"

// NewModuleSpec exposes the scheduler module's stable compile-time metadata and builder.
func NewModuleSpec() module.Spec {
	return module.Spec{
		ID:           moduleID,
		Dependencies: nil,
		Builder:      module.BuilderFunc(func(module.BuildContext) (module.Plugin, error) { return NewPlugin(), nil }),
	}
}
