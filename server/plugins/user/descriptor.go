package user

import (
	"database/sql"
	"fmt"

	"graft/server/internal/plugin"
	"graft/server/plugins/user/storeent"
)

const (
	pluginID      = "user"
	pluginVersion = "0.1.0"
)

// NewDescriptor exposes the user plugin's stable metadata and builder.
func NewDescriptor() plugin.Descriptor {
	return plugin.Descriptor{
		ID:            pluginID,
		PluginVersion: pluginVersion,
		Dependencies:  nil,
		MigrationPath: []string{"plugins/user/migrations"},
		Builder: plugin.BuilderFunc(func(ctx plugin.BuildContext) (plugin.Plugin, error) {
			sqlDB, err := plugin.ResolveService[*sql.DB](ctx.Services, (*sql.DB)(nil))
			if err != nil {
				return nil, fmt.Errorf("resolve sql db: %w", err)
			}
			storeRuntime, err := storeent.NewRuntime(sqlDB)
			if err != nil {
				return nil, fmt.Errorf("build user storeent runtime: %w", err)
			}
			userRepo, err := storeRuntime.NewUserRepository()
			if err != nil {
				return nil, fmt.Errorf("build user storeent repository: %w", err)
			}
			authRepo, err := storeRuntime.NewAuthRepository()
			if err != nil {
				return nil, fmt.Errorf("build user auth storeent repository: %w", err)
			}

			return NewPlugin(userRepo, authRepo), nil
		}),
	}
}
