import { describe, expect, it, vi } from 'vitest';

import { buildOpenApiRuntimePath, OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';

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
      url: OPENAPI_RUNTIME_PATH.getRuntimeTargets,
      params: { limit: 100, offset: 0 },
    });
    expect(requestMocks.get).toHaveBeenNthCalledWith(2, {
      url: OPENAPI_RUNTIME_PATH.getRuntimeTargets,
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

    expect(requestMocks.get).toHaveBeenCalledWith({ url: buildOpenApiRuntimePath('getRuntimeTarget', { id: 7 }) });
    expect(requestMocks.post).toHaveBeenNthCalledWith(1, {
      url: buildOpenApiRuntimePath('postRuntimeTargetRefresh', { id: 7 }),
    });
    expect(requestMocks.post).toHaveBeenNthCalledWith(2, {
      url: OPENAPI_RUNTIME_PATH.postRuntimeTargetsDiscoverLocalDocker,
    });
  });
});
