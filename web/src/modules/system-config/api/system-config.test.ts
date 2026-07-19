import { beforeEach, describe, expect, it, vi } from 'vitest';

import { buildOpenApiRuntimePath, OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import { request } from '@/utils/request';

import { getSystemConfig, getSystemConfigs, resetSystemConfig, updateSystemConfig } from './system-config';

vi.mock('@/utils/request', () => ({
  request: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
  },
}));

describe('system config api', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('reads the canonical system config collection path', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValueOnce({ items: [], total: 0 } as never);

    await getSystemConfigs();

    expect(requestGet).toHaveBeenCalledWith({
      url: OPENAPI_RUNTIME_PATH.getSystemConfigs,
    });
  });

  it('encodes config keys for detail reads', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValueOnce({ key: 'scheduler/defaults' } as never);

    await getSystemConfig('scheduler/defaults');

    expect(requestGet).toHaveBeenCalledWith({
      url: buildOpenApiRuntimePath('getSystemConfig', { key: 'scheduler/defaults' }),
    });
  });

  it('puts override values through the canonical detail path', async () => {
    const requestPut = vi.mocked(request.put);
    const payload = { value: { retentionDays: 30 } };
    requestPut.mockResolvedValueOnce({ key: 'logging/defaults' } as never);

    await updateSystemConfig('logging/defaults', payload);

    expect(requestPut).toHaveBeenCalledWith({
      url: buildOpenApiRuntimePath('putSystemConfig', { key: 'logging/defaults' }),
      data: payload,
    });
  });

  it('posts override resets through the canonical reset path', async () => {
    const requestPost = vi.mocked(request.post);
    requestPost.mockResolvedValueOnce({ key: 'logging/defaults' } as never);

    await resetSystemConfig('logging/defaults');

    expect(requestPost).toHaveBeenCalledWith({
      url: buildOpenApiRuntimePath('postSystemConfigReset', { key: 'logging/defaults' }),
    });
  });
});
