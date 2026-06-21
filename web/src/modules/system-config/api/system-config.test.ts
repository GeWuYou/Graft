import { beforeEach, describe, expect, it, vi } from 'vitest';

import { request } from '@/utils/request';

import {
  buildSystemConfigDetailApiPath,
  buildSystemConfigResetApiPath,
  SYSTEM_CONFIG_API_PATH,
} from '../contract/paths';
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
      url: SYSTEM_CONFIG_API_PATH.LIST,
    });
  });

  it('encodes config keys for detail reads', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValueOnce({ key: 'scheduler/defaults' } as never);

    await getSystemConfig('scheduler/defaults');

    expect(requestGet).toHaveBeenCalledWith({
      url: buildSystemConfigDetailApiPath('scheduler/defaults'),
    });
    expect(buildSystemConfigDetailApiPath('scheduler/defaults')).toBe('/api/system-configs/scheduler%2Fdefaults');
  });

  it('puts override values through the canonical detail path', async () => {
    const requestPut = vi.mocked(request.put);
    const payload = { value: { retentionDays: 30 } };
    requestPut.mockResolvedValueOnce({ key: 'logging/defaults' } as never);

    await updateSystemConfig('logging/defaults', payload);

    expect(requestPut).toHaveBeenCalledWith({
      url: buildSystemConfigDetailApiPath('logging/defaults'),
      data: payload,
    });
  });

  it('posts override resets through the canonical reset path', async () => {
    const requestPost = vi.mocked(request.post);
    requestPost.mockResolvedValueOnce({ key: 'logging/defaults' } as never);

    await resetSystemConfig('logging/defaults');

    expect(requestPost).toHaveBeenCalledWith({
      url: buildSystemConfigResetApiPath('logging/defaults'),
    });
    expect(buildSystemConfigResetApiPath('logging/defaults')).toBe('/api/system-configs/logging%2Fdefaults/reset');
  });
});
