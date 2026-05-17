package user

import (
	"context"
	"fmt"
	"os"
	"strings"

	"graft/server/internal/store"
)

// ResetDefaultAdminForDevelopment 在开发环境里把默认管理员重置回首次登录受限态。
//
// 该 helper 只复用当前 user 插件已冻结的默认管理员真值：
//   - 确保 `graft` 存在
//   - 把密码重置为初始化例外密码 `graft-admin`
//   - 把 `must_change_password` 重新置为 true
//   - 重新绑定最小管理员角色与当前插件声明的权限
//
// 这个入口只用于 CLI 的 dev-only 调试能力，不应被运行时业务链路或 HTTP 路由直接复用。
func ResetDefaultAdminForDevelopment(
	ctx context.Context,
	authRepo store.AuthRepository,
	rbacRepo store.RBACRepository,
) error {
	if !isDevelopmentResetEnv(os.Getenv("GRAFT_APP_ENV")) {
		return fmt.Errorf("reset default admin is only available in local/test environments, got %q", strings.TrimSpace(os.Getenv("GRAFT_APP_ENV")))
	}

	service := authService{
		auth:      authRepo,
		passwords: newPasswordHasher(),
	}

	return service.resetDefaultAdminForDevelopment(ctx, rbacRepo)
}

func (s authService) resetDefaultAdminForDevelopment(ctx context.Context, rbac store.RBACRepository) error {
	if s.auth == nil {
		return fmt.Errorf("auth repository is unavailable")
	}
	if rbac == nil {
		return fmt.Errorf("rbac repository is unavailable")
	}

	hash, err := s.passwords.Hash(defaultAdminPassword)
	if err != nil {
		return fmt.Errorf("hash default admin password: %w", err)
	}

	credential, err := s.auth.EnsureUserCredential(ctx, store.EnsureUserCredentialInput{
		Username:           defaultAdminUsername,
		Display:            defaultAdminDisplay,
		PasswordHash:       hash,
		MustChangePassword: true,
	})
	if err != nil {
		return fmt.Errorf("ensure default admin credential: %w", err)
	}

	changedAt := s.nowUTC()
	if err := s.auth.SetPasswordHash(ctx, store.SetPasswordHashInput{
		UserID:             credential.UserID,
		PasswordHash:       hash,
		MustChangePassword: true,
		ChangedAt:          &changedAt,
	}); err != nil {
		return fmt.Errorf("reset default admin password hash: %w", err)
	}

	if err := s.auth.RevokeRefreshSessionsByUserID(ctx, store.RevokeRefreshSessionsByUserIDInput{
		UserID:    credential.UserID,
		RevokedAt: changedAt,
	}); err != nil {
		return fmt.Errorf("revoke default admin refresh sessions: %w", err)
	}

	role, err := rbac.EnsureRole(ctx, store.EnsureRoleInput{
		Name:    defaultAdminRoleName,
		Display: "管理员",
		Builtin: true,
	})
	if err != nil {
		return fmt.Errorf("ensure default admin role: %w", err)
	}

	if err := ensureRolePermissions(ctx, rbac, role.ID, userPermissionItems("user")); err != nil {
		return fmt.Errorf("ensure default admin role permissions: %w", err)
	}

	if err := rbac.AssignRoleToUser(ctx, store.AssignRoleToUserInput{
		UserID: credential.UserID,
		RoleID: role.ID,
	}); err != nil {
		return fmt.Errorf("assign default admin role to user: %w", err)
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
