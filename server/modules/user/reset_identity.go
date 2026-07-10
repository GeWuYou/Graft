package user

import (
	"database/sql"
	"fmt"

	"go.uber.org/zap"

	"graft/server/internal/moduleapi"
	"graft/server/internal/permission"
	"graft/server/modules/user/storeent"
)

// NewIdentityProviderForDevelopmentReset builds the narrow profile capability
// required by auth's development-only reset command.
func NewIdentityProviderForDevelopmentReset(sqlDB *sql.DB) (moduleapi.UserIdentityProvider, error) {
	runtime, err := storeent.NewRuntime(sqlDB, zap.NewNop())
	if err != nil {
		return nil, fmt.Errorf("build user profile runtime: %w", err)
	}
	repository, err := runtime.NewUserRepository()
	if err != nil {
		return nil, fmt.Errorf("build user profile repository: %w", err)
	}
	return userIdentityProvider{users: repository}, nil
}

// DefaultAdminPermissionItems returns the user-owned permissions required by
// the development-only default-admin reset path.
func DefaultAdminPermissionItems() []permission.Item { return userPermissionItems(moduleID) }
