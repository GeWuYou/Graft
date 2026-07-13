package audit

import (
	"database/sql"
	"fmt"

	"graft/server/internal/drilldown"
	"graft/server/internal/i18n"
	"graft/server/internal/module"
	"graft/server/modules/audit/storeent"
)

const (
	moduleID = "audit"
)

// NewModuleSpec 定义审计模块的标识、依赖、迁移路径及构建器。
// 返回包含审计模块元数据和实例构建逻辑的模块规范。
func NewModuleSpec() module.Spec {
	return module.Spec{
		ID:            moduleID,
		Dependencies:  []string{"user", "rbac", "saved-view"},
		MigrationPath: []string{"modules/audit/migrations"},
		Builder: module.BuilderFunc(func(ctx module.BuildContext) (module.Module, error) {
			sqlDB, err := module.ResolveService[*sql.DB](ctx.Services, (*sql.DB)(nil))
			if err != nil {
				return nil, fmt.Errorf("resolve sql db: %w", err)
			}
			localizer, err := module.ResolveService[*i18n.Service](ctx.Services, (*i18n.Service)(nil))
			if err != nil {
				return nil, fmt.Errorf("resolve i18n service: %w", err)
			}
			repo, err := storeent.NewRepository(sqlDB, localizer, nil)
			if err != nil {
				return nil, fmt.Errorf("build audit repository: %w", err)
			}

			drilldownRepo, err := drilldown.NewRepository(sqlDB)
			if err != nil {
				return nil, fmt.Errorf("build drilldown repository: %w", err)
			}
			drilldownService, err := drilldown.NewService[ListQuery, ListQuery](drilldownRepo, newAuditScopeResolver())
			if err != nil {
				return nil, fmt.Errorf("build drilldown service: %w", err)
			}

			return NewModuleWithDrilldown(repo, drilldownService)
		}),
	}
}
