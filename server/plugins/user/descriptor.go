package user

import (
	"fmt"

	"graft/server/internal/ent"
	"graft/server/internal/plugin"
	"graft/server/plugins/user/storeent"
)

// NewDescriptor exposes the user plugin's stable metadata and builder.
func NewDescriptor() plugin.Descriptor {
	instance := NewPlugin(nil, nil)

	return plugin.Descriptor{
		ID:            instance.Name(),
		PluginVersion: instance.Version(),
		Dependencies:  append([]string(nil), instance.DependsOn()...),
		MigrationPath: []string{"plugins/user/migrations"},
		Builder: plugin.BuilderFunc(func(ctx plugin.BuildContext) (plugin.Plugin, error) {
			client, err := plugin.ResolveService[*ent.Client](ctx.Services, (*ent.Client)(nil))
			if err != nil {
				return nil, fmt.Errorf("resolve ent client: %w", err)
			}
			userRepo, err := storeent.NewUserRepository(client)
			if err != nil {
				return nil, fmt.Errorf("build user storeent repository: %w", err)
			}
			authRepo, err := storeent.NewAuthRepository(client)
			if err != nil {
				return nil, fmt.Errorf("build user auth storeent repository: %w", err)
			}

			return NewPlugin(userRepo, authRepo), nil
		}),
	}
}
