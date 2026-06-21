import { describe, expect, it, vi } from 'vitest';

const requestMocks = vi.hoisted(() => ({
  delete: vi.fn(),
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
}));

vi.mock('@/utils/request', () => ({
  request: requestMocks,
}));

import { getAnnouncements, normalizeAnnouncementListQuery, normalizeMyAnnouncementListQuery } from './announcement';

describe('announcement API query mapping', () => {
  it('omits empty filters and preserves typed backend parameters', () => {
    expect(
      normalizeAnnouncementListQuery({
        keyword: '',
        level: undefined,
        page: 1,
        page_size: 20,
        pinned: false,
        sort: 'pinned_publish_desc',
        status: 'published',
      }),
    ).toEqual({
      page: 1,
      page_size: 20,
      pinned: false,
      sort: 'pinned_publish_desc',
      status: 'published',
    });
  });

  it('returns undefined for absent query objects', () => {
    expect(normalizeAnnouncementListQuery()).toBeUndefined();
  });

  it('maps current-user list query parameters without empty values', () => {
    expect(
      normalizeMyAnnouncementListQuery({
        page: 2,
        page_size: 10,
        unread_only: false,
      }),
    ).toEqual({
      page: 2,
      page_size: 10,
      unread_only: false,
    });
  });

  it('omits absent current-user list query parameters', () => {
    expect(normalizeMyAnnouncementListQuery()).toBeUndefined();
  });

  it('returns request promises without redundant promise casts', async () => {
    const response = { items: [], page: 1, page_size: 20, total: 0 };
    requestMocks.get.mockResolvedValueOnce(response);

    await expect(getAnnouncements({ page: 1 })).resolves.toBe(response);

    expect(requestMocks.get).toHaveBeenCalledWith({
      params: { page: 1 },
      url: '/api/announcements',
    });
  });
});
