package auth

import "graft/server/internal/plugin"

const (
	pluginID      = "auth"
	pluginVersion = "0.1.0"
)

// NewDescriptor exposes the auth module's stable compile-time metadata and builder under historical plugin naming.
func NewDescriptor() plugin.Descriptor {
	return plugin.Descriptor{
		ID:            pluginID,
		PluginVersion: pluginVersion,
		Dependencies:  []string{"user"},
		MigrationPath: []string{"plugins/auth/migrations"},
		Builder: plugin.BuilderFunc(func(plugin.BuildContext) (plugin.Plugin, error) {
			return NewPlugin(), nil
		}),
	}
}
