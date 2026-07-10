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
// NewIdentityProviderForDevelopmentReset 为开发环境重置命令创建用户身份提供者。
// 如果用户 profile runtime 或用户仓储创建失败，则返回相应错误。
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
// DefaultAdminPermissionItems 提供开发环境默认管理员重置流程所需的用户权限项。
// 返回用户模块的权限项。
func DefaultAdminPermissionItems() []permission.Item { return userPermissionItems(moduleID) }
