import { describe, expect, it, vi } from 'vitest';

import { buildOpenApiRuntimePath, OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import { request } from '@/utils/request';

import {
  createUser,
  deleteUser,
  getUserById,
  getUsers,
  resetUserPassword,
  updateUser,
  updateUserStatus,
} from './users';

vi.mock('@/utils/request', () => ({
  request: {
    get: vi.fn(),
    post: vi.fn(),
  },
}));

describe('users api', () => {
  it('calls the canonical users-list path through request.ts', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValueOnce({
      items: [
        {
          id: 1,
          username: 'alice',
          display: 'Alice',
          status: 'enabled',
          roles: [{ id: 2, name: 'admin', display: 'Admin' }],
          created_at: '',
          updated_at: '',
        },
      ],
    } as never);

    await getUsers();

    expect(requestGet).toHaveBeenCalledWith({
      url: OPENAPI_RUNTIME_PATH.getUsers,
    });
  });

  it('calls the canonical user-detail path through request.ts', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValueOnce({
      id: 1,
      username: 'alice',
      display: 'Alice',
      status: 'enabled',
      roles: [],
      created_at: '',
      updated_at: '',
    } as never);

    await getUserById(1);

    expect(requestGet).toHaveBeenCalledWith({
      url: buildOpenApiRuntimePath('getUserById', { id: 1 }),
    });
  });

  it('calls the canonical user-create path through request.ts', async () => {
    const requestPost = vi.mocked(request.post);
    const payload = { username: 'alice', display: 'Alice', password: 'Password1234' };
    requestPost.mockResolvedValueOnce({ id: 1, ...payload, status: 'enabled' } as never);

    await createUser(payload);

    expect(requestPost).toHaveBeenCalledWith({
      url: OPENAPI_RUNTIME_PATH.postUsers,
      data: payload,
    });
  });

  it('calls the canonical user-update path through request.ts', async () => {
    const requestPost = vi.mocked(request.post);
    const payload = { username: 'alice', display: 'Alice Updated' };
    requestPost.mockResolvedValueOnce({ id: 1, ...payload, status: 'enabled' } as never);

    await updateUser(1, payload);

    expect(requestPost).toHaveBeenCalledWith({
      url: buildOpenApiRuntimePath('postUserUpdate', { id: 1 }),
      data: payload,
    });
  });

  it('calls the canonical user-status path through request.ts', async () => {
    const requestPost = vi.mocked(request.post);
    const payload = { status: 'disabled' as const };
    requestPost.mockResolvedValueOnce({ id: 1, username: 'alice', display: 'Alice', status: 'disabled' } as never);

    await updateUserStatus(1, payload);

    expect(requestPost).toHaveBeenCalledWith({
      url: buildOpenApiRuntimePath('postUserStatus', { id: 1 }),
      data: payload,
    });
  });

  it('calls the canonical reset-password path through request.ts', async () => {
    const requestPost = vi.mocked(request.post);
    const payload = { new_password: 'Password12345' };
    requestPost.mockResolvedValueOnce(null as never);

    await resetUserPassword(1, payload);

    expect(requestPost).toHaveBeenCalledWith({
      url: buildOpenApiRuntimePath('postUserResetPassword', { id: 1 }),
      data: payload,
    });
  });

  it('calls the canonical user-delete path through request.ts', async () => {
    const requestPost = vi.mocked(request.post);
    requestPost.mockResolvedValueOnce(null as never);

    await deleteUser(1);

    expect(requestPost).toHaveBeenCalledWith({
      url: buildOpenApiRuntimePath('postUserDelete', { id: 1 }),
    });
  });
});
