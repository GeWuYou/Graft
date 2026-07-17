import { describe, expect, it, vi } from 'vitest';

const requestMocks = vi.hoisted(() => ({ get: vi.fn() }));

vi.mock('@/utils/request', () => ({
  request: {
    get: requestMocks.get,
  },
}));

import { getLatestTaskForOwner } from './latest-task';

describe('getLatestTaskForOwner', () => {
  it('uses the application-scoped task list and returns its newest item', async () => {
    const latestTask = { id: 42, status: 'success' };
    const ownerId = 'app_01KXN51K1SW5YXS684M445V1P5';
    requestMocks.get.mockResolvedValue({ items: [latestTask] });

    await expect(getLatestTaskForOwner({ ownerId, ownerType: 'application' })).resolves.toBe(latestTask);
    expect(requestMocks.get).toHaveBeenCalledWith({
      params: { limit: 1, owner_id: ownerId, owner_type: 'application' },
      url: '/api/tasks',
    });
  });
});
