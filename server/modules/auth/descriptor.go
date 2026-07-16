package auth

import (
	"database/sql"
	"fmt"

	"graft/server/internal/module"
	"graft/server/modules/auth/storeent"
)

const (
	moduleID = "auth"
)

// NewModuleSpec 声明 auth 的模块依赖、迁移入口和持久层构建边界；运行时客户端由 core 提供的 SQL 服务创建。
func NewModuleSpec() module.Spec {
	return module.Spec{
		ID:            moduleID,
		Dependencies:  []string{"user"},
		MigrationPath: []string{"modules/auth/migrations"},
		Builder: module.BuilderFunc(func(ctx module.BuildContext) (module.Module, error) {
			sqlDB, err := module.ResolveService[*sql.DB](ctx.Services, (*sql.DB)(nil))
			if err != nil {
				return nil, fmt.Errorf("resolve sql db: %w", err)
			}
			client, err := storeent.NewClient(sqlDB)
			if err != nil {
				return nil, fmt.Errorf("create auth persistence client: %w", err)
			}
			return NewModule(client), nil
		}),
	}
}
