// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package moduleapi

import "context"

// PermissionSeed 描述 RBAC 初始化或对齐权限点时需要的最小稳定元数据。
type PermissionSeed struct {
	Code           string
	Display        string
	DisplayKey     string
	Description    string
	DescriptionKey string
	Category       string
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
	ListRoleSummariesByUserIDs(ctx context.Context, userIDs []uint64) (map[uint64][]RoleSummary, error)
}

// RBACBootstrapService 暴露默认管理员访问基线的最小幂等引导能力。
type RBACBootstrapService interface {
	EnsureDefaultAdminAccess(ctx context.Context, userID uint64, permissions []PermissionSeed) error
}
