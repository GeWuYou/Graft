package rbac

import (
	"fmt"

	"graft/server/internal/ent"
	"graft/server/internal/plugin"
	"graft/server/plugins/rbac/storeent"
)

const (
	pluginID      = "rbac"
	pluginVersion = "0.1.0"
)

var pluginDependencies = []string{"user"}

// NewDescriptor exposes the RBAC plugin's stable metadata and builder.
func NewDescriptor() plugin.Descriptor {
	return plugin.Descriptor{
		ID:            pluginID,
		PluginVersion: pluginVersion,
		Dependencies:  append([]string(nil), pluginDependencies...),
		MigrationPath: []string{"plugins/rbac/migrations"},
		Builder: plugin.BuilderFunc(func(ctx plugin.BuildContext) (plugin.Plugin, error) {
			client, err := plugin.ResolveService[*ent.Client](ctx.Services, (*ent.Client)(nil))
			if err != nil {
				return nil, fmt.Errorf("resolve ent client: %w", err)
			}
			repo, err := storeent.NewRepository(client)
			if err != nil {
				return nil, fmt.Errorf("build rbac storeent repository: %w", err)
			}

			return NewPlugin(repo), nil
		}),
	}
}
