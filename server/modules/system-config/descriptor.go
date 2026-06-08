package systemconfig

import (
	"database/sql"
	"fmt"

	"graft/server/internal/configregistry"
	"graft/server/internal/module"
	"graft/server/modules/system-config/storeent"
)

// NewModuleSpec exposes the system-config module's stable compile-time metadata and builder.
func NewModuleSpec() module.Spec {
	return module.Spec{
		ID:            moduleID,
		Dependencies:  []string{"user", "rbac"},
		MigrationPath: []string{"modules/system-config/migrations"},
		Builder: module.BuilderFunc(func(ctx module.BuildContext) (module.Module, error) {
			sqlDB, err := module.ResolveService[*sql.DB](ctx.Services, (*sql.DB)(nil))
			if err != nil {
				return nil, fmt.Errorf("resolve sql db: %w", err)
			}
			registry, err := module.ResolveService[*configregistry.Registry](ctx.Services, (*configregistry.Registry)(nil))
			if err != nil {
				return nil, fmt.Errorf("resolve config registry: %w", err)
			}
			repo, err := storeent.NewRepository(sqlDB)
			if err != nil {
				return nil, fmt.Errorf("build system config repository: %w", err)
			}
			service, err := NewService(registry, repo, nil)
			if err != nil {
				return nil, fmt.Errorf("build system config service: %w", err)
			}
			return NewModule(service)
		}),
	}
}
