import { describe, expect, it, vi } from 'vitest';

import { request } from '@/utils/request';

import { USER_API_PATH } from '../contract/paths';
import { createUser, updateUser } from './users';

vi.mock('@/utils/request', () => ({
  request: {
    get: vi.fn(),
    post: vi.fn(),
  },
}));

describe('users api', () => {
  it('calls the canonical user-create path through request.ts', async () => {
    const requestPost = vi.mocked(request.post);
    const payload = { username: 'alice', display: 'Alice', password: 'Password1234' };
    requestPost.mockResolvedValueOnce({ id: 1, ...payload, status: 'enabled' } as never);

    await createUser(payload);

    expect(requestPost).toHaveBeenCalledWith({
      url: USER_API_PATH.USERS,
      data: payload,
    });
  });

  it('calls the canonical user-update path through request.ts', async () => {
    const requestPost = vi.mocked(request.post);
    const payload = { username: 'alice', display: 'Alice Updated' };
    requestPost.mockResolvedValueOnce({ id: 1, ...payload, status: 'enabled' } as never);

    await updateUser(1, payload);

    expect(requestPost).toHaveBeenCalledWith({
      url: USER_API_PATH.USER_UPDATE(1),
      data: payload,
    });
  });
});
