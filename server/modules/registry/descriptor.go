// Package registry 负责外部 OCI 镜像仓库连接与产物仓库事实，并向 Build 暴露受限的目的地解析能力。
package registry

import (
	"database/sql"
	"fmt"

	"graft/server/internal/module"
	registrycontract "graft/server/modules/registry/contract"
	registrystore "graft/server/modules/registry/store"
)

const moduleID = registrycontract.ModuleID

// NewModuleSpec 声明由基础设施域拥有的 Registry 模块；模块不实现镜像仓库服务，也不向消费者暴露提供方凭据。
func NewModuleSpec() module.Spec {
	return module.Spec{
		ID:            moduleID,
		Dependencies:  []string{"user", "auth", "rbac"},
		MigrationPath: []string{"modules/registry/migrations"},
		Builder: module.BuilderFunc(func(ctx module.BuildContext) (module.Module, error) {
			db, err := module.ResolveService[*sql.DB](ctx.Services, (*sql.DB)(nil))
			if err != nil {
				return nil, fmt.Errorf("resolve sql db: %w", err)
			}
			repository, err := registrystore.NewSQLRepository(db)
			if err != nil {
				return nil, fmt.Errorf("build registry repository: %w", err)
			}
			return NewModule(NewService(repository)), nil
		}),
	}
}
