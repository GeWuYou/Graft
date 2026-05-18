package user

import "graft/server/internal/plugin"

// NewDescriptor exposes the user plugin's stable metadata and builder.
func NewDescriptor() plugin.Descriptor {
	instance := NewPlugin()

	return plugin.Descriptor{
		ID:            instance.Name(),
		PluginVersion: instance.Version(),
		Dependencies:  append([]string(nil), instance.DependsOn()...),
		Builder:       plugin.BuilderFunc(func() (plugin.Plugin, error) { return NewPlugin(), nil }),
	}
}
