package user

import (
	"fmt"

	"graft/server/internal/ent"
	"graft/server/internal/plugin"
	"graft/server/plugins/user/storeent"
)

const (
	pluginID      = "user"
	pluginVersion = "0.1.0"
)

// NewDescriptor exposes the user plugin's stable metadata and builder.
func NewDescriptor() plugin.Descriptor {
	return plugin.Descriptor{
		ID:            pluginID,
		PluginVersion: pluginVersion,
		Dependencies:  nil,
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
