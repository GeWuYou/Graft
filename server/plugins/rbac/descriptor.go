package rbac

import (
	"graft/server/internal/plugin"
	"graft/server/plugins/rbac/storeadapter"
)

// NewDescriptor exposes the RBAC plugin's stable metadata and builder.
func NewDescriptor() plugin.Descriptor {
	instance := NewPlugin(nil)

	return plugin.Descriptor{
		ID:            instance.Name(),
		PluginVersion: instance.Version(),
		Dependencies:  append([]string(nil), instance.DependsOn()...),
		Builder: plugin.BuilderFunc(func(ctx plugin.BuildContext) (plugin.Plugin, error) {
			return NewPlugin(storeadapter.NewInternalRepositoryAdapter(ctx.Stores.RBAC())), nil
		}),
	}
}
