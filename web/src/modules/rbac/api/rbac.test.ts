import { describe, expect, it, vi } from 'vitest';

import { request } from '@/utils/request';

import { RBAC_API_PATH } from '../contract/paths';
import {
  assignRolePermissions,
  createRole,
  getPermissions,
  getRolePermissionBindings,
  getRoles,
  updateRole,
} from './rbac';

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

  it('calls the canonical roles path through request.ts', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValueOnce({ items: [] } as never);

    await getRoles();

    expect(requestGet).toHaveBeenCalledWith({
      url: RBAC_API_PATH.ROLES,
    });
  });

  it('calls the canonical role-permissions path through request.ts', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValueOnce({ permission_ids: [] } as never);

    await getRolePermissionBindings(42);

    expect(requestGet).toHaveBeenCalledWith({
      url: RBAC_API_PATH.ROLE_PERMISSIONS(42),
    });
  });

  it('calls the canonical role-permission-assign path through request.ts', async () => {
    const requestPost = vi.mocked(request.post);
    const payload = { permission_ids: [2, 3] };
    requestPost.mockResolvedValueOnce(null as never);

    await assignRolePermissions(42, payload);

    expect(requestPost).toHaveBeenCalledWith({
      url: RBAC_API_PATH.ROLE_PERMISSION_ASSIGN(42),
      data: payload,
    });
  });

  it('calls the canonical role-create path through request.ts', async () => {
    const requestPost = vi.mocked(request.post);
    const payload = { name: 'admin', display: 'Admin', description: 'system' };
    requestPost.mockResolvedValueOnce({ id: 1, ...payload, builtin: false } as never);

    await createRole(payload);

    expect(requestPost).toHaveBeenCalledWith({
      url: RBAC_API_PATH.ROLES,
      data: payload,
    });
  });

  it('calls the canonical role-update path through request.ts', async () => {
    const requestPost = vi.mocked(request.post);
    const payload = { name: 'editor', display: 'Editor', description: 'updated' };
    requestPost.mockResolvedValueOnce({ id: 42, ...payload, builtin: false } as never);

    await updateRole(42, payload);

    expect(requestPost).toHaveBeenCalledWith({
      url: RBAC_API_PATH.ROLE_UPDATE(42),
      data: payload,
    });
  });
});
