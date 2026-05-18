package audit

import (
	"fmt"

	"graft/server/internal/plugin"
	"graft/server/internal/store"
)

// NewDescriptor exposes the audit plugin's stable metadata and builder.
func NewDescriptor() plugin.Descriptor {
	instance, err := NewPlugin(nil)
	if err != nil {
		instance = &Plugin{}
	}

	return plugin.Descriptor{
		ID:            instance.Name(),
		PluginVersion: instance.Version(),
		Dependencies:  append([]string(nil), instance.DependsOn()...),
		Builder: plugin.BuilderFunc(func(ctx plugin.BuildContext) (plugin.Plugin, error) {
			repo, err := plugin.ResolveService[store.AuditRepository](ctx.Services, (*store.AuditRepository)(nil))
			if err != nil {
				return nil, fmt.Errorf("resolve audit repository: %w", err)
			}

			return NewPlugin(repo)
		}),
	}
}
