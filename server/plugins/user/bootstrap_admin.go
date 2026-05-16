package user

import (
	"context"
	"errors"
	"fmt"

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

	credential, err := s.auth.GetUserCredentialByUsername(ctx, defaultAdminUsername)
	if err != nil {
		if !errors.Is(err, store.ErrUserNotFound) {
			return fmt.Errorf("get default admin credential: %w", err)
		}

		hash, hashErr := s.passwords.Hash(defaultAdminPassword)
		if hashErr != nil {
			return fmt.Errorf("hash default admin password: %w", hashErr)
		}

		credential, err = s.auth.EnsureUserCredential(ctx, store.EnsureUserCredentialInput{
			Username:           defaultAdminUsername,
			Display:            defaultAdminDisplay,
			PasswordHash:       hash,
			MustChangePassword: true,
		})
		if err != nil {
			return fmt.Errorf("ensure default admin credential: %w", err)
		}
	}

	role, err := rbac.EnsureRole(ctx, store.EnsureRoleInput{
		Name:    defaultAdminRoleName,
		Display: "管理员",
	})
	if err != nil {
		return fmt.Errorf("ensure default admin role: %w", err)
	}

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
	if len(permissionIDs) > 0 {
		if err := rbac.AssignPermissionsToRole(ctx, store.AssignPermissionsToRoleInput{
			RoleID:        role.ID,
			PermissionIDs: permissionIDs,
		}); err != nil {
			return fmt.Errorf("assign permissions to default admin role: %w", err)
		}
	}

	if err := rbac.AssignRoleToUser(ctx, store.AssignRoleToUserInput{
		UserID: credential.UserID,
		RoleID: role.ID,
	}); err != nil {
		return fmt.Errorf("assign default admin role to user: %w", err)
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
