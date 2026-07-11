import { describe, expect, it, vi } from 'vitest';

const requestMocks = vi.hoisted(() => ({ get: vi.fn() }));

vi.mock('@/utils/request', () => ({
  request: {
    get: requestMocks.get,
  },
}));

import { getLatestTaskForOwner } from './task';

describe('getLatestTaskForOwner', () => {
  it('uses the owner-scoped task list and returns its newest item', async () => {
    const latestTask = { id: 42, status: 'success' };
    requestMocks.get.mockResolvedValue({ items: [latestTask] });

    await expect(getLatestTaskForOwner({ ownerId: 'project-1', ownerType: 'compose_project' })).resolves.toBe(
      latestTask,
    );
    expect(requestMocks.get).toHaveBeenCalledWith({
      params: { limit: 1, owner_id: 'project-1', owner_type: 'compose_project' },
      url: '/api/tasks',
    });
  });
});
