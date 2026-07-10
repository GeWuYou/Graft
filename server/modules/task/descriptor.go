// Package task owns the platform Task Runtime's persisted state-machine facts.
package task

import (
	"database/sql"
	"fmt"

	"graft/server/internal/module"
	taskstore "graft/server/modules/task/store"
)

const moduleID = "task"

// NewModuleSpec 返回任务模块的规格定义，包括模块标识、依赖模块、迁移路径以及基于 SQL 数据库构建模块实例的构建器。
func NewModuleSpec() module.Spec {
	return module.Spec{
		ID:            moduleID,
		Dependencies:  []string{"user", "rbac"},
		MigrationPath: []string{"modules/task/migrations"},
		Builder: module.BuilderFunc(func(ctx module.BuildContext) (module.Module, error) {
			sqlDB, err := module.ResolveService[*sql.DB](ctx.Services, (*sql.DB)(nil))
			if err != nil {
				return nil, fmt.Errorf("resolve sql db: %w", err)
			}
			repository, err := taskstore.NewSQLRepository(sqlDB, taskstore.SQLDialectPostgres)
			if err != nil {
				return nil, fmt.Errorf("build task repository: %w", err)
			}
			return NewModule(repository), nil
		}),
	}
}
