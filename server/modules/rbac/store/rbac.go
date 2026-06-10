// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

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

	// ErrRoleBuiltinImmutable 表示 builtin 角色不允许执行禁用或删除等破坏性生命周期操作。
	ErrRoleBuiltinImmutable = errors.New("builtin role is immutable")

	// ErrRolePermissionsImmutable 表示目标角色的权限绑定不允许被修改。
	ErrRolePermissionsImmutable = errors.New("role permissions are immutable")

	// ErrRoleEnabledDeletionForbidden 表示启用中的角色不能直接删除。
	ErrRoleEnabledDeletionForbidden = errors.New("enabled role cannot be deleted")

	// ErrRoleBindingsExist 表示角色仍然存在用户或权限绑定，不能执行需要空绑定的生命周期动作。
	ErrRoleBindingsExist = errors.New("role bindings exist")

	// ErrRoleDisabledAssignmentForbidden 表示禁用角色不能参与新的授权绑定。
	ErrRoleDisabledAssignmentForbidden = errors.New("disabled role cannot be assigned")

	// ErrInvalidID 表示调用方提供的稳定标识不满足当前仓储契约要求。
	ErrInvalidID = errors.New("invalid id")
)

const (
	// RoleStatusEnabled 表示角色当前可参与鉴权与绑定。
	RoleStatusEnabled = "enabled"
	// RoleStatusDisabled 表示角色已被停用，不能参与新的授权绑定。
	RoleStatusDisabled = "disabled"
)

// Role 表示 RBAC 角色的稳定持久化 DTO。
type Role struct {
	ID              uint64
	Name            string
	Display         string
	Description     *string
	Builtin         bool
	Status          string
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
	DisplayKey       *string
	Description      *string
	DescriptionKey   *string
	Category         string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	RoleBindingCount int
}

// RoleFilter 描述角色列表读取支持的局部过滤条件。
type RoleFilter struct {
	Status  string
	Query   string
	Builtin *bool
}

// PermissionFilter 描述权限列表读取支持的局部过滤条件。
type PermissionFilter struct {
	Category string
	Query    string
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
	DisplayKey  *string
	Description *string
	Builtin     bool
}

// EnsurePermissionInput 描述一次最小权限存在性保障所需的输入。
type EnsurePermissionInput struct {
	Code           string
	Display        string
	DisplayKey     *string
	Description    *string
	DescriptionKey *string
	Category       string
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

// SetRoleStatusInput 描述一次角色启停状态更新所需的输入。
type SetRoleStatusInput struct {
	ID     uint64
	Status string
}

// SoftDeleteRoleInput 描述一次角色软删除所需的输入。
type SoftDeleteRoleInput struct {
	ID uint64
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

// AddPermissionsToRoleInput 描述一次角色权限增量绑定所需的输入。
type AddPermissionsToRoleInput struct {
	RoleID        uint64
	PermissionIDs []uint64
}

// RemovePermissionsFromRoleInput 描述一次角色权限解绑所需的输入。
type RemovePermissionsFromRoleInput struct {
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

// AddRolesToUserInput 描述一次用户角色增量绑定所需的输入。
type AddRolesToUserInput struct {
	UserID  uint64
	RoleIDs []uint64
}

// RemoveRolesFromUserInput 描述一次用户角色解绑所需的输入。
type RemoveRolesFromUserInput struct {
	UserID  uint64
	RoleIDs []uint64
}

// BatchUserRoleMutationInput 描述一次批量用户角色写入所需的输入。
type BatchUserRoleMutationInput struct {
	UserIDs []uint64
	RoleIDs []uint64
}

// Repository 暴露 RBAC 模块私有持久化能力。
type Repository interface {
	EnsureRole(ctx context.Context, input EnsureRoleInput) (Role, error)
	EnsurePermission(ctx context.Context, input EnsurePermissionInput) (Permission, error)
	CreateRole(ctx context.Context, input CreateRoleInput) (Role, error)
	UpdateRole(ctx context.Context, input UpdateRoleInput) (Role, error)
	SetRoleStatus(ctx context.Context, input SetRoleStatusInput) (Role, error)
	SoftDeleteRole(ctx context.Context, input SoftDeleteRoleInput) error
	AssignPermissionsToRole(ctx context.Context, input AssignPermissionsToRoleInput) error
	ReplacePermissionsForRole(ctx context.Context, input ReplacePermissionsForRoleInput) error
	AddPermissionsToRole(ctx context.Context, input AddPermissionsToRoleInput) error
	RemovePermissionsFromRole(ctx context.Context, input RemovePermissionsFromRoleInput) error
	AssignRoleToUser(ctx context.Context, input AssignRoleToUserInput) error
	ReplaceRolesForUser(ctx context.Context, input ReplaceRolesForUserInput) error
	AddRolesToUser(ctx context.Context, input AddRolesToUserInput) error
	RemoveRolesFromUser(ctx context.Context, input RemoveRolesFromUserInput) error
	GetRoleByID(ctx context.Context, roleID uint64) (Role, error)
	GetPermissionByID(ctx context.Context, permissionID uint64) (Permission, error)
	ListRolesByUserID(ctx context.Context, userID uint64) ([]Role, error)
	ListRolesByUserIDs(ctx context.Context, userIDs []uint64) (map[uint64][]Role, error)
	ListRoles(ctx context.Context, filter RoleFilter) ([]Role, error)
	ListPermissionsByUserID(ctx context.Context, userID uint64) ([]Permission, error)
	ListUserIDsByPermissionCode(ctx context.Context, permissionCode string) ([]uint64, error)
	ListPermissions(ctx context.Context, filter PermissionFilter) ([]Permission, error)
	ListRolePermissionBindings(ctx context.Context, roleID uint64) ([]RolePermissionBinding, error)
}
