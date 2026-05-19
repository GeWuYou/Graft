package pluginapi

import "context"

// PermissionSeed 描述 RBAC 初始化或对齐权限点时需要的最小稳定元数据。
type PermissionSeed struct {
	Code        string
	Display     string
	Description string
	Category    string
}

// RBACAccessService 暴露跨插件可读的最小 RBAC 快照能力。
type RBACAccessService interface {
	ListRoleNamesByUserID(ctx context.Context, userID uint64) ([]string, error)
	ListPermissionCodesByUserID(ctx context.Context, userID uint64) ([]string, error)
}

// RBACBootstrapService 暴露默认管理员访问基线的最小幂等引导能力。
type RBACBootstrapService interface {
	EnsureDefaultAdminAccess(ctx context.Context, userID uint64, permissions []PermissionSeed) error
}
