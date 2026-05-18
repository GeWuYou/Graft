package user

import (
	"graft/server/internal/plugin"
	"graft/server/plugins/user/storeadapter"
)

// NewDescriptor exposes the user plugin's stable metadata and builder.
func NewDescriptor() plugin.Descriptor {
	instance := NewPlugin(nil, nil)

	return plugin.Descriptor{
		ID:            instance.Name(),
		PluginVersion: instance.Version(),
		Dependencies:  append([]string(nil), instance.DependsOn()...),
		Builder: plugin.BuilderFunc(func(ctx plugin.BuildContext) (plugin.Plugin, error) {
			return NewPlugin(
				storeadapter.NewUserRepositoryAdapter(ctx.Stores.Users()),
				storeadapter.NewAuthRepositoryAdapter(ctx.Stores.Auth()),
			), nil
		}),
	}
}
