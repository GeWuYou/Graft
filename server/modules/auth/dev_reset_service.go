package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"graft/server/internal/i18n"
	"graft/server/internal/moduleapi"
	"graft/server/internal/permission"
	authstore "graft/server/modules/auth/store"
)

// ResetDefaultAdminForDevelopment 仅在 local/test 环境恢复默认管理员 credential、吊销 session 并重建默认访问权限；生产环境调用会被拒绝。
func ResetDefaultAdminForDevelopment(ctx context.Context, repository authstore.AuthRepository, identity moduleapi.UserIdentityProvider, localizer *i18n.Service, rbac moduleapi.RBACBootstrapService, permissions []permission.Item) error {
	if !isDevelopmentResetEnv(os.Getenv("GRAFT_APP_ENV")) {
		return fmt.Errorf("reset default admin is only available in local/test environments, got %q", strings.TrimSpace(os.Getenv("GRAFT_APP_ENV")))
	}
	if repository == nil || identity == nil || rbac == nil {
		return errors.New("reset default admin dependencies are unavailable")
	}
	profile, err := identity.EnsureDefaultAdminProfile(ctx)
	if err != nil {
		return fmt.Errorf("ensure default admin profile: %w", err)
	}
	if err := resetDefaultAdminPasswordAndSessions(ctx, repository, profile.ID); err != nil {
		return err
	}
	seeds, err := permissionSeedsFromItems(localizer, permissions)
	if err != nil {
		return fmt.Errorf("build default admin permission seeds: %w", err)
	}
	if err := rbac.EnsureDefaultAdminAccess(ctx, profile.ID, seeds); err != nil {
		return fmt.Errorf("ensure default admin access: %w", err)
	}
	return nil
}

func resetDefaultAdminPasswordAndSessions(ctx context.Context, repository authstore.AuthRepository, userID uint64) error {
	hash, err := newPasswordHasher().Hash(defaultAdminPassword)
	if err != nil {
		return fmt.Errorf("hash default admin password: %w", err)
	}
	now := time.Now().UTC()
	return repository.RunInTransaction(ctx, func(txCtx context.Context, credentials authstore.CredentialStore, sessions authstore.SessionStore) error {
		if err := credentials.SetPasswordHash(txCtx, authstore.SetPasswordHashInput{UserID: userID, PasswordHash: hash, MustChangePassword: true, ChangedAt: &now}); err != nil {
			return fmt.Errorf("reset default admin password hash: %w", err)
		}
		if err := sessions.RevokeRefreshSessionsByUserID(txCtx, authstore.RevokeRefreshSessionsByUserIDInput{UserID: userID, RevokedAt: now}); err != nil {
			return fmt.Errorf("revoke default admin refresh sessions: %w", err)
		}
		return nil
	})
}

// isDevelopmentResetEnv 判断当前环境是否允许执行开发用管理员重置。
func isDevelopmentResetEnv(env string) bool {
	switch strings.TrimSpace(env) {
	case "local", "test":
		return true
	default:
		return false
	}
}
