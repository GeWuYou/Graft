package moduleapi

import "context"

// PermissionScope describes the effective resource extent of an RBAC permission.
type PermissionScope string

const (
	// PermissionScopeNone 表示主体不拥有目标权限。
	PermissionScopeNone PermissionScope = "none"
	// PermissionScopeOwned 表示主体只能访问自己创建的资源。
	PermissionScopeOwned PermissionScope = "owned"
	// PermissionScopeAll 表示主体可以访问该权限覆盖的全部资源。
	PermissionScopeAll PermissionScope = "all"
)

// PermissionScopeResolver exposes the effective RBAC binding scope without
// exposing role membership or persistence internals to resource modules.
type PermissionScopeResolver interface {
	ResolvePermissionScope(ctx context.Context, userID uint64, permission string) (PermissionScope, error)
}

// PermissionSeed 描述 RBAC 初始化或对齐权限点时需要的最小稳定元数据。
type PermissionSeed struct {
	Code           string
	Display        string
	DisplayKey     string
	Description    string
	DescriptionKey string
	Module         string
}

// RoleSummary 描述跨模块可读的最小角色摘要。
type RoleSummary struct {
	ID      uint64
	Name    string
	Display string
}

// RBACAccessService 暴露跨模块可读的最小 RBAC 快照能力。
type RBACAccessService interface {
	ListRoleNamesByUserID(ctx context.Context, userID uint64) ([]string, error)
	ListPermissionCodesByUserID(ctx context.Context, userID uint64) ([]string, error)
	ListUserIDsByPermissionCode(ctx context.Context, permissionCode string) ([]uint64, error)
	// ListUserIDsByRoleID 返回当前绑定到有效角色的稳定用户 ID，有序且不重复。
	ListUserIDsByRoleID(ctx context.Context, roleID uint64) ([]uint64, error)
	ListRoleSummariesByUserIDs(ctx context.Context, userIDs []uint64) (map[uint64][]RoleSummary, error)
}

// RBACBootstrapService 暴露默认管理员访问基线的最小幂等引导能力。
type RBACBootstrapService interface {
	EnsureDefaultAdminAccess(ctx context.Context, userID uint64, permissions []PermissionSeed) error
}

// SecurityPosture 包含由 RBAC 模块拥有的安全态势计数器。
type SecurityPosture struct {
	TotalUsers           int `json:"total_users"`
	DisabledUsers        int `json:"disabled_users"`
	RoleCount            int `json:"role_count"`
	BuiltinRoleCount     int `json:"builtin_role_count"`
	CustomRoleCount      int `json:"custom_role_count"`
	PermissionCount      int `json:"permission_count"`
	RoleAssignmentCount  int `json:"role_assignment_count"`
	UnassignedUserCount  int `json:"unassigned_user_count"`
	EmptyCustomRoleCount int `json:"empty_custom_role_count"`
}

// RBACSecurityPostureService 暴露聚合授权态势，但不泄漏仓储实现。
type RBACSecurityPostureService interface {
	ReadSecurityPosture(ctx context.Context) (SecurityPosture, error)
}
