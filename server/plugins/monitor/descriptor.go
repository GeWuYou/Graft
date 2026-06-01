package monitor

import "graft/server/internal/plugin"

const (
	moduleID = "monitor"
)

// NewModuleSpec exposes the monitor module's stable compile-time metadata and builder.
func NewModuleSpec() plugin.ModuleSpec {
	return plugin.ModuleSpec{
		ID:           moduleID,
		Dependencies: []string{"user", "rbac"},
		Builder: plugin.BuilderFunc(func(plugin.BuildContext) (plugin.Plugin, error) {
			return NewPlugin(), nil
		}),
	}
}
