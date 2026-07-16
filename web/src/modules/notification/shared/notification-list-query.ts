import { useQuery } from '@tanstack/vue-query';
import { computed, type MaybeRef, toValue } from 'vue';

import { queryClient } from '@/shared/query';

import { getNotifications } from '../api/notification';
import type { NotificationItem, NotificationListQuery, NotificationListResponse } from '../types/notification';

const NOTIFICATION_QUERY_SCOPE = ['notification'] as const;
const DEFAULT_PAGE = 1;
const DEFAULT_PAGE_SIZE = 20;

export type NormalizedNotificationListQuery = NotificationListQuery & {
  page: number;
  page_size: number;
};

export const notificationQueryKeys = {
  list: (query: NormalizedNotificationListQuery) => [...NOTIFICATION_QUERY_SCOPE, 'list', query] as const,
  lists: () => [...NOTIFICATION_QUERY_SCOPE, 'list'] as const,
};

/**
 * 通知列表 key 只包含已规范化的 API 输入，路由和筛选表单状态仍由页面拥有。
 */
export function normalizeNotificationListQuery(query: NotificationListQuery): NormalizedNotificationListQuery {
  const normalized: NormalizedNotificationListQuery = {
    page: query.page && query.page > 0 ? query.page : DEFAULT_PAGE,
    page_size: query.page_size && query.page_size > 0 ? query.page_size : DEFAULT_PAGE_SIZE,
  };

  if (query.status && query.status !== 'all') normalized.status = query.status;
  if (query.severity) normalized.severity = query.severity;
  if (query.category) normalized.category = query.category;
  if (query.source_module) normalized.source_module = query.source_module;
  if (query.occurred_from) normalized.occurred_from = query.occurred_from;
  if (query.occurred_to) normalized.occurred_to = query.occurred_to;

  return normalized;
}

/** 通知列表快照仅由 Query cache 持有，query function 继续通过模块 API 边界读取。 */
export function useNotificationsQuery(query: MaybeRef<NotificationListQuery>) {
  return useQuery(
    {
      queryKey: computed(() => notificationQueryKeys.list(normalizeNotificationListQuery(toValue(query)))),
      queryFn: ({ queryKey }) => getNotifications(queryKey[2]),
    },
    queryClient,
  );
}

/** 将单条状态 mutation 同步到缓存，并移除已不再符合筛选条件的列表项。 */
export function updateNotificationListCaches(updated: NotificationItem) {
  for (const [queryKey, current] of queryClient.getQueriesData<NotificationListResponse>({
    queryKey: notificationQueryKeys.lists(),
  })) {
    if (!current || !current.items.some((item) => item.delivery_id === updated.delivery_id)) continue;

    const query = queryKey[2] as NormalizedNotificationListQuery;
    const items = current.items.flatMap((item) => {
      if (item.delivery_id !== updated.delivery_id) return [item];
      return query.status === 'unread' && updated.status !== 'unread' ? [] : [updated];
    });

    queryClient.setQueryData(queryKey, {
      ...current,
      items,
      total: current.total - (items.length === current.items.length ? 0 : 1),
    });
  }
}

/** 批量或筛选结果变化时失效模块列表，避免以局部行操作伪造不完整分页快照。 */
export function invalidateNotificationListQueries() {
  return queryClient.invalidateQueries({ queryKey: notificationQueryKeys.lists() });
}
