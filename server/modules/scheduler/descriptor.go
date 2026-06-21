package scheduler

import "graft/server/internal/module"

// NewModuleSpec exposes the scheduler module's stable compile-time metadata and builder.
func NewModuleSpec() module.Spec {
	return module.Spec{
		ID:            moduleID,
		Dependencies:  []string{"notification", "system-config"},
		MigrationPath: []string{"modules/scheduler/migrations"},
		Builder:       module.BuilderFunc(func(module.BuildContext) (module.Module, error) { return NewModule(), nil }),
	}
}
