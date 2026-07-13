import { describe, expect, it, vi } from 'vitest';

const requestMocks = vi.hoisted(() => ({ get: vi.fn() }));

vi.mock('@/utils/request', () => ({ request: requestMocks }));

import { listRuntimeTargets } from './runtime-target';

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
