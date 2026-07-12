// Package runtimetarget owns persisted runtime connection identities and discovery facts.
package runtimetarget

import (
	"database/sql"
	"fmt"

	"graft/server/internal/module"
	store "graft/server/modules/runtime-target/store"
)

const moduleID = "runtime-target"

// NewModuleSpec returns the compile-time runtime-target module descriptor.
func NewModuleSpec() module.Spec {
	return module.Spec{
		ID:            moduleID,
		Dependencies:  []string{"user", "auth", "rbac"},
		MigrationPath: []string{"modules/runtime-target/migrations"},
		Builder: module.BuilderFunc(func(ctx module.BuildContext) (module.Module, error) {
			db, err := module.ResolveService[*sql.DB](ctx.Services, (*sql.DB)(nil))
			if err != nil {
				return nil, fmt.Errorf("resolve sql db: %w", err)
			}
			return NewModule(store.NewSQLRepository(db)), nil
		}),
	}
}
