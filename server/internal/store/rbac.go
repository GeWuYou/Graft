package store

import (
	"context"
	"time"
)

// Role 表示 RBAC 角色的稳定持久化 DTO。
type Role struct {
	ID          uint64
	Name        string
	Display     string
	Description *string
	Builtin     bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Permission 表示 RBAC 权限点的稳定持久化 DTO。
type Permission struct {
	ID          uint64
	Code        string
	Display     string
	Description *string
	Category    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// EnsureRoleInput 描述一次最小角色存在性保障所需的输入。
type EnsureRoleInput struct {
	Name        string
	Display     string
	Description *string
	Builtin     bool
}

// EnsurePermissionInput 描述一次最小权限存在性保障所需的输入。
type EnsurePermissionInput struct {
	Code        string
	Display     string
	Description *string
	Category    string
}

// AssignPermissionsToRoleInput 描述一次角色权限最小绑定所需的输入。
type AssignPermissionsToRoleInput struct {
	RoleID        uint64
	PermissionIDs []uint64
}

// AssignRoleToUserInput 描述一次用户角色最小绑定所需的输入。
type AssignRoleToUserInput struct {
	UserID uint64
	RoleID uint64
}

// RBACRepository 暴露未来 RBAC 插件所需的最小持久化查询集合。
//
// 该接口当前只承载角色和权限解析路径，后续如需管理类写操作，应在真实插件需求出现时再收敛扩展。
type RBACRepository interface {
	// EnsureRole 幂等确保目标角色存在。
	EnsureRole(ctx context.Context, input EnsureRoleInput) (Role, error)

	// EnsurePermission 幂等确保目标权限存在。
	EnsurePermission(ctx context.Context, input EnsurePermissionInput) (Permission, error)

	// AssignPermissionsToRole 幂等把一组权限绑定到角色。
	AssignPermissionsToRole(ctx context.Context, input AssignPermissionsToRoleInput) error

	// AssignRoleToUser 幂等把目标角色绑定到用户。
	AssignRoleToUser(ctx context.Context, input AssignRoleToUserInput) error

	// ListRolesByUserID 返回指定用户当前绑定的全部角色。
	ListRolesByUserID(ctx context.Context, userID uint64) ([]Role, error)

	// ListRoles 返回当前稳定排序下的角色快照。
	ListRoles(ctx context.Context) ([]Role, error)

	// ListPermissionsByUserID 返回指定用户经由角色解析得到的全部权限点。
	ListPermissionsByUserID(ctx context.Context, userID uint64) ([]Permission, error)

	// ListPermissions 返回当前稳定排序下的权限快照。
	ListPermissions(ctx context.Context) ([]Permission, error)
}
