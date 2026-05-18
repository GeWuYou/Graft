package audit

import (
	"fmt"

	"graft/server/internal/plugin"
	"graft/server/internal/store"
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
			repo, err := plugin.ResolveService[store.AuditRepository](ctx.Services, (*store.AuditRepository)(nil))
			if err != nil {
				return nil, fmt.Errorf("resolve audit repository: %w", err)
			}

			return NewPlugin(repo)
		}),
	}
}
