package backup

import (
	"database/sql"
	"fmt"

	"graft/server/internal/config"
	"graft/server/internal/module"
	"graft/server/modules/backup/store"
)

const moduleID = "platform-backup"

// NewModuleSpec 返回 platform-backup 的编译期模块描述符。
func NewModuleSpec() module.Spec {
	return module.Spec{
		ID:            moduleID,
		Dependencies:  []string{"user", "rbac", "task"},
		MigrationPath: []string{"modules/backup/migrations"},
		Builder: module.BuilderFunc(func(ctx module.BuildContext) (module.Module, error) {
			db, err := module.ResolveService[*sql.DB](ctx.Services, (*sql.DB)(nil))
			if err != nil {
				return nil, fmt.Errorf("resolve sql db: %w", err)
			}
			repository, err := store.NewSQLRepository(db)
			if err != nil {
				return nil, fmt.Errorf("build backup repository: %w", err)
			}
			runtimeConfig, err := module.ResolveService[*config.Config](ctx.Services, (*config.Config)(nil))
			if err != nil {
				return nil, fmt.Errorf("resolve runtime config: %w", err)
			}
			writer, err := newFileArtifactWriter(runtimeConfig)
			if err != nil {
				return nil, fmt.Errorf("build backup artifact writer: %w", err)
			}
			service := NewService(repository)
			service.setArtifactWriter(writer)
			return NewModule(service), nil
		}),
	}
}
