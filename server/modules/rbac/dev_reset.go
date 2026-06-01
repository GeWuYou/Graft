package rbac

import (
	"database/sql"
	"fmt"

	"graft/server/internal/pluginapi"
	rbacstore "graft/server/modules/rbac/store"
	"graft/server/modules/rbac/storeent"
)

// RepositoryForReset 将 dev-reset helper 收敛到 RBAC 插件自有的 repository 边界。
type RepositoryForReset = rbacstore.Repository

// NewRepositoryForReset 暴露 RBAC 插件用于 dev-reset 的 repository 边界。
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

// NewBootstrapServiceForReset 通过插件内 RBAC repository contract 暴露 dev-reset helper；
// 组合根在跨过该边界前负责适配过渡期依赖。
func NewBootstrapServiceForReset(repo rbacstore.Repository) pluginapi.RBACBootstrapService {
	return NewBootstrapService(repo)
}
