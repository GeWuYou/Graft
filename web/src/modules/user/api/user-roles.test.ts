import { describe, expect, it, vi } from 'vitest';

import { buildOpenApiRuntimePath, OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import { request } from '@/utils/request';

import { getRoles, getUserRoleBindings, mutateBatchUserRoles, mutateUserRoles } from './user-roles';

vi.mock('@/utils/request', () => ({
  request: {
    get: vi.fn(),
    post: vi.fn(),
  },
}));

describe('user role api', () => {
  it('calls the canonical roles path through request.ts', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValueOnce([] as never);

    await getRoles();

    expect(requestGet).toHaveBeenCalledWith({
      url: OPENAPI_RUNTIME_PATH.getRoles,
    });
  });

  it('calls the canonical user role binding path through request.ts', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValueOnce({ role_ids: [1, 3] } as never);

    await getUserRoleBindings(42);

    expect(requestGet).toHaveBeenCalledWith({
      url: buildOpenApiRuntimePath('getUserRoles', { id: 42 }),
    });
  });

  it('calls the single-user replace path through request.ts', async () => {
    const requestPost = vi.mocked(request.post);
    requestPost.mockResolvedValueOnce(null as never);

    await mutateUserRoles(42, 'replace', { role_ids: [1, 3] });

    expect(requestPost).toHaveBeenCalledWith({
      url: buildOpenApiRuntimePath('postUserRolesReplace', { id: 42 }),
      data: { role_ids: [1, 3] },
    });
  });

  it('calls the single-user add path through request.ts', async () => {
    const requestPost = vi.mocked(request.post);
    requestPost.mockResolvedValueOnce(null as never);

    await mutateUserRoles(42, 'add', { role_ids: [2] });

    expect(requestPost).toHaveBeenCalledWith({
      url: buildOpenApiRuntimePath('postUserRolesAdd', { id: 42 }),
      data: { role_ids: [2] },
    });
  });

  it('calls the batch remove path through request.ts', async () => {
    const requestPost = vi.mocked(request.post);
    requestPost.mockResolvedValueOnce(null as never);

    await mutateBatchUserRoles('remove', { user_ids: [7, 9], role_ids: [2] });

    expect(requestPost).toHaveBeenCalledWith({
      url: OPENAPI_RUNTIME_PATH.postUsersRolesRemove,
      data: { user_ids: [7, 9], role_ids: [2] },
    });
  });
});
