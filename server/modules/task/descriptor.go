// Package task owns the platform Task Runtime's persisted state-machine facts.
package task

import (
	"database/sql"
	"fmt"

	"graft/server/internal/module"
	taskstore "graft/server/modules/task/store"
)

const moduleID = "task"

// NewModuleSpec exposes the Task Runtime's stable compile-time metadata.
func NewModuleSpec() module.Spec {
	return module.Spec{
		ID:            moduleID,
		MigrationPath: []string{"modules/task/migrations"},
		Builder: module.BuilderFunc(func(ctx module.BuildContext) (module.Module, error) {
			sqlDB, err := module.ResolveService[*sql.DB](ctx.Services, (*sql.DB)(nil))
			if err != nil {
				return nil, fmt.Errorf("resolve sql db: %w", err)
			}
			repository, err := taskstore.NewSQLRepository(sqlDB)
			if err != nil {
				return nil, fmt.Errorf("build task repository: %w", err)
			}
			return NewModule(repository), nil
		}),
	}
}
