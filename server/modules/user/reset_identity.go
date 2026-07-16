package user

import (
	"database/sql"
	"fmt"

	"go.uber.org/zap"

	"graft/server/internal/moduleapi"
	"graft/server/internal/permission"
	"graft/server/modules/user/storeent"
)

// NewIdentityProviderForDevelopmentReset 为开发环境重置命令创建用户身份提供者；失败时返回 profile runtime 或仓储初始化错误。
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

// DefaultAdminPermissionItems 提供开发环境默认管理员重置流程所需的 user 模块权限项。
// 返回用户模块的权限项。
func DefaultAdminPermissionItems() []permission.Item { return userPermissionItems(moduleID) }
