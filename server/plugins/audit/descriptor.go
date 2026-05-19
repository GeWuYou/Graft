package audit

import (
	"fmt"

	"graft/server/internal/ent"
	"graft/server/internal/plugin"
	"graft/server/plugins/audit/storeent"
)

const (
	pluginID      = "audit"
	pluginVersion = "0.1.0"
)

// NewDescriptor exposes the audit plugin's stable metadata and builder.
func NewDescriptor() plugin.Descriptor {
	return plugin.Descriptor{
		ID:            pluginID,
		PluginVersion: pluginVersion,
		Dependencies:  nil,
		Builder: plugin.BuilderFunc(func(ctx plugin.BuildContext) (plugin.Plugin, error) {
			client, err := plugin.ResolveService[*ent.Client](ctx.Services, (*ent.Client)(nil))
			if err != nil {
				return nil, fmt.Errorf("resolve ent client: %w", err)
			}
			repo, err := storeent.NewRepository(client)
			if err != nil {
				return nil, fmt.Errorf("build audit storeent repository: %w", err)
			}

			return NewPlugin(repo)
		}),
	}
}
