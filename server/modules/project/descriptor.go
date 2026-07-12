package project

import (
	"database/sql"
	"fmt"

	"graft/server/internal/module"
	projectstore "graft/server/modules/project/store"
)

const moduleID = "project"

// NewModuleSpec 返回项目模块的规范，包括其依赖项、迁移路径和构建器。
func NewModuleSpec() module.Spec {
	return module.Spec{
		ID:            moduleID,
		Dependencies:  []string{"user", "auth", "rbac", "container", "runtime-target", "saved-view", "system-config", "task"},
		MigrationPath: []string{"modules/project/migrations"},
		Builder: module.BuilderFunc(func(ctx module.BuildContext) (module.Module, error) {
			sqlDB, err := module.ResolveService[*sql.DB](ctx.Services, (*sql.DB)(nil))
			if err != nil {
				return nil, fmt.Errorf("resolve sql db: %w", err)
			}
			repository, err := projectstore.NewSQLRepository(sqlDB)
			if err != nil {
				return nil, fmt.Errorf("build project repository: %w", err)
			}
			service, err := NewService(repository)
			if err != nil {
				return nil, fmt.Errorf("build project service: %w", err)
			}
			return NewModule(service), nil
		}),
	}
}
