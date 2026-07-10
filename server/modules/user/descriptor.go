package user

import (
	"database/sql"
	"fmt"

	"graft/server/internal/module"
	"graft/server/modules/user/storeent"

	"go.uber.org/zap"
)

const (
	moduleID = "user"
)

// NewModuleSpec 构建并返回用户模块的稳定元数据及其运行时构建器。
func NewModuleSpec() module.Spec {
	return module.Spec{
		ID:            moduleID,
		Dependencies:  nil,
		MigrationPath: []string{"modules/user/migrations"},
		Builder: module.BuilderFunc(func(ctx module.BuildContext) (module.Module, error) {
			sqlDB, err := module.ResolveService[*sql.DB](ctx.Services, (*sql.DB)(nil))
			if err != nil {
				return nil, fmt.Errorf("resolve sql db: %w", err)
			}
			runtimeLogger, err := module.ResolveService[*zap.Logger](ctx.Services, (*zap.Logger)(nil))
			if err != nil {
				return nil, fmt.Errorf("resolve runtime logger: %w", err)
			}
			storeRuntime, err := storeent.NewRuntime(sqlDB, runtimeLogger)
			if err != nil {
				return nil, fmt.Errorf("build user storeent runtime: %w", err)
			}
			userRepo, err := storeRuntime.NewUserRepository()
			if err != nil {
				return nil, fmt.Errorf("build user storeent repository: %w", err)
			}
			return NewModule(userRepo), nil
		}),
	}
}
