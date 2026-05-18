package rbac

import (
	"context"
	"errors"
	"slices"
	"strings"

	"graft/server/internal/pluginapi"
	rbacstore "graft/server/plugins/rbac/store"
)

type accessService struct {
	rbac rbacstore.Repository
}

func (s accessService) ListRoleNamesByUserID(ctx context.Context, userID uint64) ([]string, error) {
	if s.rbac == nil {
		return nil, errors.New("rbac repository is unavailable")
	}

	roles, err := s.rbac.ListRolesByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return stableStrings(roles, func(role rbacstore.Role) string { return role.Name }), nil
}

func (s accessService) ListPermissionCodesByUserID(ctx context.Context, userID uint64) ([]string, error) {
	if s.rbac == nil {
		return nil, errors.New("rbac repository is unavailable")
	}

	permissions, err := s.rbac.ListPermissionsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return stableStrings(permissions, func(permission rbacstore.Permission) string { return permission.Code }), nil
}

var _ pluginapi.RBACAccessService = accessService{}

func stableStrings[T any](items []T, extract func(T) string) []string {
	values := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		value := strings.TrimSpace(extract(item))
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}

		seen[value] = struct{}{}
		values = append(values, value)
	}

	slices.Sort(values)
	return values
}
