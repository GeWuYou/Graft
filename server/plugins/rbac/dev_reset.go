package rbac

import (
	"graft/server/internal/pluginapi"
	"graft/server/internal/store"
	"graft/server/plugins/rbac/storeadapter"
)

// NewBootstrapServiceForReset keeps the transitional internal-store adapter
// inside the RBAC plugin boundary for the dev-only reset-admin CLI flow.
func NewBootstrapServiceForReset(repo store.RBACRepository) pluginapi.RBACBootstrapService {
	return NewBootstrapService(storeadapter.NewInternalRepositoryAdapter(repo))
}
