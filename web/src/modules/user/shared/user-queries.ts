import { useQuery } from '@tanstack/vue-query';

import { queryClient } from '@/shared/query';

import { getUsers } from '../api/users';
import type { UserListItem, UserListResponse } from '../types/user';

const USER_QUERY_SCOPE = ['user'] as const;

export const userQueryKeys = {
  list: () => [...USER_QUERY_SCOPE, 'list'] as const,
};

/** 用户列表快照只由 Query cache 持有，页面不保留第二份服务端数据。 */
export function useUsersQuery() {
  return useQuery(
    {
      queryKey: userQueryKeys.list(),
      queryFn: getUsers,
    },
    queryClient,
  );
}

/**
 * updateUserListCache 将已确认的用户 mutation 精确写回当前列表快照。
 *
 * 调用方只更新 API 已影响的条目，避免以页面局部 ref 覆盖或复制 Query cache。
 */
export function updateUserListCache(updateItems: (items: UserListItem[]) => UserListItem[]) {
  queryClient.setQueryData<UserListResponse>(userQueryKeys.list(), (current) => {
    if (!current) {
      return current;
    }

    return {
      ...current,
      items: updateItems(current.items),
    };
  });
}
