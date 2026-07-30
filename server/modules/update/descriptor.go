package update

import (
	"database/sql"
	"fmt"

	"graft/server/internal/module"
)

// NewModuleSpec 返回 platform-update 的编译期模块描述符。
func NewModuleSpec() module.Spec {
	return module.Spec{ID: moduleID, Dependencies: []string{"user", "rbac", "task", "platform-backup", "deployment"}, MigrationPath: []string{"modules/update/migrations"}, Builder: module.BuilderFunc(func(ctx module.BuildContext) (module.Module, error) {
		db, err := module.ResolveService[*sql.DB](ctx.Services, (*sql.DB)(nil))
		if err != nil {
			return nil, fmt.Errorf("resolve sql db: %w", err)
		}
		operations, err := newSQLOperationStore(db)
		if err != nil {
			return nil, err
		}
		diagnostics, err := newSQLFailureDiagnosticStore(db)
		if err != nil {
			return nil, err
		}
		cache, err := newSQLDiscoveryCache(db)
		if err != nil {
			return nil, err
		}
		return NewModule(operations, diagnostics, cache), nil
	})}
}
