// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package rbac

import (
	"database/sql"
	"fmt"

	"graft/server/internal/moduleapi"
	rbacstore "graft/server/modules/rbac/store"
	"graft/server/modules/rbac/storeent"
)

// RepositoryForReset 将 dev-reset helper 收敛到 RBAC 模块自有的 repository 边界。
type RepositoryForReset = rbacstore.Repository

// NewRepositoryForReset 暴露 RBAC 模块用于 dev-reset 的 repository 边界。
func NewRepositoryForReset(sqlDB *sql.DB) (RepositoryForReset, error) {
	if sqlDB == nil {
		return nil, fmt.Errorf("rbac reset repository requires a non-nil sql db")
	}
	repo, err := storeent.NewRepository(sqlDB)
	if err != nil {
		return nil, fmt.Errorf("build rbac reset repository: %w", err)
	}

	return repo, nil
}

// NewBootstrapServiceForReset 通过模块内 RBAC repository contract 暴露 dev-reset helper；
// 组合根在跨过该边界前负责适配过渡期依赖。
func NewBootstrapServiceForReset(repo rbacstore.Repository) moduleapi.RBACBootstrapService {
	return NewBootstrapService(repo)
}
