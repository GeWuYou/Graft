package rbac

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"graft/server/internal/moduleapi"
	rbacstore "graft/server/modules/rbac/store"
)

var (
	errRBACRepositoryUnavailable  = errors.New("rbac repository is unavailable")
	errRBACUserServiceUnavailable = errors.New("user service is unavailable")
)

func runRBACQuery[T any](repository rbacstore.Repository, query func(rbacstore.Repository) (T, error)) (T, error) {
	var zero T
	if repository == nil {
		return zero, errRBACRepositoryUnavailable
	}

	result, err := query(repository)
	if err != nil {
		return zero, err
	}

	return result, nil
}

func runNamedRBACQuery[T any](repository rbacstore.Repository, action string, query func(rbacstore.Repository) (T, error)) (T, error) {
	result, err := runRBACQuery(repository, query)
	if err == nil {
		return result, nil
	}

	var zero T
	if errors.Is(err, errRBACRepositoryUnavailable) {
		return zero, err
	}

	return zero, fmt.Errorf("%s: %w", action, err)
}

func requireRBACUserService(users moduleapi.UserService) (moduleapi.UserService, error) {
	if users == nil {
		return nil, errRBACUserServiceUnavailable
	}

	return users, nil
}

func listStableStringsByUserID[T any](
	ctx context.Context,
	repository rbacstore.Repository,
	userID uint64,
	fetch func(rbacstore.Repository, context.Context, uint64) ([]T, error),
	extract func(T) string,
) ([]string, error) {
	items, err := runRBACQuery(repository, func(repo rbacstore.Repository) ([]T, error) {
		return fetch(repo, ctx, userID)
	})
	if err != nil {
		return nil, err
	}

	return stableStrings(items, extract), nil
}

func listStableUserIDsByPermissionCode(ctx context.Context, repository rbacstore.Repository, permissionCode string) ([]uint64, error) {
	userIDs, err := runNamedRBACQuery(repository, fmt.Sprintf("list user ids by permission %q", permissionCode), func(repo rbacstore.Repository) ([]uint64, error) {
		return repo.ListUserIDsByPermissionCode(ctx, permissionCode)
	})
	if err != nil {
		return nil, err
	}

	return stableUint64s(userIDs), nil
}

func listRoleSummariesByUserIDs(ctx context.Context, repository rbacstore.Repository, userIDs []uint64) (map[uint64][]moduleapi.RoleSummary, error) {
	rolesByUserID, err := runRBACQuery(repository, func(repo rbacstore.Repository) (map[uint64][]rbacstore.Role, error) {
		return repo.ListRolesByUserIDs(ctx, userIDs)
	})
	if err != nil {
		return nil, err
	}

	summaries := make(map[uint64][]moduleapi.RoleSummary, len(rolesByUserID))
	for userID, roles := range rolesByUserID {
		summaries[userID] = roleSummaries(roles)
	}

	for _, userID := range userIDs {
		if _, ok := summaries[userID]; !ok {
			summaries[userID] = []moduleapi.RoleSummary{}
		}
	}

	return summaries, nil
}

func listRoleIDsByUserID(
	ctx context.Context,
	users moduleapi.UserService,
	repository rbacstore.Repository,
	userID uint64,
) ([]uint64, error) {
	userService, err := requireRBACUserService(users)
	if err != nil {
		return nil, err
	}

	if _, err := userService.GetUserByID(ctx, userID); err != nil {
		return nil, err
	}

	roles, err := runRBACQuery(repository, func(repo rbacstore.Repository) ([]rbacstore.Role, error) {
		return repo.ListRolesByUserID(ctx, userID)
	})
	if err != nil {
		return nil, err
	}

	roleIDs := make([]uint64, 0, len(roles))
	for _, role := range roles {
		roleIDs = append(roleIDs, role.ID)
	}

	return sortedUint64s(roleIDs), nil
}

func getRBACRecordByID[T any](
	ctx context.Context,
	repository rbacstore.Repository,
	id uint64,
	action string,
	fetch func(rbacstore.Repository, context.Context, uint64) (T, error),
) (T, error) {
	return runNamedRBACQuery(repository, action, func(repo rbacstore.Repository) (T, error) {
		return fetch(repo, ctx, id)
	})
}

func listRBACRecords[Filter any, Record any](
	ctx context.Context,
	repository rbacstore.Repository,
	filter Filter,
	action string,
	fetch func(rbacstore.Repository, context.Context, Filter) ([]Record, error),
) ([]Record, error) {
	return runNamedRBACQuery(repository, action, func(repo rbacstore.Repository) ([]Record, error) {
		return fetch(repo, ctx, filter)
	})
}

func roleName(role rbacstore.Role) string {
	return role.Name
}

func permissionCode(permission rbacstore.Permission) string {
	return permission.Code
}

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

func stableUint64s(values []uint64) []uint64 {
	stable := make([]uint64, 0, len(values))
	seen := make(map[uint64]struct{}, len(values))
	for _, value := range values {
		if value == 0 {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}

		seen[value] = struct{}{}
		stable = append(stable, value)
	}

	slices.Sort(stable)
	return stable
}

func sortedUint64s(values []uint64) []uint64 {
	sorted := append([]uint64(nil), values...)
	slices.Sort(sorted)
	return sorted
}

func roleSummaries(roles []rbacstore.Role) []moduleapi.RoleSummary {
	summaries := make([]moduleapi.RoleSummary, 0, len(roles))
	for _, role := range roles {
		summaries = append(summaries, moduleapi.RoleSummary{
			ID:      role.ID,
			Name:    strings.TrimSpace(role.Name),
			Display: strings.TrimSpace(role.Display),
		})
	}

	slices.SortFunc(summaries, func(left, right moduleapi.RoleSummary) int {
		if left.ID == right.ID {
			return strings.Compare(left.Name, right.Name)
		}
		if left.ID < right.ID {
			return -1
		}
		return 1
	})

	return summaries
}
