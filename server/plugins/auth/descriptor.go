package auth

import "graft/server/internal/plugin"

const (
	moduleID = "auth"
)

// NewModuleSpec exposes the auth module's stable compile-time metadata and builder.
func NewModuleSpec() plugin.ModuleSpec {
	return plugin.ModuleSpec{
		ID:            moduleID,
		Dependencies:  []string{"user"},
		MigrationPath: []string{"plugins/auth/migrations"},
		Builder: plugin.BuilderFunc(func(plugin.BuildContext) (plugin.Plugin, error) {
			return NewPlugin(), nil
		}),
	}
}
