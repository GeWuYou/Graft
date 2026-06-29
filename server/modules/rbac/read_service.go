package rbac

import (
	"context"
	"fmt"

	"graft/server/internal/moduleapi"
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
	users moduleapi.UserService
	rbac  rbacstore.Repository
}

func (r managementReader) GetRole(ctx context.Context, roleID uint64) (rbacstore.Role, error) {
	return getRBACRecordByID(ctx, r.rbac, roleID, fmt.Sprintf("get role by id %d", roleID), rbacstore.Repository.GetRoleByID)
}

func (r managementReader) GetPermission(ctx context.Context, permissionID uint64) (rbacstore.Permission, error) {
	return getRBACRecordByID(ctx, r.rbac, permissionID, fmt.Sprintf("get permission by id %d", permissionID), rbacstore.Repository.GetPermissionByID)
}

func (r managementReader) ListRoles(ctx context.Context, filter rbacstore.RoleFilter) ([]rbacstore.Role, error) {
	return listRBACRecords(ctx, r.rbac, filter, "list roles", rbacstore.Repository.ListRoles)
}

func (r managementReader) ListPermissions(ctx context.Context, filter rbacstore.PermissionFilter) ([]rbacstore.Permission, error) {
	return listRBACRecords(ctx, r.rbac, filter, "list permissions", rbacstore.Repository.ListPermissions)
}

func (r managementReader) ListRolePermissionBindings(ctx context.Context, roleID uint64) ([]rbacstore.RolePermissionBinding, error) {
	return getRBACRecordByID(
		ctx,
		r.rbac,
		roleID,
		fmt.Sprintf("list role permission bindings for role %d", roleID),
		rbacstore.Repository.ListRolePermissionBindings,
	)
}

func (r managementReader) ListRoleIDsByUserID(ctx context.Context, userID uint64) ([]uint64, error) {
	return listRoleIDsByUserID(ctx, r.users, r.rbac, userID)
}
