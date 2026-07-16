// Package task owns the platform Task Runtime's persisted state-machine facts.
package task

import (
	"database/sql"
	"fmt"

	"graft/server/internal/module"
	taskstore "graft/server/modules/task/store"
)

// moduleID 是任务模块在编译期模块注册表中的稳定标识，必须与依赖声明和迁移目录一致。
const moduleID = "task"

// NewModuleSpec 返回任务模块的规格定义。
//
// 规格声明任务模块的稳定标识、依赖模块和迁移路径；构建器从服务容器解析 SQL 数据库，并在依赖缺失或仓储创建失败时返回错误。
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
