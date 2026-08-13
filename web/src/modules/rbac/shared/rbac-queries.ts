import { useQuery } from '@tanstack/vue-query';
import { computed, type MaybeRef, toValue } from 'vue';

import { queryClient } from '@/shared/query';

import { getPermissions, getRoles } from '../api/rbac';
import type { PermissionFilters } from '../types/permission';
import type { PermissionListItem } from '../types/permission';

const RBAC_QUERY_SCOPE = ['rbac'] as const;

export type RoleFilters = {
  keyword?: string;
  builtin?: boolean;
  limit?: number;
  offset?: number;
};

export type NormalizedPermissionFilters = Required<PermissionFilters>;
export type NormalizedRoleFilters = {
  keyword: string;
  builtin?: boolean;
  limit: number;
  offset: number;
};

export const rbacQueryKeys = {
  permissionCatalog: () => [...RBAC_QUERY_SCOPE, 'permission-catalog'] as const,
  permissionListScope: () => [...RBAC_QUERY_SCOPE, 'permissions'] as const,
  permissionList: (filters: NormalizedPermissionFilters) => [...RBAC_QUERY_SCOPE, 'permissions', filters] as const,
  rolesScope: () => [...RBAC_QUERY_SCOPE, 'roles'] as const,
  roles: (filters: NormalizedRoleFilters = normalizeRoleFilters()) => [...RBAC_QUERY_SCOPE, 'roles', filters] as const,
};

/** 将可选筛选条件规范化，避免同一权限快照因空字段产生多个缓存键。 */
export function normalizePermissionFilters(filters?: PermissionFilters): NormalizedPermissionFilters {
  return {
    keyword: filters?.keyword?.trim() ?? '',
    module: filters?.module?.trim() ?? '',
    limit: filters?.limit ?? 20,
    offset: filters?.offset ?? 0,
  };
}

function normalizeRoleFilters(filters?: RoleFilters): NormalizedRoleFilters {
  return {
    keyword: filters?.keyword?.trim() ?? '',
    builtin: filters?.builtin,
    limit: filters?.limit ?? 20,
    offset: filters?.offset ?? 0,
  };
}

function toPermissionRequestFilters(filters: NormalizedPermissionFilters): PermissionFilters {
  return {
    ...(filters.keyword ? { keyword: filters.keyword } : {}),
    ...(filters.module ? { module: filters.module } : {}),
    limit: filters.limit,
    offset: filters.offset,
  };
}

function toRoleRequestFilters(filters: NormalizedRoleFilters): RoleFilters {
  return {
    ...(filters.keyword ? { keyword: filters.keyword } : {}),
    ...(filters.builtin === undefined ? {} : { builtin: filters.builtin }),
    limit: filters.limit,
    offset: filters.offset,
  };
}

/** 角色列表快照只由 Query cache 持有，抽屉草稿和筛选仍由页面管理。 */
export function useRolesQuery(filters: MaybeRef<RoleFilters> = {}) {
  return useQuery(
    {
      queryKey: computed(() => rbacQueryKeys.roles(normalizeRoleFilters(toValue(filters)))),
      queryFn: ({ queryKey }) => getRoles(toRoleRequestFilters(queryKey[2])),
    },
    queryClient,
  );
}

/** 角色编辑器所需的权限目录只有具备读取权限时才请求。 */
export function usePermissionCatalogQuery(enabled: MaybeRef<boolean>) {
  return useQuery(
    {
      queryKey: rbacQueryKeys.permissionCatalog(),
      queryFn: getCompletePermissionCatalog,
      enabled: computed(() => toValue(enabled)),
    },
    queryClient,
  );
}

async function getCompletePermissionCatalog() {
  const limit = 100;
  let offset = 0;
  let total = 0;
  const items: PermissionListItem[] = [];

  do {
    const response = await getPermissions({ limit, offset });
    items.push(...response.items);
    total = response.total;
    offset += response.items.length;
  } while (offset < total && offset > 0);

  return { items, total, limit, offset: 0 };
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

/** 角色写入后只失效角色列表快照，避免跨分页直接修改可能不完整的缓存页。 */
export function invalidateRolesQuery() {
  return queryClient.invalidateQueries({ queryKey: rbacQueryKeys.rolesScope() });
}

/** 角色权限绑定会同时改变角色摘要和权限绑定计数，失效相关快照以恢复服务端权威数据。 */
export function invalidateRolePermissionQueries() {
  return Promise.all([
    invalidateRolesQuery(),
    queryClient.invalidateQueries({ queryKey: rbacQueryKeys.permissionCatalog() }),
    queryClient.invalidateQueries({ queryKey: rbacQueryKeys.permissionListScope() }),
  ]);
}
