package rbac

import (
	"context"
	"fmt"

	"graft/server/internal/moduleapi"
	rbacstore "graft/server/modules/rbac/store"
)

type accessService struct {
	rbac  rbacstore.Repository
	users moduleapi.UserSecurityReader
}

type securityPostureInputs struct {
	users       []moduleapi.UserSecuritySummary
	roles       []rbacstore.Role
	permissions []rbacstore.Permission
}

// ReadSecurityPosture aggregates RBAC-owned counters without exposing persistence details.
func (s accessService) ReadSecurityPosture(ctx context.Context) (moduleapi.SecurityPosture, error) {
	if s.rbac == nil || s.users == nil {
		return moduleapi.SecurityPosture{}, fmt.Errorf("rbac security posture dependencies are unavailable")
	}
	inputs, err := s.loadSecurityPostureInputs(ctx)
	if err != nil {
		return moduleapi.SecurityPosture{}, err
	}
	return s.buildSecurityPosture(ctx, inputs)
}

func (s accessService) loadSecurityPostureInputs(ctx context.Context) (securityPostureInputs, error) {
	users, err := s.users.ListSecuritySummaries(ctx)
	if err != nil {
		return securityPostureInputs{}, fmt.Errorf("list security user summaries: %w", err)
	}
	roles, err := s.rbac.ListRoles(ctx, rbacstore.RoleFilter{})
	if err != nil {
		return securityPostureInputs{}, fmt.Errorf("list roles for security posture: %w", err)
	}
	permissions, err := s.rbac.ListPermissions(ctx, rbacstore.PermissionFilter{})
	if err != nil {
		return securityPostureInputs{}, fmt.Errorf("list permissions for security posture: %w", err)
	}
	return securityPostureInputs{users: users, roles: roles, permissions: permissions}, nil
}

func (s accessService) buildSecurityPosture(ctx context.Context, inputs securityPostureInputs) (moduleapi.SecurityPosture, error) {
	userIDs := make([]uint64, 0, len(inputs.users))
	posture := moduleapi.SecurityPosture{TotalUsers: len(inputs.users), RoleCount: len(inputs.roles), PermissionCount: len(inputs.permissions)}
	for _, user := range inputs.users {
		userIDs = append(userIDs, user.ID)
		if user.Status == "disabled" {
			posture.DisabledUsers++
		}
	}
	rolesByUserID, err := s.rbac.ListRolesByUserIDs(ctx, userIDs)
	if err != nil {
		return moduleapi.SecurityPosture{}, fmt.Errorf("list role bindings for security posture: %w", err)
	}
	for _, roleIDs := range rolesByUserID {
		posture.RoleAssignmentCount += len(roleIDs)
	}
	for _, user := range inputs.users {
		if len(rolesByUserID[user.ID]) == 0 {
			posture.UnassignedUserCount++
		}
	}
	for _, role := range inputs.roles {
		if role.Builtin {
			posture.BuiltinRoleCount++
			continue
		}
		posture.CustomRoleCount++
		if role.PermissionCount == 0 {
			posture.EmptyCustomRoleCount++
		}
	}
	return posture, nil
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
var _ moduleapi.RBACSecurityPostureService = accessService{}
