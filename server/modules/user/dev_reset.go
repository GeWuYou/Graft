// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package user

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	"graft/server/internal/moduleapi"
	userstore "graft/server/modules/user/store"
	"graft/server/modules/user/storeent"

	"go.uber.org/zap"
)

// AuthRepositoryForReset narrows the dev-reset helper to the module-owned auth boundary.
type AuthRepositoryForReset = userstore.AuthRepository

// NewAuthRepositoryForReset exposes the user module's dev-reset auth boundary.
func NewAuthRepositoryForReset(sqlDB *sql.DB) (AuthRepositoryForReset, error) {
	storeRuntime, err := storeent.NewRuntime(sqlDB, zap.NewNop())
	if err != nil {
		return nil, fmt.Errorf("build user storeent runtime: %w", err)
	}

	authRepo, err := storeRuntime.NewAuthRepository()
	if err != nil {
		return nil, fmt.Errorf("build user auth repository: %w", err)
	}

	return authRepo, nil
}

// ResetDefaultAdminForDevelopment 在开发环境里把默认管理员重置回首次登录受限态。
//
// 该 helper 只复用当前 user 模块已冻结的默认管理员真值：
//   - 确保 `graft` 存在
//   - 把密码重置为初始化例外密码 `graft-admin`
//   - 把 `must_change_password` 重新置为 true
//   - 重新绑定最小管理员角色与当前模块声明的权限
//
// 这个入口只用于 CLI 的 dev-only 调试能力，不应被运行时业务链路或 HTTP 路由直接复用。
func ResetDefaultAdminForDevelopment(
	ctx context.Context,
	authRepo userstore.AuthRepository,
	rbac moduleapi.RBACBootstrapService,
) error {
	if !isDevelopmentResetEnv(os.Getenv("GRAFT_APP_ENV")) {
		return fmt.Errorf("reset default admin is only available in local/test environments, got %q", strings.TrimSpace(os.Getenv("GRAFT_APP_ENV")))
	}

	service := authService{
		auth:      authRepo,
		passwords: newPasswordHasher(),
	}

	return service.resetDefaultAdminForDevelopment(ctx, rbac)
}

func (s authService) resetDefaultAdminForDevelopment(ctx context.Context, rbac moduleapi.RBACBootstrapService) error {
	if s.auth == nil {
		return fmt.Errorf("auth repository is unavailable")
	}
	if rbac == nil {
		return fmt.Errorf("rbac bootstrap service is unavailable")
	}

	hash, err := s.passwords.Hash(defaultAdminPassword)
	if err != nil {
		return fmt.Errorf("hash default admin password: %w", err)
	}

	credential, err := s.auth.EnsureUserCredential(ctx, userstore.EnsureUserCredentialInput{
		Username:           defaultAdminUsername,
		Display:            defaultAdminDisplay,
		PasswordHash:       hash,
		MustChangePassword: true,
	})
	if err != nil {
		return fmt.Errorf("ensure default admin credential: %w", err)
	}

	changedAt := s.nowUTC()
	if err := s.auth.SetPasswordHash(ctx, userstore.SetPasswordHashInput{
		UserID:             credential.UserID,
		PasswordHash:       hash,
		MustChangePassword: true,
		ChangedAt:          &changedAt,
	}); err != nil {
		return fmt.Errorf("reset default admin password hash: %w", err)
	}

	if err := s.auth.RevokeRefreshSessionsByUserID(ctx, userstore.RevokeRefreshSessionsByUserIDInput{
		UserID:    credential.UserID,
		RevokedAt: changedAt,
	}); err != nil {
		return fmt.Errorf("revoke default admin refresh sessions: %w", err)
	}

	if err := rbac.EnsureDefaultAdminAccess(ctx, credential.UserID, permissionSeedsFromItems(userPermissionItems("user"))); err != nil {
		return fmt.Errorf("ensure default admin access: %w", err)
	}

	return nil
}

func isDevelopmentResetEnv(env string) bool {
	switch strings.TrimSpace(env) {
	case "local", "test":
		return true
	default:
		return false
	}
}
