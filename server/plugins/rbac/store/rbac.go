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

	// ErrInvalidID 表示调用方提供的稳定标识不满足当前仓储契约要求。
	ErrInvalidID = errors.New("invalid id")
)

// Role 表示 RBAC 角色的稳定持久化 DTO。
type Role struct {
	ID              uint64
	Name            string
	Display         string
	Description     *string
	Builtin         bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
	PermissionCount int
	UserCount       int
}

// Permission 表示 RBAC 权限点的稳定持久化 DTO。
type Permission struct {
	ID               uint64
	Code             string
	Display          string
	Description      *string
	Category         string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	RoleBindingCount int
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

// Repository 暴露 RBAC 插件私有持久化能力。
type Repository interface {
	EnsureRole(ctx context.Context, input EnsureRoleInput) (Role, error)
	EnsurePermission(ctx context.Context, input EnsurePermissionInput) (Permission, error)
	CreateRole(ctx context.Context, input CreateRoleInput) (Role, error)
	UpdateRole(ctx context.Context, input UpdateRoleInput) (Role, error)
	AssignPermissionsToRole(ctx context.Context, input AssignPermissionsToRoleInput) error
	ReplacePermissionsForRole(ctx context.Context, input ReplacePermissionsForRoleInput) error
	AssignRoleToUser(ctx context.Context, input AssignRoleToUserInput) error
	ReplaceRolesForUser(ctx context.Context, input ReplaceRolesForUserInput) error
	GetRoleByID(ctx context.Context, roleID uint64) (Role, error)
	ListRolesByUserID(ctx context.Context, userID uint64) ([]Role, error)
	ListRoles(ctx context.Context) ([]Role, error)
	ListPermissionsByUserID(ctx context.Context, userID uint64) ([]Permission, error)
	ListPermissions(ctx context.Context) ([]Permission, error)
	ListRolePermissionBindings(ctx context.Context, roleID uint64) ([]RolePermissionBinding, error)
}
