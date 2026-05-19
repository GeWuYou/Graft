package rbac

import (
	"graft/server/internal/pluginapi"
	rbacstore "graft/server/plugins/rbac/store"
)

// NewBootstrapServiceForReset exposes the dev-reset helper on the plugin-local
// RBAC repository contract; composition roots adapt transitional dependencies
// before crossing this boundary.
func NewBootstrapServiceForReset(repo rbacstore.Repository) pluginapi.RBACBootstrapService {
	return NewBootstrapService(repo)
}
