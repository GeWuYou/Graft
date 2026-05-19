package audit

import (
	"database/sql"
	"fmt"

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
			sqlDB, err := plugin.ResolveService[*sql.DB](ctx.Services, (*sql.DB)(nil))
			if err != nil {
				return nil, fmt.Errorf("resolve sql db: %w", err)
			}
			repo, err := storeent.NewRepository(sqlDB)
			if err != nil {
				return nil, fmt.Errorf("build audit repository: %w", err)
			}

			return NewPlugin(repo)
		}),
	}
}
