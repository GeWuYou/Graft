// Package storeadapter keeps Phase 2 compatibility adapters between shared store
// seams and plugin-owned RBAC persistence contracts.
package storeadapter

import (
	"context"
	"errors"

	"graft/server/internal/pluginapi"
	internalstore "graft/server/internal/store"
	rbacstore "graft/server/plugins/rbac/store"
)

// NewInternalRepositoryAdapter 把过渡期共享 store 仓储收敛为 RBAC 插件私有仓储契约。
func NewInternalRepositoryAdapter(repo internalstore.RBACRepository) rbacstore.Repository {
	return internalRepositoryAdapter{delegate: repo}
}

type internalRepositoryAdapter struct {
	delegate internalstore.RBACRepository
}

func (a internalRepositoryAdapter) EnsureRole(ctx context.Context, input rbacstore.EnsureRoleInput) (rbacstore.Role, error) {
	record, err := a.delegate.EnsureRole(ctx, internalstore.EnsureRoleInput{
		Name:        input.Name,
		Display:     input.Display,
		Description: input.Description,
		Builtin:     input.Builtin,
	})
	return toRole(record), mapError(err)
}

func (a internalRepositoryAdapter) EnsurePermission(ctx context.Context, input rbacstore.EnsurePermissionInput) (rbacstore.Permission, error) {
	record, err := a.delegate.EnsurePermission(ctx, internalstore.EnsurePermissionInput{
		Code:        input.Code,
		Display:     input.Display,
		Description: input.Description,
		Category:    input.Category,
	})
	return toPermission(record), mapError(err)
}

func (a internalRepositoryAdapter) CreateRole(ctx context.Context, input rbacstore.CreateRoleInput) (rbacstore.Role, error) {
	record, err := a.delegate.CreateRole(ctx, internalstore.CreateRoleInput{
		Name:        input.Name,
		Display:     input.Display,
		Description: input.Description,
		Builtin:     input.Builtin,
	})
	return toRole(record), mapError(err)
}

func (a internalRepositoryAdapter) UpdateRole(ctx context.Context, input rbacstore.UpdateRoleInput) (rbacstore.Role, error) {
	record, err := a.delegate.UpdateRole(ctx, internalstore.UpdateRoleInput{
		ID:          input.ID,
		Name:        input.Name,
		Display:     input.Display,
		Description: input.Description,
	})
	return toRole(record), mapError(err)
}

func (a internalRepositoryAdapter) AssignPermissionsToRole(ctx context.Context, input rbacstore.AssignPermissionsToRoleInput) error {
	return mapError(a.delegate.AssignPermissionsToRole(ctx, internalstore.AssignPermissionsToRoleInput{
		RoleID:        input.RoleID,
		PermissionIDs: input.PermissionIDs,
	}))
}

func (a internalRepositoryAdapter) ReplacePermissionsForRole(ctx context.Context, input rbacstore.ReplacePermissionsForRoleInput) error {
	return mapError(a.delegate.ReplacePermissionsForRole(ctx, internalstore.ReplacePermissionsForRoleInput{
		RoleID:        input.RoleID,
		PermissionIDs: input.PermissionIDs,
	}))
}

func (a internalRepositoryAdapter) AssignRoleToUser(ctx context.Context, input rbacstore.AssignRoleToUserInput) error {
	return mapError(a.delegate.AssignRoleToUser(ctx, internalstore.AssignRoleToUserInput{
		UserID: input.UserID,
		RoleID: input.RoleID,
	}))
}

func (a internalRepositoryAdapter) ReplaceRolesForUser(ctx context.Context, input rbacstore.ReplaceRolesForUserInput) error {
	return mapError(a.delegate.ReplaceRolesForUser(ctx, internalstore.ReplaceRolesForUserInput{
		UserID:  input.UserID,
		RoleIDs: input.RoleIDs,
	}))
}

func (a internalRepositoryAdapter) GetRoleByID(ctx context.Context, roleID uint64) (rbacstore.Role, error) {
	record, err := a.delegate.GetRoleByID(ctx, roleID)
	return toRole(record), mapError(err)
}

func (a internalRepositoryAdapter) ListRolesByUserID(ctx context.Context, userID uint64) ([]rbacstore.Role, error) {
	records, err := a.delegate.ListRolesByUserID(ctx, userID)
	return toRoles(records), mapError(err)
}

func (a internalRepositoryAdapter) ListRoles(ctx context.Context) ([]rbacstore.Role, error) {
	records, err := a.delegate.ListRoles(ctx)
	return toRoles(records), mapError(err)
}

func (a internalRepositoryAdapter) ListPermissionsByUserID(ctx context.Context, userID uint64) ([]rbacstore.Permission, error) {
	records, err := a.delegate.ListPermissionsByUserID(ctx, userID)
	return toPermissions(records), mapError(err)
}

func (a internalRepositoryAdapter) ListPermissions(ctx context.Context) ([]rbacstore.Permission, error) {
	records, err := a.delegate.ListPermissions(ctx)
	return toPermissions(records), mapError(err)
}

func (a internalRepositoryAdapter) ListRolePermissionBindings(ctx context.Context, roleID uint64) ([]rbacstore.RolePermissionBinding, error) {
	records, err := a.delegate.ListRolePermissionBindings(ctx, roleID)
	return toRolePermissionBindings(records), mapError(err)
}

func toRole(record internalstore.Role) rbacstore.Role {
	return rbacstore.Role{
		ID:          record.ID,
		Name:        record.Name,
		Display:     record.Display,
		Description: record.Description,
		Builtin:     record.Builtin,
		CreatedAt:   record.CreatedAt,
		UpdatedAt:   record.UpdatedAt,
	}
}

func toPermission(record internalstore.Permission) rbacstore.Permission {
	return rbacstore.Permission{
		ID:          record.ID,
		Code:        record.Code,
		Display:     record.Display,
		Description: record.Description,
		Category:    record.Category,
		CreatedAt:   record.CreatedAt,
		UpdatedAt:   record.UpdatedAt,
	}
}

func toRoles(records []internalstore.Role) []rbacstore.Role {
	items := make([]rbacstore.Role, 0, len(records))
	for _, record := range records {
		items = append(items, toRole(record))
	}
	return items
}

func toPermissions(records []internalstore.Permission) []rbacstore.Permission {
	items := make([]rbacstore.Permission, 0, len(records))
	for _, record := range records {
		items = append(items, toPermission(record))
	}
	return items
}

func toRolePermissionBindings(records []internalstore.RolePermissionBinding) []rbacstore.RolePermissionBinding {
	items := make([]rbacstore.RolePermissionBinding, 0, len(records))
	for _, record := range records {
		items = append(items, rbacstore.RolePermissionBinding{
			RoleID:       record.RoleID,
			PermissionID: record.PermissionID,
		})
	}
	return items
}

func mapError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, internalstore.ErrRoleNotFound):
		return rbacstore.ErrRoleNotFound
	case errors.Is(err, internalstore.ErrPermissionNotFound):
		return rbacstore.ErrPermissionNotFound
	case errors.Is(err, internalstore.ErrRoleNameConflict):
		return rbacstore.ErrRoleNameConflict
	case errors.Is(err, internalstore.ErrInvalidID):
		return rbacstore.ErrInvalidID
	case errors.Is(err, internalstore.ErrUserNotFound):
		return pluginapi.ErrUserNotFound
	default:
		return err
	}
}
