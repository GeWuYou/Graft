import { describe, expect, it, vi } from 'vitest';

import { buildOpenApiRuntimePath, OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import { request } from '@/utils/request';

import {
  addRolePermissions,
  createRole,
  deletePermissionSavedView,
  deleteRole,
  deleteRoleSavedView,
  getPermissionDetail,
  getPermissions,
  getPermissionSavedViews,
  getRoleDetail,
  getRolePermissionBindings,
  getRoles,
  getRoleSavedViews,
  postPermissionSavedView,
  postRoleSavedView,
  putPermissionSavedView,
  putRoleSavedView,
  removeRolePermissions,
  replaceRolePermissions,
  updateRole,
  updateRoleStatus,
} from './rbac';

vi.mock('@/utils/request', () => ({
  request: {
    delete: vi.fn(),
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
  },
}));

describe('rbac api', () => {
  it('calls the canonical permissions path through request.ts', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValueOnce({ items: [] } as never);

    await getPermissions();

    expect(requestGet).toHaveBeenCalledWith({
      url: OPENAPI_RUNTIME_PATH.getPermissions,
      params: undefined,
    });
  });

  it('calls the canonical roles path through request.ts', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValueOnce({ items: [] } as never);

    await getRoles();

    expect(requestGet).toHaveBeenCalledWith({
      url: OPENAPI_RUNTIME_PATH.getRoles,
    });
  });

  it('calls the canonical role-permissions path through request.ts', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValueOnce({ permission_ids: [] } as never);

    await getRolePermissionBindings(42);

    expect(requestGet).toHaveBeenCalledWith({
      url: buildOpenApiRuntimePath('getRolePermissions', { id: 42 }),
    });
  });

  it('calls the canonical role detail path through request.ts', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValueOnce({ id: 42 } as never);

    await getRoleDetail(42);

    expect(requestGet).toHaveBeenCalledWith({
      url: buildOpenApiRuntimePath('getRole', { id: 42 }),
    });
  });

  it('calls the canonical permission detail path through request.ts', async () => {
    const requestGet = vi.mocked(request.get);
    requestGet.mockResolvedValueOnce({ id: 7 } as never);

    await getPermissionDetail(7);

    expect(requestGet).toHaveBeenCalledWith({
      url: buildOpenApiRuntimePath('getPermission', { id: 7 }),
    });
  });

  it('calls the canonical role-permissions replace path through request.ts', async () => {
    const requestPost = vi.mocked(request.post);
    const payload = { permission_ids: [2, 3] };
    requestPost.mockResolvedValueOnce(null as never);

    await replaceRolePermissions(42, payload);

    expect(requestPost).toHaveBeenCalledWith({
      url: buildOpenApiRuntimePath('postRolePermissionsReplace', { id: 42 }),
      data: payload,
    });
  });

  it('calls the canonical role-permissions add path through request.ts', async () => {
    const requestPost = vi.mocked(request.post);
    const payload = { permission_ids: [2, 3] };
    requestPost.mockResolvedValueOnce(null as never);

    await addRolePermissions(42, payload);

    expect(requestPost).toHaveBeenCalledWith({
      url: buildOpenApiRuntimePath('postRolePermissionsAdd', { id: 42 }),
      data: payload,
    });
  });

  it('calls the canonical role-permissions remove path through request.ts', async () => {
    const requestDelete = vi.mocked(request.delete);
    const payload = { permission_ids: [2, 3] };
    requestDelete.mockResolvedValueOnce({ operation_id: 'request-1' } as never);

    await removeRolePermissions(42, payload);

    expect(requestDelete).toHaveBeenCalledWith({
      url: buildOpenApiRuntimePath('deleteRolePermissions', { id: 42 }),
      data: payload,
    });
  });

  it('calls the canonical role status path through request.ts', async () => {
    const requestPost = vi.mocked(request.post);
    const payload = { status: 'disabled' as const };
    requestPost.mockResolvedValueOnce({ id: 42, status: 'disabled' } as never);

    await updateRoleStatus(42, payload);

    expect(requestPost).toHaveBeenCalledWith({
      url: buildOpenApiRuntimePath('postRoleStatus', { id: 42 }),
      data: payload,
    });
  });

  it('calls the canonical role delete path through request.ts', async () => {
    const requestPost = vi.mocked(request.post);
    requestPost.mockResolvedValueOnce(null as never);

    await deleteRole(42);

    expect(requestPost).toHaveBeenCalledWith({
      url: buildOpenApiRuntimePath('postRoleDelete', { id: 42 }),
    });
  });

  it('calls the canonical role-create path through request.ts', async () => {
    const requestPost = vi.mocked(request.post);
    const payload = { name: 'admin', display: 'Admin', description: 'system' };
    requestPost.mockResolvedValueOnce({ id: 1, ...payload, builtin: false } as never);

    await createRole(payload);

    expect(requestPost).toHaveBeenCalledWith({
      url: OPENAPI_RUNTIME_PATH.postRoles,
      data: payload,
    });
  });

  it('calls the canonical role-update path through request.ts', async () => {
    const requestPost = vi.mocked(request.post);
    const payload = { name: 'editor', display: 'Editor', description: 'updated' };
    requestPost.mockResolvedValueOnce({ id: 42, ...payload, builtin: false } as never);

    await updateRole(42, payload);

    expect(requestPost).toHaveBeenCalledWith({
      url: buildOpenApiRuntimePath('postRoleUpdate', { id: 42 }),
      data: payload,
    });
  });

  it('calls the canonical role saved-view paths through request.ts', async () => {
    const requestGet = vi.mocked(request.get);
    const requestPost = vi.mocked(request.post);
    const requestPut = vi.mocked(request.put);
    const requestDelete = vi.mocked(request.delete);
    const payload = {
      name: 'System roles',
      page_size: 25,
      query_state: { keyword: 'admin', type: 'builtin' },
      visible_columns: ['role', 'builtin'],
      is_default: false,
    };
    requestGet.mockResolvedValueOnce({ items: [] } as never);
    requestPost.mockResolvedValueOnce({ id: 7, ...payload } as never);
    requestPut.mockResolvedValueOnce({ id: 7, ...payload } as never);
    requestDelete.mockResolvedValueOnce(undefined as never);

    await getRoleSavedViews();
    await postRoleSavedView(payload);
    await putRoleSavedView(7, payload);
    await deleteRoleSavedView(7);

    expect(requestGet).toHaveBeenCalledWith({ url: OPENAPI_RUNTIME_PATH.getRoleSavedViews });
    expect(requestPost).toHaveBeenCalledWith({ url: OPENAPI_RUNTIME_PATH.postRoleSavedView, data: payload });
    expect(requestPut).toHaveBeenCalledWith({
      url: buildOpenApiRuntimePath('putRoleSavedView', { viewId: 7 }),
      data: payload,
    });
    expect(requestDelete).toHaveBeenCalledWith({
      url: buildOpenApiRuntimePath('deleteRoleSavedView', { viewId: 7 }),
    });
  });

  it('calls the canonical permission saved-view paths through request.ts', async () => {
    const requestGet = vi.mocked(request.get);
    const requestPost = vi.mocked(request.post);
    const requestPut = vi.mocked(request.put);
    const requestDelete = vi.mocked(request.delete);
    const payload = {
      name: 'RBAC permissions',
      page_size: 25,
      query_state: { keyword: 'read', module: 'rbac' },
      visible_columns: ['permission', 'module'],
      is_default: false,
    };
    requestGet.mockResolvedValueOnce({ items: [] } as never);
    requestPost.mockResolvedValueOnce({ id: 8, ...payload } as never);
    requestPut.mockResolvedValueOnce({ id: 8, ...payload } as never);
    requestDelete.mockResolvedValueOnce(undefined as never);

    await getPermissionSavedViews();
    await postPermissionSavedView(payload);
    await putPermissionSavedView(8, payload);
    await deletePermissionSavedView(8);

    expect(requestGet).toHaveBeenCalledWith({ url: OPENAPI_RUNTIME_PATH.getPermissionSavedViews });
    expect(requestPost).toHaveBeenCalledWith({ url: OPENAPI_RUNTIME_PATH.postPermissionSavedView, data: payload });
    expect(requestPut).toHaveBeenCalledWith({
      url: buildOpenApiRuntimePath('putPermissionSavedView', { viewId: 8 }),
      data: payload,
    });
    expect(requestDelete).toHaveBeenCalledWith({
      url: buildOpenApiRuntimePath('deletePermissionSavedView', { viewId: 8 }),
    });
  });
});
