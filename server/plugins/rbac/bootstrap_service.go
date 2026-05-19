package rbac

import (
	"context"
	"errors"
	"fmt"

	"graft/server/internal/pluginapi"
	rbacstore "graft/server/plugins/rbac/store"
)

type bootstrapService struct {
	rbac rbacstore.Repository
}

// NewBootstrapService exposes the stable RBAC bootstrap capability over a plugin-local repository.
func NewBootstrapService(rbac rbacstore.Repository) pluginapi.RBACBootstrapService {
	if rbac == nil {
		return nil
	}

	return bootstrapService{rbac: rbac}
}

func (s bootstrapService) EnsureDefaultAdminAccess(
	ctx context.Context,
	userID uint64,
	permissions []pluginapi.PermissionSeed,
) error {
	if s.rbac == nil {
		return errors.New("rbac repository is unavailable")
	}

	role, err := s.rbac.EnsureRole(ctx, rbacstore.EnsureRoleInput{
		Name:    builtinAdminRoleName,
		Display: "管理员",
		Builtin: true,
	})
	if err != nil {
		return fmt.Errorf("ensure default admin role: %w", err)
	}

	if err := ensureRolePermissions(ctx, s.rbac, role.ID, permissions); err != nil {
		return err
	}
	if err := s.rbac.AssignRoleToUser(ctx, rbacstore.AssignRoleToUserInput{
		UserID: userID,
		RoleID: role.ID,
	}); err != nil {
		return fmt.Errorf("assign default admin role to user: %w", err)
	}

	return nil
}

func ensureRolePermissions(
	ctx context.Context,
	rbac rbacstore.Repository,
	roleID uint64,
	permissions []pluginapi.PermissionSeed,
) error {
	permissionIDs := make([]uint64, 0, len(permissions))
	for _, item := range permissions {
		record, err := rbac.EnsurePermission(ctx, rbacstore.EnsurePermissionInput{
			Code:        item.Code,
			Display:     item.Display,
			Description: stringPtrOrNil(item.Description),
			Category:    item.Category,
		})
		if err != nil {
			return fmt.Errorf("ensure permission %s: %w", item.Code, err)
		}
		permissionIDs = append(permissionIDs, record.ID)
	}
	if len(permissionIDs) == 0 {
		return nil
	}

	if err := rbac.AssignPermissionsToRole(ctx, rbacstore.AssignPermissionsToRoleInput{
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

var _ pluginapi.RBACBootstrapService = bootstrapService{}
