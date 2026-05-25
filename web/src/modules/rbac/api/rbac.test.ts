import { describe, expect, it, vi } from 'vitest';

import { request } from '@/utils/request';

import { RBAC_API_PATH } from '../contract/paths';
import { getPermissions } from './rbac';

vi.mock('@/utils/request', () => ({
  request: {
    get: vi.fn(),
    post: vi.fn(),
  },
}));

describe('rbac api', () => {
  it('calls the canonical permissions path through request.ts', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValueOnce({ items: [] } as never);

    await getPermissions();

    expect(requestGet).toHaveBeenCalledWith({
      url: RBAC_API_PATH.PERMISSIONS,
    });
  });
});
