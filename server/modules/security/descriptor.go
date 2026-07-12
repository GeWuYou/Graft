package security

import "graft/server/internal/module"

const moduleID = "security"

// NewModuleSpec exposes the security overview module's compile-time metadata.
func NewModuleSpec() module.Spec {
	return module.Spec{
		ID:           moduleID,
		Dependencies: []string{"user", "rbac", "audit"},
		Builder: module.BuilderFunc(func(module.BuildContext) (module.Module, error) {
			return NewModule(), nil
		}),
	}
}
