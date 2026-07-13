import { describe, expect, it, vi } from 'vitest';

const requestMocks = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }));

vi.mock('@/utils/request', () => ({ request: requestMocks }));

import { discoverLocalDocker, getRuntimeTarget, listRuntimeTargets, refreshRuntimeTarget } from './runtime-target';

describe('listRuntimeTargets', () => {
  it('loads every selector page instead of truncating at the first 100 targets', async () => {
    requestMocks.get
      .mockResolvedValueOnce({
        items: Array.from({ length: 100 }, (_, id) => ({ id: id + 1 })),
        total: 101,
        limit: 100,
        offset: 0,
      })
      .mockResolvedValueOnce({
        items: [{ id: 101 }],
        total: 101,
        limit: 100,
        offset: 100,
      });

    const targets = await listRuntimeTargets();

    expect(targets).toHaveLength(101);
    expect(requestMocks.get).toHaveBeenNthCalledWith(1, {
      url: '/api/runtime-targets',
      params: { limit: 100, offset: 0 },
    });
    expect(requestMocks.get).toHaveBeenNthCalledWith(2, {
      url: '/api/runtime-targets',
      params: { limit: 100, offset: 100 },
    });
  });
});

describe('runtime target detail API', () => {
  it('uses the canonical detail, refresh, and Docker discovery routes', async () => {
    requestMocks.get.mockResolvedValueOnce({ id: 7 });
    requestMocks.post.mockResolvedValueOnce({ id: 7 }).mockResolvedValueOnce({ id: 7 });

    await getRuntimeTarget(7);
    await refreshRuntimeTarget(7);
    await discoverLocalDocker();

    expect(requestMocks.get).toHaveBeenCalledWith({ url: '/api/runtime-targets/7' });
    expect(requestMocks.post).toHaveBeenNthCalledWith(1, { url: '/api/runtime-targets/7/refresh' });
    expect(requestMocks.post).toHaveBeenNthCalledWith(2, { url: '/api/runtime-targets/discover-local-docker' });
  });
});
