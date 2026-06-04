import { describe, expect, it, vi } from 'vitest';

import { request } from '@/utils/request';

import { buildModuleRuntimeDetailApiPath, MONITOR_API_PATH } from '../contract/paths';
import { getModuleRuntimeDetail, getModuleRuntimeSnapshot } from './module-runtime';

vi.mock('@/utils/request', () => ({
  request: {
    get: vi.fn(),
  },
}));

describe('monitor module-runtime api', () => {
  it('calls the canonical module runtime snapshot path through request.ts', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValueOnce({ items: [], summary: {} } as never);

    await getModuleRuntimeSnapshot();

    expect(requestGet).toHaveBeenCalledWith({
      url: MONITOR_API_PATH.MODULE_RUNTIME,
    });
  });

  it('encodes module keys for module runtime detail reads', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValueOnce({ module_key: 'audit/log' } as never);

    await getModuleRuntimeDetail('audit/log');

    expect(requestGet).toHaveBeenCalledWith({
      url: buildModuleRuntimeDetailApiPath('audit/log'),
    });
    expect(buildModuleRuntimeDetailApiPath('audit/log')).toBe('/api/modules/runtime/audit%2Flog');
  });
});
