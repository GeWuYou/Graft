import { describe, expect, it, vi } from 'vitest';

import { RBAC_API_PATH } from '@/modules/rbac/contract/paths';
import { request } from '@/utils/request';

import { assignUserRoles, getRoles, getUserRoleBindings } from './user-roles';

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
      url: RBAC_API_PATH.ROLES,
    });
  });

  it('calls the canonical user role binding path through request.ts', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValueOnce({ role_ids: [1, 3] } as never);

    await getUserRoleBindings(42);

    expect(requestGet).toHaveBeenCalledWith({
      url: RBAC_API_PATH.USER_ROLES(42),
    });
  });

  it('calls the canonical user role assign path through request.ts', async () => {
    const requestPost = vi.mocked(request.post);
    requestPost.mockResolvedValueOnce(null as never);

    await assignUserRoles(42, { role_ids: [1, 3] });

    expect(requestPost).toHaveBeenCalledWith({
      url: RBAC_API_PATH.USER_ROLE_ASSIGN(42),
      data: { role_ids: [1, 3] },
    });
  });
});
