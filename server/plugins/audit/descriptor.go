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
	moduleID = "audit"
)

// NewModuleSpec exposes the audit module's stable compile-time metadata and builder.
func NewModuleSpec() plugin.ModuleSpec {
	return plugin.ModuleSpec{
		ID:            moduleID,
		Dependencies:  []string{"user", "rbac"},
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
