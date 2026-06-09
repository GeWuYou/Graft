// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package rbac

import (
	"context"
	"errors"
	"slices"
	"strings"

	"graft/server/internal/moduleapi"
	rbacstore "graft/server/modules/rbac/store"
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

func (s accessService) ListRoleSummariesByUserIDs(
	ctx context.Context,
	userIDs []uint64,
) (map[uint64][]moduleapi.RoleSummary, error) {
	if s.rbac == nil {
		return nil, errors.New("rbac repository is unavailable")
	}

	rolesByUserID, err := s.rbac.ListRolesByUserIDs(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	summaries := make(map[uint64][]moduleapi.RoleSummary, len(rolesByUserID))
	for userID, roles := range rolesByUserID {
		items := make([]moduleapi.RoleSummary, 0, len(roles))
		for _, role := range roles {
			items = append(items, moduleapi.RoleSummary{
				ID:      role.ID,
				Name:    strings.TrimSpace(role.Name),
				Display: strings.TrimSpace(role.Display),
			})
		}
		slices.SortFunc(items, func(left, right moduleapi.RoleSummary) int {
			if left.ID == right.ID {
				return strings.Compare(left.Name, right.Name)
			}
			if left.ID < right.ID {
				return -1
			}
			return 1
		})
		summaries[userID] = items
	}

	for _, userID := range userIDs {
		if _, ok := summaries[userID]; !ok {
			summaries[userID] = []moduleapi.RoleSummary{}
		}
	}

	return summaries, nil
}

var _ moduleapi.RBACAccessService = accessService{}

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
