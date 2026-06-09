// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package user

import (
	"database/sql"
	"fmt"

	"graft/server/internal/module"
	"graft/server/modules/user/storeent"

	"go.uber.org/zap"
)

const (
	moduleID = "user"
)

// NewModuleSpec exposes the user module's stable compile-time metadata and builder.
func NewModuleSpec() module.Spec {
	return module.Spec{
		ID:            moduleID,
		Dependencies:  nil,
		MigrationPath: []string{"modules/user/migrations"},
		Builder: module.BuilderFunc(func(ctx module.BuildContext) (module.Module, error) {
			sqlDB, err := module.ResolveService[*sql.DB](ctx.Services, (*sql.DB)(nil))
			if err != nil {
				return nil, fmt.Errorf("resolve sql db: %w", err)
			}
			runtimeLogger, err := module.ResolveService[*zap.Logger](ctx.Services, (*zap.Logger)(nil))
			if err != nil {
				return nil, fmt.Errorf("resolve runtime logger: %w", err)
			}
			storeRuntime, err := storeent.NewRuntime(sqlDB, runtimeLogger)
			if err != nil {
				return nil, fmt.Errorf("build user storeent runtime: %w", err)
			}
			userRepo, err := storeRuntime.NewUserRepository()
			if err != nil {
				return nil, fmt.Errorf("build user storeent repository: %w", err)
			}
			authRepo, err := storeRuntime.NewAuthRepository()
			if err != nil {
				return nil, fmt.Errorf("build user auth repository: %w", err)
			}

			return NewModule(userRepo, authRepo), nil
		}),
	}
}
