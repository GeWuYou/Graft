package rbac

import (
	"database/sql"
	"fmt"

	"graft/server/internal/plugin"
	"graft/server/plugins/rbac/storeent"
)

const (
	pluginID      = "rbac"
	pluginVersion = "0.1.0"
)

var pluginDependencies = []string{"user"}

// NewDescriptor exposes the RBAC module's stable compile-time metadata and builder under historical plugin naming.
func NewDescriptor() plugin.Descriptor {
	return plugin.Descriptor{
		ID:            pluginID,
		PluginVersion: pluginVersion,
		Dependencies:  append([]string(nil), pluginDependencies...),
		MigrationPath: []string{"plugins/rbac/migrations"},
		Builder: plugin.BuilderFunc(func(ctx plugin.BuildContext) (plugin.Plugin, error) {
			sqlDB, err := plugin.ResolveService[*sql.DB](ctx.Services, (*sql.DB)(nil))
			if err != nil {
				return nil, fmt.Errorf("resolve sql db: %w", err)
			}
			repo, err := storeent.NewRepository(sqlDB)
			if err != nil {
				return nil, fmt.Errorf("build rbac repository: %w", err)
			}

			return NewPlugin(repo), nil
		}),
	}
}
