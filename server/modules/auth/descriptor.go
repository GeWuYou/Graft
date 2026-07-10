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

// NewModuleSpec 返回描述 auth 模块元数据、依赖关系、迁移路径及构建方式的模块规格。
// 构建模块时解析 SQL 数据库服务并创建持久层客户端。
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
				return nil, err
			}
			return NewModule(client), nil
		}),
	}
}
