package rbac

import (
	"context"
	"errors"

	"graft/server/internal/store"
)

type readManagementService interface {
	ListRoles(ctx context.Context) ([]store.Role, error)
	ListPermissions(ctx context.Context) ([]store.Permission, error)
	ListRolePermissionBindings(ctx context.Context, roleID uint64) ([]store.RolePermissionBinding, error)
}

type managementReader struct {
	rbac store.RBACRepository
}

func (r managementReader) ListRoles(ctx context.Context) ([]store.Role, error) {
	if r.rbac == nil {
		return nil, errors.New("rbac repository is unavailable")
	}

	return r.rbac.ListRoles(ctx)
}

func (r managementReader) ListPermissions(ctx context.Context) ([]store.Permission, error) {
	if r.rbac == nil {
		return nil, errors.New("rbac repository is unavailable")
	}

	return r.rbac.ListPermissions(ctx)
}

func (r managementReader) ListRolePermissionBindings(ctx context.Context, roleID uint64) ([]store.RolePermissionBinding, error) {
	if r.rbac == nil {
		return nil, errors.New("rbac repository is unavailable")
	}

	return r.rbac.ListRolePermissionBindings(ctx, roleID)
}
