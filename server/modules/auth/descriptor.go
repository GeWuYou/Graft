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

// NewModuleSpec exposes the auth module's stable compile-time metadata and builder.
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
