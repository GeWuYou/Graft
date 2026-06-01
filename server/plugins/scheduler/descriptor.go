package scheduler

import "graft/server/internal/plugin"

// NewModuleSpec exposes the scheduler module's stable compile-time metadata and builder.
func NewModuleSpec() plugin.ModuleSpec {
	return plugin.ModuleSpec{
		ID:           moduleID,
		Dependencies: nil,
		Builder:      plugin.BuilderFunc(func(plugin.BuildContext) (plugin.Plugin, error) { return NewPlugin(), nil }),
	}
}
