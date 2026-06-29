package rbac

import (
	"context"

	"graft/server/internal/moduleapi"
	rbacstore "graft/server/modules/rbac/store"
)

type accessService struct {
	rbac rbacstore.Repository
}

func (s accessService) ListRoleNamesByUserID(ctx context.Context, userID uint64) ([]string, error) {
	return listStableStringsByUserID(ctx, s.rbac, userID, rbacstore.Repository.ListRolesByUserID, roleName)
}

func (s accessService) ListPermissionCodesByUserID(ctx context.Context, userID uint64) ([]string, error) {
	return listStableStringsByUserID(ctx, s.rbac, userID, rbacstore.Repository.ListPermissionsByUserID, permissionCode)
}

func (s accessService) ListUserIDsByPermissionCode(ctx context.Context, permissionCode string) ([]uint64, error) {
	return listStableUserIDsByPermissionCode(ctx, s.rbac, permissionCode)
}

func (s accessService) ListRoleSummariesByUserIDs(
	ctx context.Context,
	userIDs []uint64,
) (map[uint64][]moduleapi.RoleSummary, error) {
	return listRoleSummariesByUserIDs(ctx, s.rbac, userIDs)
}

var _ moduleapi.RBACAccessService = accessService{}
