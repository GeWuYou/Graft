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

// ResetDefaultAdminForDevelopment restores the default administrator through
// auth-owned credentials and the user-owned profile identity capability.
//
// the configured default access.
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
	if err := repository.SetPasswordHash(ctx, authstore.SetPasswordHashInput{UserID: userID, PasswordHash: hash, MustChangePassword: true, ChangedAt: &now}); err != nil {
		return fmt.Errorf("reset default admin password hash: %w", err)
	}
	if err := repository.RevokeRefreshSessionsByUserID(ctx, authstore.RevokeRefreshSessionsByUserIDInput{UserID: userID, RevokedAt: now}); err != nil {
		return fmt.Errorf("revoke default admin refresh sessions: %w", err)
	}
	return nil
}

// isDevelopmentResetEnv determines whether the environment permits development admin resets.
func isDevelopmentResetEnv(env string) bool {
	switch strings.TrimSpace(env) {
	case "local", "test":
		return true
	default:
		return false
	}
}
