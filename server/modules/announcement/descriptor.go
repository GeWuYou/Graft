// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package announcement

import (
	"database/sql"
	"fmt"

	"graft/server/internal/module"
	announcementstore "graft/server/modules/announcement/store"
)

const moduleID = "announcement"

// NewModuleSpec exposes the announcement module's stable compile-time metadata and builder.
func NewModuleSpec() module.Spec {
	return module.Spec{
		ID:            moduleID,
		Dependencies:  []string{"user", "rbac"},
		MigrationPath: []string{"modules/announcement/migrations"},
		Builder: module.BuilderFunc(func(ctx module.BuildContext) (module.Module, error) {
			sqlDB, err := module.ResolveService[*sql.DB](ctx.Services, (*sql.DB)(nil))
			if err != nil {
				return nil, fmt.Errorf("resolve sql db: %w", err)
			}
			repository, err := announcementstore.NewSQLRepository(sqlDB)
			if err != nil {
				return nil, fmt.Errorf("build announcement repository: %w", err)
			}
			service, err := NewService(repository)
			if err != nil {
				return nil, fmt.Errorf("build announcement service: %w", err)
			}
			return NewModule(service), nil
		}),
	}
}
