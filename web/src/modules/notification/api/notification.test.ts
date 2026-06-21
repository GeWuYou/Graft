import { beforeEach, describe, expect, it, vi } from 'vitest';

import { request } from '@/utils/request';

import { buildNotificationDeleteApiPath, buildNotificationReadApiPath, NOTIFICATION_API_PATH } from '../contract/paths';
import {
  deleteNotification,
  getNotifications,
  getNotificationUnreadCount,
  markNotificationRead,
  markNotificationsReadAll,
} from './notification';

vi.mock('@/utils/request', () => ({
  request: {
    delete: vi.fn(),
    get: vi.fn(),
    post: vi.fn(),
  },
}));

describe('notification api', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('passes list filters to the canonical notification list path', async () => {
    const requestGet = vi.mocked(request.get);
    const query = { page: 2, page_size: 20, status: 'unread' as const };
    requestGet.mockResolvedValueOnce({ items: [], total: 0, page: 2, page_size: 20 } as never);

    await getNotifications(query);

    expect(requestGet).toHaveBeenCalledWith({
      url: NOTIFICATION_API_PATH.LIST,
      params: query,
    });
  });

  it('omits the UI-only all status from backend list params', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValueOnce({ items: [], total: 0, page: 1, page_size: 20 } as never);

    await getNotifications({ page: 1, page_size: 20, status: 'all' });

    expect(requestGet).toHaveBeenCalledWith({
      url: NOTIFICATION_API_PATH.LIST,
      params: { page: 1, page_size: 20 },
    });
  });

  it('reads unread count through the canonical count path', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValueOnce({ count: 3 } as never);

    await getNotificationUnreadCount();

    expect(requestGet).toHaveBeenCalledWith({
      url: NOTIFICATION_API_PATH.UNREAD_COUNT,
    });
  });

  it('marks one notification read through the delivery read path', async () => {
    const requestPost = vi.mocked(request.post);
    requestPost.mockResolvedValueOnce({ delivery_id: 42 } as never);

    await markNotificationRead(42);

    expect(requestPost).toHaveBeenCalledWith({
      url: buildNotificationReadApiPath(42),
    });
    expect(buildNotificationReadApiPath(42)).toBe('/api/notifications/42/read');
  });

  it('marks filtered notifications read through the canonical read-all path', async () => {
    const requestPost = vi.mocked(request.post);
    const payload = { severity: 'error' as const, source_module: 'scheduler' };
    requestPost.mockResolvedValueOnce({ updated_count: 2 } as never);

    await markNotificationsReadAll(payload);

    expect(requestPost).toHaveBeenCalledWith({
      url: NOTIFICATION_API_PATH.READ_ALL,
      data: payload,
    });
  });

  it('deletes one notification through the delivery delete path', async () => {
    const requestDelete = vi.mocked(request.delete);
    requestDelete.mockResolvedValueOnce({} as never);

    await deleteNotification(42);

    expect(requestDelete).toHaveBeenCalledWith({
      url: buildNotificationDeleteApiPath(42),
    });
    expect(buildNotificationDeleteApiPath(42)).toBe('/api/notifications/42');
  });
});
