package user

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"graft/server/internal/permission"
	"graft/server/internal/pluginapi"
	userstore "graft/server/modules/user/store"
)

// ensureDefaultAdmin 幂等确保默认管理员存在且具备当前 MVP 所需的最小后台可见性。
func (s authService) ensureDefaultAdmin(
	ctx context.Context,
	rbac pluginapi.RBACBootstrapService,
	permissions []permission.Item,
) error {
	if s.auth == nil {
		return fmt.Errorf("auth repository is unavailable")
	}
	if rbac == nil {
		return fmt.Errorf("rbac bootstrap service is unavailable")
	}

	credential, err := s.ensureAdminCredential(ctx)
	if err != nil {
		return err
	}
	return rbac.EnsureDefaultAdminAccess(ctx, credential.UserID, permissionSeedsFromItems(permissions))
}

func (s authService) ensureAdminCredential(ctx context.Context) (userstore.UserCredential, error) {
	credential, err := s.auth.GetUserCredentialByUsername(ctx, defaultAdminUsername)
	if err == nil {
		return s.reconcileDefaultAdminCredential(ctx, credential)
	}
	if !errors.Is(err, userstore.ErrUserNotFound) {
		return userstore.UserCredential{}, fmt.Errorf("get default admin credential: %w", err)
	}

	hash, hashErr := s.passwords.Hash(defaultAdminPassword)
	if hashErr != nil {
		return userstore.UserCredential{}, fmt.Errorf("hash default admin password: %w", hashErr)
	}

	credential, err = s.auth.EnsureUserCredential(ctx, userstore.EnsureUserCredentialInput{
		Username:           defaultAdminUsername,
		Display:            defaultAdminDisplay,
		PasswordHash:       hash,
		MustChangePassword: true,
	})
	if err != nil {
		return userstore.UserCredential{}, fmt.Errorf("ensure default admin credential: %w", err)
	}

	return credential, nil
}

func (s authService) reconcileDefaultAdminCredential(
	ctx context.Context,
	credential userstore.UserCredential,
) (userstore.UserCredential, error) {
	if credential.MustChangePassword || credential.PasswordHash == nil || *credential.PasswordHash == "" {
		return credential, nil
	}

	if err := s.passwords.Compare(*credential.PasswordHash, defaultAdminPassword); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return credential, nil
		}
		return userstore.UserCredential{}, fmt.Errorf("compare default admin password hash: %w", err)
	}

	if err := s.auth.SetPasswordHash(ctx, userstore.SetPasswordHashInput{
		UserID:             credential.UserID,
		PasswordHash:       *credential.PasswordHash,
		MustChangePassword: true,
		ChangedAt:          credential.PasswordChangedAt,
	}); err != nil {
		return userstore.UserCredential{}, fmt.Errorf("mark default admin credential for password change: %w", err)
	}

	credential.MustChangePassword = true
	return credential, nil
}

func permissionSeedsFromItems(items []permission.Item) []pluginapi.PermissionSeed {
	seeds := make([]pluginapi.PermissionSeed, 0, len(items))
	for _, item := range items {
		seeds = append(seeds, pluginapi.PermissionSeed{
			Code:        item.Code,
			Display:     item.Name,
			Description: item.Description,
			Category:    item.Category,
		})
	}

	return seeds
}
