package user

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"graft/server/internal/permission"
	"graft/server/internal/store"
)

// ensureDefaultAdmin 幂等确保默认管理员存在且具备当前 MVP 所需的最小后台可见性。
func (s authService) ensureDefaultAdmin(ctx context.Context, rbac store.RBACRepository, permissions []permission.Item) error {
	if s.auth == nil {
		return fmt.Errorf("auth repository is unavailable")
	}
	if rbac == nil {
		return fmt.Errorf("rbac repository is unavailable")
	}

	credential, err := s.ensureAdminCredential(ctx)
	if err != nil {
		return err
	}
	role, err := rbac.EnsureRole(ctx, store.EnsureRoleInput{
		Name:    defaultAdminRoleName,
		Display: "管理员",
	})
	if err != nil {
		return fmt.Errorf("ensure default admin role: %w", err)
	}

	if err := ensureRolePermissions(ctx, rbac, role.ID, permissions); err != nil {
		return err
	}
	if err := rbac.AssignRoleToUser(ctx, store.AssignRoleToUserInput{
		UserID: credential.UserID,
		RoleID: role.ID,
	}); err != nil {
		return fmt.Errorf("assign default admin role to user: %w", err)
	}

	return nil
}

func (s authService) ensureAdminCredential(ctx context.Context) (store.UserCredential, error) {
	credential, err := s.auth.GetUserCredentialByUsername(ctx, defaultAdminUsername)
	if err == nil {
		return s.reconcileDefaultAdminCredential(ctx, credential)
	}
	if !errors.Is(err, store.ErrUserNotFound) {
		return store.UserCredential{}, fmt.Errorf("get default admin credential: %w", err)
	}

	hash, hashErr := s.passwords.Hash(defaultAdminPassword)
	if hashErr != nil {
		return store.UserCredential{}, fmt.Errorf("hash default admin password: %w", hashErr)
	}

	credential, err = s.auth.EnsureUserCredential(ctx, store.EnsureUserCredentialInput{
		Username:           defaultAdminUsername,
		Display:            defaultAdminDisplay,
		PasswordHash:       hash,
		MustChangePassword: true,
	})
	if err != nil {
		return store.UserCredential{}, fmt.Errorf("ensure default admin credential: %w", err)
	}

	return credential, nil
}

func (s authService) reconcileDefaultAdminCredential(
	ctx context.Context,
	credential store.UserCredential,
) (store.UserCredential, error) {
	if credential.MustChangePassword || credential.PasswordHash == nil || *credential.PasswordHash == "" {
		return credential, nil
	}

	if err := s.passwords.Compare(*credential.PasswordHash, defaultAdminPassword); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return credential, nil
		}
		return store.UserCredential{}, fmt.Errorf("compare default admin password hash: %w", err)
	}

	if err := s.auth.SetPasswordHash(ctx, store.SetPasswordHashInput{
		UserID:             credential.UserID,
		PasswordHash:       *credential.PasswordHash,
		MustChangePassword: true,
		ChangedAt:          credential.PasswordChangedAt,
	}); err != nil {
		return store.UserCredential{}, fmt.Errorf("mark default admin credential for password change: %w", err)
	}

	credential.MustChangePassword = true
	return credential, nil
}

func ensureRolePermissions(
	ctx context.Context,
	rbac store.RBACRepository,
	roleID uint64,
	permissions []permission.Item,
) error {
	permissionIDs := make([]uint64, 0, len(permissions))
	for _, item := range permissions {
		record, err := rbac.EnsurePermission(ctx, store.EnsurePermissionInput{
			Code:        item.Code,
			Display:     item.Name,
			Description: stringPtrOrNil(item.Description),
		})
		if err != nil {
			return fmt.Errorf("ensure permission %s: %w", item.Code, err)
		}
		permissionIDs = append(permissionIDs, record.ID)
	}
	if len(permissionIDs) == 0 {
		return nil
	}

	if err := rbac.AssignPermissionsToRole(ctx, store.AssignPermissionsToRoleInput{
		RoleID:        roleID,
		PermissionIDs: permissionIDs,
	}); err != nil {
		return fmt.Errorf("assign permissions to default admin role: %w", err)
	}

	return nil
}

func stringPtrOrNil(value string) *string {
	if value == "" {
		return nil
	}
	result := value
	return &result
}
