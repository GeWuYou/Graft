import { afterEach, describe, expect, it, vi } from 'vitest';

import { buildOpenApiRuntimePath, OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import { request } from '@/utils/request';

import { getBackup, listBackups, submitBackup } from './backup';

vi.mock('@/utils/request', () => ({
  request: {
    get: vi.fn(),
    post: vi.fn(),
  },
}));

describe('platform backup api', () => {
  afterEach(() => {
    vi.clearAllMocks();
  });

  it('reads only the canonical safe history and detail endpoints', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValue({ items: [], limit: 20, offset: 0, total: 0 } as never);

    await listBackups({ limit: 20, offset: 0 });
    await getBackup(42);

    expect(requestGet).toHaveBeenNthCalledWith(1, {
      url: OPENAPI_RUNTIME_PATH.listPlatformBackups,
      params: { limit: 20, offset: 0 },
    });
    expect(requestGet).toHaveBeenNthCalledWith(2, {
      url: buildOpenApiRuntimePath('getPlatformBackup', { id: 42 }),
    });
  });

  it('submits the selected manual retention with a caller-generated idempotency key', async () => {
    const requestPost = vi.mocked(request.post);
    requestPost.mockResolvedValue({ status: 'pending', task_id: 8 } as never);

    await submitBackup({ retention: '30d' }, 'backup-request-1');

    expect(requestPost).toHaveBeenCalledWith({
      url: OPENAPI_RUNTIME_PATH.postPlatformBackup,
      data: { retention: '30d' },
      headers: { 'Idempotency-Key': 'backup-request-1' },
    });
  });
});
