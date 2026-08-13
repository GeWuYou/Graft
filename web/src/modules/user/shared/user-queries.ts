import { useQuery } from '@tanstack/vue-query';
import { computed } from 'vue';

import { queryClient } from '@/shared/query';

import { getUsers, type UserListQuery } from '../api/users';

const USER_QUERY_SCOPE = ['user'] as const;

export const userQueryKeys = {
  list: (query: UserListQuery) => [...USER_QUERY_SCOPE, 'list', query] as const,
};

/** 用户列表快照只由 Query cache 持有，页面不保留第二份服务端数据。 */
export function useUsersQuery(query: () => UserListQuery) {
  return useQuery(
    {
      queryKey: computed(() => userQueryKeys.list(query())),
      queryFn: () => getUsers(query()),
    },
    queryClient,
  );
}

/** 用户 mutation 会影响多个筛选页，因此统一失效整个用户列表命名空间。 */
export function invalidateUserListQueries() {
  return queryClient.invalidateQueries({ queryKey: USER_QUERY_SCOPE });
}
