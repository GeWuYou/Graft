package store

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrRoleNotFound 表示请求的角色不存在。
	ErrRoleNotFound = errors.New("role not found")

	// ErrPermissionNotFound 表示请求引用的权限不存在。
	ErrPermissionNotFound = errors.New("permission not found")

	// ErrRoleNameConflict 表示目标角色名称已被其它角色占用。
	ErrRoleNameConflict = errors.New("role name conflict")
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

// RolePermissionBinding 表示角色当前绑定的一条稳定权限关系。
type RolePermissionBinding struct {
	RoleID       uint64
	PermissionID uint64
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

// CreateRoleInput 描述一次显式角色创建所需的输入。
type CreateRoleInput struct {
	Name        string
	Display     string
	Description *string
	Builtin     bool
}

// UpdateRoleInput 描述一次显式角色更新所需的输入。
type UpdateRoleInput struct {
	ID          uint64
	Name        string
	Display     string
	Description *string
}

// AssignPermissionsToRoleInput 描述一次角色权限最小绑定所需的输入。
type AssignPermissionsToRoleInput struct {
	RoleID        uint64
	PermissionIDs []uint64
}

// ReplacePermissionsForRoleInput 描述一次角色权限覆盖写入所需的输入。
type ReplacePermissionsForRoleInput struct {
	RoleID        uint64
	PermissionIDs []uint64
}

// AssignRoleToUserInput 描述一次用户角色最小绑定所需的输入。
type AssignRoleToUserInput struct {
	UserID uint64
	RoleID uint64
}

// ReplaceRolesForUserInput 描述一次用户角色覆盖写入所需的输入。
type ReplaceRolesForUserInput struct {
	UserID  uint64
	RoleIDs []uint64
}

// RBACRepository 暴露未来 RBAC 插件所需的最小持久化查询集合。
//
// 该接口当前只承载角色和权限解析路径，后续如需管理类写操作，应在真实插件需求出现时再收敛扩展。
type RBACRepository interface {
	// EnsureRole 幂等确保目标角色存在。
	EnsureRole(ctx context.Context, input EnsureRoleInput) (Role, error)

	// EnsurePermission 幂等确保目标权限存在。
	EnsurePermission(ctx context.Context, input EnsurePermissionInput) (Permission, error)

	// CreateRole 显式创建一个角色，命名冲突时返回 ErrRoleNameConflict。
	CreateRole(ctx context.Context, input CreateRoleInput) (Role, error)

	// UpdateRole 按稳定 ID 更新一个角色，未命中时返回 ErrRoleNotFound。
	UpdateRole(ctx context.Context, input UpdateRoleInput) (Role, error)

	// AssignPermissionsToRole 幂等把一组权限绑定到角色。
	AssignPermissionsToRole(ctx context.Context, input AssignPermissionsToRoleInput) error

	// ReplacePermissionsForRole 把角色权限覆盖为目标集合。
	ReplacePermissionsForRole(ctx context.Context, input ReplacePermissionsForRoleInput) error

	// AssignRoleToUser 幂等把目标角色绑定到用户。
	AssignRoleToUser(ctx context.Context, input AssignRoleToUserInput) error

	// ReplaceRolesForUser 把用户角色覆盖为目标集合。
	ReplaceRolesForUser(ctx context.Context, input ReplaceRolesForUserInput) error

	// GetRoleByID 按 ID 返回单个角色记录，未命中时返回 ErrRoleNotFound。
	GetRoleByID(ctx context.Context, roleID uint64) (Role, error)

	// ListRolesByUserID 返回指定用户当前绑定的全部角色。
	ListRolesByUserID(ctx context.Context, userID uint64) ([]Role, error)

	// ListRoles 返回当前稳定排序下的角色快照。
	ListRoles(ctx context.Context) ([]Role, error)

	// ListPermissionsByUserID 返回指定用户经由角色解析得到的全部权限点。
	ListPermissionsByUserID(ctx context.Context, userID uint64) ([]Permission, error)

	// ListPermissions 返回当前稳定排序下的权限快照。
	ListPermissions(ctx context.Context) ([]Permission, error)

	// ListRolePermissionBindings 返回指定角色当前绑定的权限关系快照。
	ListRolePermissionBindings(ctx context.Context, roleID uint64) ([]RolePermissionBinding, error)
}
