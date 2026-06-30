package project

import (
	"database/sql"
	"fmt"

	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	projectstore "graft/server/modules/project/store"
)

const moduleID = "project"

// 它声明模块 ID、依赖项和迁移路径，并提供在运行时构建模块实例的入口。
func NewModuleSpec() module.Spec {
	return module.Spec{
		ID:            moduleID,
		Dependencies:  []string{"user", "auth", "rbac"},
		MigrationPath: []string{"modules/project/migrations"},
		Builder: module.BuilderFunc(func(ctx module.BuildContext) (module.Module, error) {
			sqlDB, err := module.ResolveService[*sql.DB](ctx.Services, (*sql.DB)(nil))
			if err != nil {
				return nil, fmt.Errorf("resolve sql db: %w", err)
			}
			repository, err := projectstore.NewSQLRepository(sqlDB)
			if err != nil {
				return nil, fmt.Errorf("build project repository: %w", err)
			}
			var options []ServiceOption
			runtimeReader, runtimeErr := module.ResolveService[moduleapi.ContainerProjectRuntimeReader](ctx.Services, (*moduleapi.ContainerProjectRuntimeReader)(nil))
			if runtimeErr == nil && runtimeReader != nil {
				options = append(options, WithRuntimeReader(runtimeReader))
			}
			configResolver, configErr := module.ResolveService[moduleapi.SystemConfigResolver](ctx.Services, (*moduleapi.SystemConfigResolver)(nil))
			if configErr == nil && configResolver != nil {
				options = append(options, WithSystemConfigResolver(configResolver))
			}
			service, err := NewService(repository, options...)
			if err != nil {
				return nil, fmt.Errorf("build project service: %w", err)
			}
			return NewModule(service), nil
		}),
	}
}
