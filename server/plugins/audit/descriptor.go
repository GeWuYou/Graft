package audit

import (
	"database/sql"
	"fmt"

	auditcore "graft/server/internal/audit"
	"graft/server/internal/drilldown"
	"graft/server/internal/plugin"
	"graft/server/plugins/audit/storeent"
)

const (
	pluginID      = "audit"
	pluginVersion = "0.1.0"
)

var pluginDependencies = []string{"user", "rbac"}

// NewDescriptor exposes the audit module's stable compile-time metadata and builder under historical plugin naming.
func NewDescriptor() plugin.Descriptor {
	return plugin.Descriptor{
		ID:            pluginID,
		PluginVersion: pluginVersion,
		Dependencies:  append([]string(nil), pluginDependencies...),
		MigrationPath: []string{"plugins/audit/migrations"},
		Builder: plugin.BuilderFunc(func(ctx plugin.BuildContext) (plugin.Plugin, error) {
			sqlDB, err := plugin.ResolveService[*sql.DB](ctx.Services, (*sql.DB)(nil))
			if err != nil {
				return nil, fmt.Errorf("resolve sql db: %w", err)
			}
			repo, err := storeent.NewRepository(sqlDB, nil)
			if err != nil {
				return nil, fmt.Errorf("build audit repository: %w", err)
			}

			drilldownRepo, err := drilldown.NewRepository(sqlDB)
			if err != nil {
				return nil, fmt.Errorf("build drilldown repository: %w", err)
			}
			drilldownService, err := drilldown.NewService[auditcore.ListQuery, auditcore.ListQuery](drilldownRepo, newAuditScopeResolver())
			if err != nil {
				return nil, fmt.Errorf("build drilldown service: %w", err)
			}

			return NewPluginWithDrilldown(repo, drilldownService)
		}),
	}
}
