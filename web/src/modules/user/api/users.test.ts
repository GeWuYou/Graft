import { describe, expect, it, vi } from 'vitest';

import { request } from '@/utils/request';

import { USER_API_PATH } from '../contract/paths';
import { createUser, resetUserPassword, updateUser, updateUserStatus } from './users';

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

  it('calls the canonical user-status path through request.ts', async () => {
    const requestPost = vi.mocked(request.post);
    const payload = { status: 'disabled' as const };
    requestPost.mockResolvedValueOnce({ id: 1, username: 'alice', display: 'Alice', status: 'disabled' } as never);

    await updateUserStatus(1, payload);

    expect(requestPost).toHaveBeenCalledWith({
      url: USER_API_PATH.USER_STATUS(1),
      data: payload,
    });
  });

  it('calls the canonical reset-password path through request.ts', async () => {
    const requestPost = vi.mocked(request.post);
    const payload = { new_password: 'Password12345' };
    requestPost.mockResolvedValueOnce(null as never);

    await resetUserPassword(1, payload);

    expect(requestPost).toHaveBeenCalledWith({
      url: USER_API_PATH.USER_RESET_PASSWORD(1),
      data: payload,
    });
  });
});
