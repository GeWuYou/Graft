package rbac

import (
	"database/sql"
	"fmt"

	"graft/server/internal/module"
	"graft/server/modules/rbac/storeent"
)

const (
	moduleID = "rbac"
)

// NewModuleSpec 返回 RBAC 模块的稳定元数据及其构建器配置，并声明对 auth 和 user 模块的依赖。
func NewModuleSpec() module.Spec {
	return module.Spec{
		ID:            moduleID,
		Dependencies:  []string{"auth", "user", "saved-view"},
		MigrationPath: []string{"modules/rbac/migrations"},
		Builder: module.BuilderFunc(func(ctx module.BuildContext) (module.Module, error) {
			sqlDB, err := module.ResolveService[*sql.DB](ctx.Services, (*sql.DB)(nil))
			if err != nil {
				return nil, fmt.Errorf("resolve sql db: %w", err)
			}
			repo, err := storeent.NewRepository(sqlDB)
			if err != nil {
				return nil, fmt.Errorf("build rbac repository: %w", err)
			}

			return NewModule(repo), nil
		}),
	}
}
