import { beforeEach, describe, expect, it } from 'vitest';

import { queryClient } from '@/shared/query';

import type { NotificationItem, NotificationListResponse } from '../types/notification';
import {
  invalidateNotificationListQueries,
  normalizeNotificationListQuery,
  notificationQueryKeys,
  updateNotificationListCaches,
} from './notification-list-query';

function notification(overrides: Partial<NotificationItem> = {}): NotificationItem {
  return {
    category: 'TASK',
    delivery_created_at: '2026-06-11T10:47:21Z',
    delivery_id: 8,
    event_id: 1,
    event_type: 'task_succeeded',
    message: 'Scheduled task succeeded.',
    navigation: { kind: 'SCHEDULER_RUN', payload: {} },
    occurred_at: '2026-06-11T10:47:21Z',
    severity: 'info',
    source_module: 'scheduler',
    status: 'unread',
    target_ref: '1',
    target_type: 'USER',
    title: 'Scheduled task succeeded',
    ...overrides,
  };
}

describe('notification list queries', () => {
  beforeEach(() => {
    queryClient.clear();
  });

  it('normalizes optional filters and pagination into a stable module key', () => {
    expect(normalizeNotificationListQuery({ page: 0, page_size: 0, status: 'all' })).toEqual({
      page: 1,
      page_size: 20,
    });
    expect(normalizeNotificationListQuery({ page: 2, page_size: 10 })).toEqual({ page: 2, page_size: 10 });
    expect(notificationQueryKeys.list(normalizeNotificationListQuery({ source_module: 'scheduler' }))).toEqual([
      'notification',
      'list',
      { page: 1, page_size: 20, source_module: 'scheduler' },
    ]);
  });

  it('updates only cached pages that contain the confirmed mutation response', () => {
    const query = normalizeNotificationListQuery({ status: 'all' });
    const current: NotificationListResponse = {
      items: [notification()],
      page: 1,
      page_size: 20,
      total: 1,
    };
    queryClient.setQueryData(notificationQueryKeys.list(query), current);

    updateNotificationListCaches(notification({ read_at: '2026-06-11T11:00:00Z', status: 'read' }));

    expect(
      queryClient.getQueryData<NotificationListResponse>(notificationQueryKeys.list(query))?.items[0]?.status,
    ).toBe('read');
  });

  it('invalidates all module list variants without clearing unrelated server data', async () => {
    const notificationKey = notificationQueryKeys.list(normalizeNotificationListQuery({ status: 'unread' }));
    queryClient.setQueryData(notificationKey, { items: [], page: 1, page_size: 20, total: 0 });
    queryClient.setQueryData(['user', 'list'], { items: [] });

    await invalidateNotificationListQueries();

    expect(queryClient.getQueryState(notificationKey)?.isInvalidated).toBe(true);
    expect(queryClient.getQueryData(['user', 'list'])).toEqual({ items: [] });
  });
});
