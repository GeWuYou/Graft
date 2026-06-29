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

// runRBACQuery 执行一个 RBAC 仓库查询并处理仓库不可用错误。
// 当 repository 为 nil 时返回零值和 errRBACRepositoryUnavailable；查询失败时返回零值和原始错误。
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

// runNamedRBACQuery 执行带名称的 RBAC 查询，并在失败时包装错误信息。
// 当仓库不可用时，直接返回该错误；其他错误会附加 action 作为上下文。
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

// requireRBACUserService 确保用户服务可用。
// 当传入的用户服务为 nil 时返回 errRBACUserServiceUnavailable。
func requireRBACUserService(users moduleapi.UserService) (moduleapi.UserService, error) {
	if users == nil {
		return nil, errRBACUserServiceUnavailable
	}

	return users, nil
}

// listStableStringsByUserID 获取指定用户相关条目的稳定字符串列表。
// 它会先从仓储读取条目，再通过 extract 提取字符串，去除首尾空白、过滤空值、去重并排序后返回。
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

// listStableUserIDsByPermissionCode 根据权限码查询用户 ID，并返回去重、过滤零值且按升序排列的结果。
func listStableUserIDsByPermissionCode(ctx context.Context, repository rbacstore.Repository, permissionCode string) ([]uint64, error) {
	userIDs, err := runNamedRBACQuery(repository, fmt.Sprintf("list user ids by permission %q", permissionCode), func(repo rbacstore.Repository) ([]uint64, error) {
		return repo.ListUserIDsByPermissionCode(ctx, permissionCode)
	})
	if err != nil {
		return nil, err
	}

	return stableUint64s(userIDs), nil
}

// listRoleSummariesByUserIDs 获取每个用户 ID 对应的角色摘要列表，并保证输入中的每个用户 ID 都在结果中有键。
// 对于没有角色的用户，返回空切片。
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

// listRoleIDsByUserID 返回指定用户拥有的角色 ID，并按升序排序。
//
// 在查询前会校验用户服务可用，并确认用户存在。
//
// @returns 按升序排列的角色 ID 列表，或查询失败时返回错误。
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

// getRBACRecordByID 使用指定查询从 RBAC 仓储中按 ID 获取单条记录，并为错误添加操作名称上下文。
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

// action 参数用于包装查询错误时的操作名称，fetch 函数执行实际的检索操作。
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

// roleName 返回角色名称。
func roleName(role rbacstore.Role) string {
	return role.Name
}

// permissionCode 返回权限对象中的代码字段。
func permissionCode(permission rbacstore.Permission) string {
	return permission.Code
}

// stableStrings 提取、清理并稳定化字符串列表。
// 它会对每个元素应用提取函数，去除首尾空白，跳过空字符串，去重并按字典序排序后返回。
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

// stableUint64s 返回去重、过滤 0 且按升序排列的 uint64 列表。
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

// sortedUint64s 返回 values 的排序副本。
func sortedUint64s(values []uint64) []uint64 {
	sorted := append([]uint64(nil), values...)
	slices.Sort(sorted)
	return sorted
}

// roleSummaries 将角色转换为摘要并按 ID 和名称排序。
// 结果中的名称和显示文本会去除首尾空白。
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
