// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package notification

import (
	"database/sql"
	"fmt"

	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	notificationstore "graft/server/modules/notification/store"
)

const moduleID = "notification"

// NewModuleSpec exposes the notification module's stable compile-time metadata and builder.
func NewModuleSpec() module.Spec {
	return module.Spec{
		ID:            moduleID,
		Dependencies:  []string{"user", "rbac"},
		MigrationPath: []string{"modules/notification/migrations"},
		Builder: module.BuilderFunc(func(ctx module.BuildContext) (module.Module, error) {
			sqlDB, err := module.ResolveService[*sql.DB](ctx.Services, (*sql.DB)(nil))
			if err != nil {
				return nil, fmt.Errorf("resolve sql db: %w", err)
			}
			repository, err := notificationstore.NewSQLRepository(sqlDB)
			if err != nil {
				return nil, fmt.Errorf("build notification repository: %w", err)
			}
			service, err := NewService(repository)
			if err != nil {
				return nil, fmt.Errorf("build notification service: %w", err)
			}
			rbac, err := module.ResolveService[moduleapi.RBACAccessService](ctx.Services, (*moduleapi.RBACAccessService)(nil))
			if err != nil {
				return nil, fmt.Errorf("resolve rbac access service: %w", err)
			}
			publisher, err := NewPublisher(repository, rbac)
			if err != nil {
				return nil, fmt.Errorf("build notification publisher: %w", err)
			}
			return NewModule(service, publisher), nil
		}),
	}
}
