package auth

import "graft/server/internal/module"

const (
	moduleID = "auth"
)

// NewModuleSpec exposes the auth module's stable compile-time metadata and builder.
func NewModuleSpec() module.Spec {
	return module.Spec{
		ID:            moduleID,
		Dependencies:  []string{"user"},
		MigrationPath: []string{"modules/auth/migrations"},
		Builder: module.BuilderFunc(func(module.BuildContext) (module.Module, error) {
			return NewModule(), nil
		}),
	}
}
