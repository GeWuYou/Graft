package scheduler

import "graft/server/internal/plugin"

// NewDescriptor exposes the scheduler module's stable compile-time metadata and builder under historical plugin naming.
func NewDescriptor() plugin.Descriptor {
	instance := NewPlugin()

	return plugin.Descriptor{
		ID:            instance.Name(),
		PluginVersion: instance.Version(),
		Dependencies:  append([]string(nil), instance.DependsOn()...),
		Builder:       plugin.BuilderFunc(func(plugin.BuildContext) (plugin.Plugin, error) { return NewPlugin(), nil }),
	}
}
