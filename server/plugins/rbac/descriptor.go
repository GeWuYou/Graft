package rbac

import (
	"fmt"

	"graft/server/internal/plugin"
	"graft/server/internal/store"
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
			repo, err := plugin.ResolveService[store.RBACRepository](ctx.Services, (*store.RBACRepository)(nil))
			if err != nil {
				return nil, fmt.Errorf("resolve rbac repository: %w", err)
			}

			return NewPlugin(storeadapter.NewInternalRepositoryAdapter(repo)), nil
		}),
	}
}
