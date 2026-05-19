package rbac

import (
	"context"
	"errors"
	"sort"

	"graft/server/internal/pluginapi"
	rbacstore "graft/server/plugins/rbac/store"
)

type readManagementService interface {
	ListRoles(ctx context.Context) ([]rbacstore.Role, error)
	ListPermissions(ctx context.Context) ([]rbacstore.Permission, error)
	ListRolePermissionBindings(ctx context.Context, roleID uint64) ([]rbacstore.RolePermissionBinding, error)
	ListRoleIDsByUserID(ctx context.Context, userID uint64) ([]uint64, error)
}

type managementReader struct {
	users pluginapi.UserService
	rbac  rbacstore.Repository
}

func (r managementReader) ListRoles(ctx context.Context) ([]rbacstore.Role, error) {
	if r.rbac == nil {
		return nil, errors.New("rbac repository is unavailable")
	}

	return r.rbac.ListRoles(ctx)
}

func (r managementReader) ListPermissions(ctx context.Context) ([]rbacstore.Permission, error) {
	if r.rbac == nil {
		return nil, errors.New("rbac repository is unavailable")
	}

	return r.rbac.ListPermissions(ctx)
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
