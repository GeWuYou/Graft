// Package runtimetarget 负责持久化运行时连接身份和发现事实。
package runtimetarget

import (
	"database/sql"
	"fmt"

	"graft/server/internal/config"
	"graft/server/internal/module"
	credentialvaultcontract "graft/server/modules/credential-vault/contract"
	store "graft/server/modules/runtime-target/store"
)

const moduleID = "runtime-target"

// NewModuleSpec 返回 runtime-target 模块的编译期描述符。
func NewModuleSpec() module.Spec {
	return module.Spec{
		ID:            moduleID,
		Dependencies:  []string{"user", "auth", "rbac", "saved-view", credentialvaultcontract.ModuleID},
		MigrationPath: []string{"modules/runtime-target/migrations"},
		Builder: module.BuilderFunc(func(ctx module.BuildContext) (module.Module, error) {
			db, err := module.ResolveService[*sql.DB](ctx.Services, (*sql.DB)(nil))
			if err != nil {
				return nil, fmt.Errorf("resolve sql db: %w", err)
			}
			runtimeConfig, err := module.ResolveService[*config.Config](ctx.Services, (*config.Config)(nil))
			if err != nil {
				return nil, fmt.Errorf("resolve runtime config: %w", err)
			}
			enrollmentSecurity := runtimeConfig.EnrollmentSecurity
			pepper, err := config.NewEnrollmentPepperProvider(enrollmentSecurity)
			if err != nil {
				return nil, fmt.Errorf("build enrollment pepper provider: %w", err)
			}
			return NewModule(store.NewSQLRepository(db), pepper), nil
		}),
	}
}
