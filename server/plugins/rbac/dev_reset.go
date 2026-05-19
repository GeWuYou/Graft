package rbac

import (
	"fmt"

	"graft/server/internal/ent"
	"graft/server/internal/pluginapi"
	rbacstore "graft/server/plugins/rbac/store"
	"graft/server/plugins/rbac/storeent"
)

// RepositoryForReset narrows the dev-reset helper to the plugin-owned RBAC boundary.
type RepositoryForReset = rbacstore.Repository

// NewRepositoryForReset exposes the RBAC plugin's dev-reset repository boundary.
func NewRepositoryForReset(client *ent.Client) (RepositoryForReset, error) {
	repo, err := storeent.NewRepository(client)
	if err != nil {
		return nil, fmt.Errorf("build rbac reset repository: %w", err)
	}

	return repo, nil
}

// NewBootstrapServiceForReset exposes the dev-reset helper on the plugin-local
// RBAC repository contract; composition roots adapt transitional dependencies
// before crossing this boundary.
func NewBootstrapServiceForReset(repo rbacstore.Repository) pluginapi.RBACBootstrapService {
	return NewBootstrapService(repo)
}
