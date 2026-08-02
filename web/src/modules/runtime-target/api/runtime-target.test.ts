import { describe, expect, it, vi } from 'vitest';

import { buildOpenApiRuntimePath, OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';

const requestMocks = vi.hoisted(() => ({ delete: vi.fn(), get: vi.fn(), post: vi.fn(), put: vi.fn() }));

vi.mock('@/utils/request', () => ({ request: requestMocks }));

import {
  deleteRuntimeTargetSavedView,
  discoverLocalDocker,
  getRuntimeTarget,
  listRuntimeTargets,
  postRuntimeTargetSavedView,
  putRuntimeTargetSavedView,
  refreshRuntimeTarget,
} from './runtime-target';

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

it('maps saved-view update and delete paths through generated placeholders', async () => {
  requestMocks.put.mockResolvedValueOnce({ id: 3 });
  requestMocks.delete.mockResolvedValueOnce(undefined);

  await putRuntimeTargetSavedView(3, {
    name: 'Healthy Edge',
    pageSize: 20,
    queryState: { health: 'healthy' },
    visibleColumns: ['health'],
    isDefault: false,
  });
  await deleteRuntimeTargetSavedView(3);

  expect(requestMocks.put).toHaveBeenCalledWith({
    url: buildOpenApiRuntimePath('putRuntimeTargetSavedView', { viewId: 3 }),
    data: {
      name: 'Healthy Edge',
      page_size: 20,
      query_state: { health: 'healthy' },
      visible_columns: ['health'],
      is_default: false,
    },
  });
  expect(requestMocks.delete).toHaveBeenCalledWith({
    url: buildOpenApiRuntimePath('deleteRuntimeTargetSavedView', { viewId: 3 }),
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

describe('runtime target saved view API', () => {
  it('maps page state to the generated saved-view request payload', async () => {
    requestMocks.post.mockResolvedValueOnce({ id: 3 });

    await postRuntimeTargetSavedView({
      name: 'Healthy Edge',
      pageSize: 50,
      queryState: { health: 'healthy', sort: 'health:desc' },
      visibleColumns: ['health', 'provider'],
      isDefault: true,
    });

    expect(requestMocks.post).toHaveBeenCalledWith({
      url: OPENAPI_RUNTIME_PATH.postRuntimeTargetSavedView,
      data: {
        name: 'Healthy Edge',
        page_size: 50,
        query_state: { health: 'healthy', sort: 'health:desc' },
        visible_columns: ['health', 'provider'],
        is_default: true,
      },
    });
  });
});
