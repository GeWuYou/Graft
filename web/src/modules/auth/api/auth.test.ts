import { describe, expect, it, vi } from 'vitest';

import { request } from '@/utils/request';

import { AUTH_API_PATH } from '../contract/paths';
import { getBootstrap, login } from './auth';

vi.mock('@/utils/request', () => ({
  request: {
    get: vi.fn(),
    post: vi.fn(),
  },
}));

describe('auth api', () => {
  it('calls the canonical login path through request.ts', async () => {
    const requestPost = vi.mocked(request.post);
    const payload = { username: 'graft', password: 'admin' };
    requestPost.mockResolvedValueOnce({ access_token: 'token' } as never);

    await login(payload);

    expect(requestPost).toHaveBeenCalledWith({
      url: AUTH_API_PATH.LOGIN,
      data: payload,
    });
  });

  it('calls the canonical bootstrap path through request.ts', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValueOnce({ user: { id: 1, username: 'graft', display_name: 'Graft' } } as never);

    await getBootstrap();

    expect(requestGet).toHaveBeenCalledWith({
      url: AUTH_API_PATH.BOOTSTRAP,
    });
  });
});
