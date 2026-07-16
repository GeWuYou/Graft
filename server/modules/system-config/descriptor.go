package systemconfig

import (
	"database/sql"
	"fmt"

	"graft/server/internal/cachex"
	"graft/server/internal/configregistry"
	"graft/server/internal/module"
	"graft/server/modules/system-config/storeent"
)

// NewModuleSpec 返回系统配置模块的依赖、迁移路径和服务构建器；覆盖值持久化与快照缓存均在模块内装配。
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
			cacheManager, err := module.ResolveService[*cachex.Manager](ctx.Services, (*cachex.Manager)(nil))
			if err != nil {
				return nil, fmt.Errorf("resolve cache manager: %w", err)
			}
			snapshotCache, err := cacheManager.NewCache(systemConfigSnapshotCacheName)
			if err != nil {
				return nil, fmt.Errorf("build system config snapshot cache: %w", err)
			}
			repo, err := storeent.NewRepository(sqlDB)
			if err != nil {
				return nil, fmt.Errorf("build system config repository: %w", err)
			}
			service, err := NewService(registry, repo, ServiceOptions{
				Cache: snapshotCache,
			})
			if err != nil {
				return nil, fmt.Errorf("build system config service: %w", err)
			}
			return NewModule(service)
		}),
	}
}
