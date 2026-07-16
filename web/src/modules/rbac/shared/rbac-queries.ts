import { useQuery } from '@tanstack/vue-query';
import { computed, type MaybeRef, toValue } from 'vue';

import { queryClient } from '@/shared/query';

import { getPermissions, getRoles } from '../api/rbac';
import type { RoleListItem, RoleListResponse } from '../contract/role';
import type { PermissionFilters } from '../types/permission';

const RBAC_QUERY_SCOPE = ['rbac'] as const;

export type NormalizedPermissionFilters = Required<PermissionFilters>;

export const rbacQueryKeys = {
  permissionCatalog: () => [...RBAC_QUERY_SCOPE, 'permission-catalog'] as const,
  permissionListScope: () => [...RBAC_QUERY_SCOPE, 'permissions'] as const,
  permissionList: (filters: NormalizedPermissionFilters) => [...RBAC_QUERY_SCOPE, 'permissions', filters] as const,
  roles: () => [...RBAC_QUERY_SCOPE, 'roles'] as const,
};

/** 将可选筛选条件规范化，避免同一权限快照因空字段产生多个缓存键。 */
export function normalizePermissionFilters(filters?: PermissionFilters): NormalizedPermissionFilters {
  return { keyword: filters?.keyword?.trim() ?? '', module: filters?.module?.trim() ?? '' };
}

function toPermissionRequestFilters(filters: NormalizedPermissionFilters): PermissionFilters {
  return {
    ...(filters.keyword ? { keyword: filters.keyword } : {}),
    ...(filters.module ? { module: filters.module } : {}),
  };
}

/** 角色列表快照只由 Query cache 持有，抽屉草稿和筛选仍由页面管理。 */
export function useRolesQuery() {
  return useQuery({ queryKey: rbacQueryKeys.roles(), queryFn: getRoles }, queryClient);
}

/** 角色编辑器所需的权限目录只有具备读取权限时才请求。 */
export function usePermissionCatalogQuery(enabled: MaybeRef<boolean>) {
  return useQuery(
    {
      queryKey: rbacQueryKeys.permissionCatalog(),
      queryFn: () => getPermissions(),
      enabled: computed(() => toValue(enabled)),
    },
    queryClient,
  );
}

/** 权限列表按规范化服务器筛选条件缓存；分页和列显示保持页面本地。 */
export function usePermissionListQuery(filters: MaybeRef<PermissionFilters>) {
  return useQuery(
    {
      queryKey: computed(() => rbacQueryKeys.permissionList(normalizePermissionFilters(toValue(filters)))),
      queryFn: ({ queryKey }) => getPermissions(toPermissionRequestFilters(queryKey[2])),
    },
    queryClient,
  );
}

/** 将确认后的角色 mutation 响应精确写回列表快照。 */
export function updateRoleListCache(updateItems: (items: RoleListItem[]) => RoleListItem[]) {
  queryClient.setQueryData<RoleListResponse>(rbacQueryKeys.roles(), (current) =>
    current ? { ...current, items: updateItems(current.items) } : current,
  );
}

/** 角色权限绑定会同时改变角色摘要和权限绑定计数，失效相关快照以恢复服务端权威数据。 */
export function invalidateRolesQuery() {
  return Promise.all([
    queryClient.invalidateQueries({ queryKey: rbacQueryKeys.roles() }),
    queryClient.invalidateQueries({ queryKey: rbacQueryKeys.permissionCatalog() }),
    queryClient.invalidateQueries({ queryKey: rbacQueryKeys.permissionListScope() }),
  ]);
}
