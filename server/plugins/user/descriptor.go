package user

import (
	"fmt"

	"graft/server/internal/plugin"
	"graft/server/internal/store"
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
			userRepo, err := plugin.ResolveService[store.UserRepository](ctx.Services, (*store.UserRepository)(nil))
			if err != nil {
				return nil, fmt.Errorf("resolve user repository: %w", err)
			}
			authRepo, err := plugin.ResolveService[store.AuthRepository](ctx.Services, (*store.AuthRepository)(nil))
			if err != nil {
				return nil, fmt.Errorf("resolve auth repository: %w", err)
			}

			return NewPlugin(
				storeadapter.NewUserRepositoryAdapter(userRepo),
				storeadapter.NewAuthRepositoryAdapter(authRepo),
			), nil
		}),
	}
}
