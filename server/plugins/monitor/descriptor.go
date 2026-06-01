package monitor

import "graft/server/internal/plugin"

const (
	pluginID      = "monitor"
	pluginVersion = "0.1.0"
)

var pluginDependencies = []string{"user", "rbac"}

// NewDescriptor exposes the monitor module's stable compile-time metadata and builder under historical plugin naming.
func NewDescriptor() plugin.Descriptor {
	return plugin.Descriptor{
		ID:            pluginID,
		PluginVersion: pluginVersion,
		Dependencies:  append([]string(nil), pluginDependencies...),
		Builder: plugin.BuilderFunc(func(plugin.BuildContext) (plugin.Plugin, error) {
			return NewPlugin(), nil
		}),
	}
}
