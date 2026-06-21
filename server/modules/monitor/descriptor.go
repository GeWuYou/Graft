package monitor

import "graft/server/internal/module"

const (
	moduleID = "monitor"
)

// NewModuleSpec exposes the monitor module's stable compile-time metadata and builder.
func NewModuleSpec() module.Spec {
	return module.Spec{
		ID:           moduleID,
		Dependencies: []string{"user", "rbac"},
		Builder: module.BuilderFunc(func(module.BuildContext) (module.Module, error) {
			return NewModule(), nil
		}),
	}
}
