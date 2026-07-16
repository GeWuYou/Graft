import { useMutation, useQuery } from '@tanstack/vue-query';
import { computed, type ComputedRef } from 'vue';

import { queryClient } from '@/shared/query';

import {
  getAnnouncementUnreadCount,
  getMyAnnouncements,
  markAllAnnouncementsRead,
  markAnnouncementRead,
} from '../api/announcement';
import type { MyAnnouncementListQuery } from '../types/announcement';

const ANNOUNCEMENT_QUERY_SCOPE = ['announcement'] as const;
const MY_ANNOUNCEMENTS_QUERY_SCOPE = [...ANNOUNCEMENT_QUERY_SCOPE, 'my'] as const;

const announcementQueryKeys = {
  myAnnouncementList: (query: MyAnnouncementListQuery) =>
    [
      ...MY_ANNOUNCEMENTS_QUERY_SCOPE,
      'list',
      query.page ?? 1,
      query.page_size ?? 20,
      query.unread_only === true,
    ] as const,
  myAnnouncementLists: () => [...MY_ANNOUNCEMENTS_QUERY_SCOPE, 'list'] as const,
  unreadCount: () => [...MY_ANNOUNCEMENTS_QUERY_SCOPE, 'unread-count'] as const,
};

/** 已读状态变化后失效当前账号的公告列表与未读数，恢复服务端权威快照。 */
export async function invalidateMyAnnouncementQueries() {
  await Promise.all([
    queryClient.invalidateQueries({ queryKey: announcementQueryKeys.myAnnouncementLists() }),
    queryClient.invalidateQueries({ queryKey: announcementQueryKeys.unreadCount() }),
  ]);
}

export function useAnnouncementUnreadCountQuery() {
  return useQuery(
    {
      queryKey: announcementQueryKeys.unreadCount(),
      queryFn: getAnnouncementUnreadCount,
    },
    queryClient,
  );
}

export function useMyAnnouncementsQuery(query: ComputedRef<MyAnnouncementListQuery>) {
  return useQuery(
    {
      queryKey: computed(() => announcementQueryKeys.myAnnouncementList(query.value)),
      queryFn: () => getMyAnnouncements(query.value),
    },
    queryClient,
  );
}

export function useMarkAnnouncementReadMutation() {
  return useMutation(
    {
      mutationFn: (id: number) => markAnnouncementRead(id),
      onSuccess: invalidateMyAnnouncementQueries,
    },
    queryClient,
  );
}

export function useMarkAllAnnouncementsReadMutation() {
  return useMutation(
    {
      mutationFn: () => markAllAnnouncementsRead(),
      onSuccess: invalidateMyAnnouncementQueries,
    },
    queryClient,
  );
}
