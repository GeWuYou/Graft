package rbac

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"graft/server/internal/pluginapi"
	rbacstore "graft/server/modules/rbac/store"
)

type readManagementService interface {
	GetRole(ctx context.Context, roleID uint64) (rbacstore.Role, error)
	GetPermission(ctx context.Context, permissionID uint64) (rbacstore.Permission, error)
	ListRoles(ctx context.Context, filter rbacstore.RoleFilter) ([]rbacstore.Role, error)
	ListPermissions(ctx context.Context, filter rbacstore.PermissionFilter) ([]rbacstore.Permission, error)
	ListRolePermissionBindings(ctx context.Context, roleID uint64) ([]rbacstore.RolePermissionBinding, error)
	ListRoleIDsByUserID(ctx context.Context, userID uint64) ([]uint64, error)
}

type managementReader struct {
	users pluginapi.UserService
	rbac  rbacstore.Repository
}

func (r managementReader) GetRole(ctx context.Context, roleID uint64) (rbacstore.Role, error) {
	if r.rbac == nil {
		return rbacstore.Role{}, errors.New("rbac repository is unavailable")
	}

	role, err := r.rbac.GetRoleByID(ctx, roleID)
	if err != nil {
		return rbacstore.Role{}, fmt.Errorf("get role by id %d: %w", roleID, err)
	}
	return role, nil
}

func (r managementReader) GetPermission(ctx context.Context, permissionID uint64) (rbacstore.Permission, error) {
	if r.rbac == nil {
		return rbacstore.Permission{}, errors.New("rbac repository is unavailable")
	}

	permission, err := r.rbac.GetPermissionByID(ctx, permissionID)
	if err != nil {
		return rbacstore.Permission{}, fmt.Errorf("get permission by id %d: %w", permissionID, err)
	}
	return permission, nil
}

func (r managementReader) ListRoles(ctx context.Context, filter rbacstore.RoleFilter) ([]rbacstore.Role, error) {
	if r.rbac == nil {
		return nil, errors.New("rbac repository is unavailable")
	}

	roles, err := r.rbac.ListRoles(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}
	return roles, nil
}

func (r managementReader) ListPermissions(ctx context.Context, filter rbacstore.PermissionFilter) ([]rbacstore.Permission, error) {
	if r.rbac == nil {
		return nil, errors.New("rbac repository is unavailable")
	}

	permissions, err := r.rbac.ListPermissions(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list permissions: %w", err)
	}
	return permissions, nil
}

func (r managementReader) ListRolePermissionBindings(ctx context.Context, roleID uint64) ([]rbacstore.RolePermissionBinding, error) {
	if r.rbac == nil {
		return nil, errors.New("rbac repository is unavailable")
	}

	return r.rbac.ListRolePermissionBindings(ctx, roleID)
}

func (r managementReader) ListRoleIDsByUserID(ctx context.Context, userID uint64) ([]uint64, error) {
	if r.users == nil {
		return nil, errors.New("user service is unavailable")
	}
	if r.rbac == nil {
		return nil, errors.New("rbac repository is unavailable")
	}

	if _, err := r.users.GetUserByID(ctx, userID); err != nil {
		return nil, err
	}

	roles, err := r.rbac.ListRolesByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	roleIDs := make([]uint64, 0, len(roles))
	for _, role := range roles {
		roleIDs = append(roleIDs, role.ID)
	}
	sort.Slice(roleIDs, func(i, j int) bool {
		return roleIDs[i] < roleIDs[j]
	})

	return roleIDs, nil
}
