package user

import (
	"database/sql"
	"fmt"

	"graft/server/internal/plugin"
	"graft/server/plugins/user/storeent"

	"go.uber.org/zap"
)

const (
	moduleID = "user"
)

// NewModuleSpec exposes the user module's stable compile-time metadata and builder.
func NewModuleSpec() plugin.ModuleSpec {
	return plugin.ModuleSpec{
		ID:            moduleID,
		Dependencies:  nil,
		MigrationPath: []string{"plugins/user/migrations"},
		Builder: plugin.BuilderFunc(func(ctx plugin.BuildContext) (plugin.Plugin, error) {
			sqlDB, err := plugin.ResolveService[*sql.DB](ctx.Services, (*sql.DB)(nil))
			if err != nil {
				return nil, fmt.Errorf("resolve sql db: %w", err)
			}
			runtimeLogger, err := plugin.ResolveService[*zap.Logger](ctx.Services, (*zap.Logger)(nil))
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
			authRepo, err := storeRuntime.NewAuthRepository()
			if err != nil {
				return nil, fmt.Errorf("build user auth repository: %w", err)
			}

			return NewPlugin(userRepo, authRepo), nil
		}),
	}
}
