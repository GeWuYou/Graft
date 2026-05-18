package rbac

import (
	"fmt"

	"graft/server/internal/plugin"
	"graft/server/internal/store"
	"graft/server/plugins/rbac/storeadapter"
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
		Builder: plugin.BuilderFunc(func(ctx plugin.BuildContext) (plugin.Plugin, error) {
			repo, err := plugin.ResolveService[store.RBACRepository](ctx.Services, (*store.RBACRepository)(nil))
			if err != nil {
				return nil, fmt.Errorf("resolve rbac repository: %w", err)
			}

			return NewPlugin(storeadapter.NewInternalRepositoryAdapter(repo)), nil
		}),
	}
}
