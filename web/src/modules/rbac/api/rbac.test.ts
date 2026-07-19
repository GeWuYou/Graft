import { describe, expect, it, vi } from 'vitest';

import { buildOpenApiRuntimePath, OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import { request } from '@/utils/request';

import {
  addRolePermissions,
  createRole,
  deleteRole,
  getPermissionDetail,
  getPermissions,
  getRoleDetail,
  getRolePermissionBindings,
  getRoles,
  removeRolePermissions,
  replaceRolePermissions,
  updateRole,
  updateRoleStatus,
} from './rbac';

const RBAC_API_PATH = {
  PERMISSIONS: OPENAPI_RUNTIME_PATH.getPermissions,
  ROLES: OPENAPI_RUNTIME_PATH.getRoles,
  ROLE_PERMISSIONS: (id: number) => buildOpenApiRuntimePath('getRolePermissions', { id }),
  ROLE_DETAIL: (id: number) => buildOpenApiRuntimePath('getRole', { id }),
  PERMISSION_DETAIL: (id: number) => buildOpenApiRuntimePath('getPermission', { id }),
  ROLE_PERMISSIONS_REPLACE: (id: number) => buildOpenApiRuntimePath('postRolePermissionsReplace', { id }),
  ROLE_PERMISSIONS_ADD: (id: number) => buildOpenApiRuntimePath('postRolePermissionsAdd', { id }),
  ROLE_PERMISSIONS_REMOVE: (id: number) => buildOpenApiRuntimePath('postRolePermissionsRemove', { id }),
  ROLE_STATUS: (id: number) => buildOpenApiRuntimePath('postRoleStatus', { id }),
  ROLE_DELETE: (id: number) => buildOpenApiRuntimePath('postRoleDelete', { id }),
  ROLE_UPDATE: (id: number) => buildOpenApiRuntimePath('postRoleUpdate', { id }),
};

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
      params: undefined,
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

  it('calls the canonical role detail path through request.ts', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValueOnce({ id: 42 } as never);

    await getRoleDetail(42);

    expect(requestGet).toHaveBeenCalledWith({
      url: RBAC_API_PATH.ROLE_DETAIL(42),
    });
  });

  it('calls the canonical permission detail path through request.ts', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValueOnce({ id: 7 } as never);

    await getPermissionDetail(7);

    expect(requestGet).toHaveBeenCalledWith({
      url: RBAC_API_PATH.PERMISSION_DETAIL(7),
    });
  });

  it('calls the canonical role-permissions replace path through request.ts', async () => {
    const requestPost = vi.mocked(request.post);
    const payload = { permission_ids: [2, 3] };
    requestPost.mockResolvedValueOnce(null as never);

    await replaceRolePermissions(42, payload);

    expect(requestPost).toHaveBeenCalledWith({
      url: RBAC_API_PATH.ROLE_PERMISSIONS_REPLACE(42),
      data: payload,
    });
  });

  it('calls the canonical role-permissions add path through request.ts', async () => {
    const requestPost = vi.mocked(request.post);
    const payload = { permission_ids: [2, 3] };
    requestPost.mockResolvedValueOnce(null as never);

    await addRolePermissions(42, payload);

    expect(requestPost).toHaveBeenCalledWith({
      url: RBAC_API_PATH.ROLE_PERMISSIONS_ADD(42),
      data: payload,
    });
  });

  it('calls the canonical role-permissions remove path through request.ts', async () => {
    const requestPost = vi.mocked(request.post);
    const payload = { permission_ids: [2, 3] };
    requestPost.mockResolvedValueOnce(null as never);

    await removeRolePermissions(42, payload);

    expect(requestPost).toHaveBeenCalledWith({
      url: RBAC_API_PATH.ROLE_PERMISSIONS_REMOVE(42),
      data: payload,
    });
  });

  it('calls the canonical role status path through request.ts', async () => {
    const requestPost = vi.mocked(request.post);
    const payload = { status: 'disabled' as const };
    requestPost.mockResolvedValueOnce({ id: 42, status: 'disabled' } as never);

    await updateRoleStatus(42, payload);

    expect(requestPost).toHaveBeenCalledWith({
      url: RBAC_API_PATH.ROLE_STATUS(42),
      data: payload,
    });
  });

  it('calls the canonical role delete path through request.ts', async () => {
    const requestPost = vi.mocked(request.post);
    requestPost.mockResolvedValueOnce(null as never);

    await deleteRole(42);

    expect(requestPost).toHaveBeenCalledWith({
      url: RBAC_API_PATH.ROLE_DELETE(42),
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
