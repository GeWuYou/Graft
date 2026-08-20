import { buildOpenApiRuntimePath, OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import type { components, paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import type { RoleListItem, RoleListResponse } from '../contract/role';
import type { PermissionDetailResponse, PermissionFilters, PermissionListResponse } from '../types/permission';
import type {
  CloneRolePayload,
  CreateRolePayload,
  ReplaceRolePermissionsPayload,
  RoleDetailResponse,
  RolePermissionBindingResponse,
  RolePermissionMutationPayload,
  UpdateRolePayload,
  UpdateRoleStatusPayload,
} from '../types/rbac';

type PermissionsPath = typeof OPENAPI_RUNTIME_PATH.getPermissions;
type RolesPath = typeof OPENAPI_RUNTIME_PATH.getRoles;
type PermissionSavedViewsPath = typeof OPENAPI_RUNTIME_PATH.getPermissionSavedViews;
type RoleSavedViewsPath = typeof OPENAPI_RUNTIME_PATH.getRoleSavedViews;
type RolePermissionsPath = typeof OPENAPI_RUNTIME_PATH.getRolePermissions;
type RolePermissionsReplacePath = typeof OPENAPI_RUNTIME_PATH.postRolePermissionsReplace;
type GetPermissionsOperation = paths[PermissionsPath]['get'];
type GetRolesOperation = paths[RolesPath]['get'];
type GetRolePermissionsOperation = paths[RolePermissionsPath]['get'];
type PermissionSavedViewsOperation = paths[PermissionSavedViewsPath];
type RoleSavedViewsOperation = paths[RoleSavedViewsPath];
type PostRolesOperation = paths[RolesPath]['post'];
type PostRoleCloneOperation = paths[typeof OPENAPI_RUNTIME_PATH.postRoleClone]['post'];
type PostRoleUpdateOperation = paths[typeof OPENAPI_RUNTIME_PATH.postRoleUpdate]['post'];
type PostRolePermissionsReplaceOperation = paths[RolePermissionsReplacePath]['post'];
type GetPermissionsEnvelope = GetPermissionsOperation['responses'][200]['content']['application/json'];
type GetRolesEnvelope = GetRolesOperation['responses'][200]['content']['application/json'];
type GetRolePermissionsEnvelope = GetRolePermissionsOperation['responses'][200]['content']['application/json'];
type GetPermissionsData = NonNullable<GetPermissionsEnvelope['data']>;
type GetRolesData = NonNullable<GetRolesEnvelope['data']>;
type GetRolePermissionsData = NonNullable<GetRolePermissionsEnvelope['data']>;
type PostRolesRequest = PostRolesOperation['requestBody']['content']['application/json'];
type PostRoleCloneRequest = PostRoleCloneOperation['requestBody']['content']['application/json'];
type PostRoleUpdateRequest = PostRoleUpdateOperation['requestBody']['content']['application/json'];
type PostRolePermissionsReplaceRequest =
  PostRolePermissionsReplaceOperation['requestBody']['content']['application/json'];
type DestructiveBatchResult = components['schemas']['DestructiveBatchResult'];
export type PermissionSavedViewPayload = NonNullable<
  PermissionSavedViewsOperation['post']['requestBody']
>['content']['application/json'];
export type RoleSavedViewPayload = NonNullable<
  RoleSavedViewsOperation['post']['requestBody']
>['content']['application/json'];
export type PermissionSavedViewRecord = NonNullable<
  PermissionSavedViewsOperation['get']['responses'][200]['content']['application/json']['data']
>['items'][number];
export type RoleSavedViewRecord = NonNullable<
  RoleSavedViewsOperation['get']['responses'][200]['content']['application/json']['data']
>['items'][number];
type PermissionSavedViewList = NonNullable<
  PermissionSavedViewsOperation['get']['responses'][200]['content']['application/json']['data']
>;
type RoleSavedViewList = NonNullable<
  RoleSavedViewsOperation['get']['responses'][200]['content']['application/json']['data']
>;

export function getRoles(filters?: { keyword?: string; builtin?: boolean; limit?: number; offset?: number }) {
  return request.get<GetRolesData>({
    url: OPENAPI_RUNTIME_PATH.getRoles,
    params: filters,
  }) as Promise<RoleListResponse>;
}

export const getRoleSavedViews = () =>
  request.get<RoleSavedViewList>({ url: OPENAPI_RUNTIME_PATH.getRoleSavedViews }).then((response) => response.items);
export const postRoleSavedView = (data: RoleSavedViewPayload) =>
  request.post<RoleSavedViewRecord>({ url: OPENAPI_RUNTIME_PATH.postRoleSavedView, data });
export const putRoleSavedView = (id: number, data: RoleSavedViewPayload) =>
  request.put<RoleSavedViewRecord>({ url: buildOpenApiRuntimePath('putRoleSavedView', { viewId: id }), data });
export const deleteRoleSavedView = (id: number) =>
  request.delete({ url: buildOpenApiRuntimePath('deleteRoleSavedView', { viewId: id }) }).then(() => undefined);

export function getRoleDetail(roleId: number) {
  return request.get<RoleDetailResponse>({
    url: buildOpenApiRuntimePath('getRole', { id: roleId }),
  });
}

export function getPermissions(filters?: PermissionFilters) {
  return request.get<GetPermissionsData>({
    url: OPENAPI_RUNTIME_PATH.getPermissions,
    params: filters,
  }) as Promise<PermissionListResponse>;
}

export const getPermissionSavedViews = () =>
  request
    .get<PermissionSavedViewList>({ url: OPENAPI_RUNTIME_PATH.getPermissionSavedViews })
    .then((response) => response.items);
export const postPermissionSavedView = (data: PermissionSavedViewPayload) =>
  request.post<PermissionSavedViewRecord>({ url: OPENAPI_RUNTIME_PATH.postPermissionSavedView, data });
export const putPermissionSavedView = (id: number, data: PermissionSavedViewPayload) =>
  request.put<PermissionSavedViewRecord>({
    url: buildOpenApiRuntimePath('putPermissionSavedView', { viewId: id }),
    data,
  });
export const deletePermissionSavedView = (id: number) =>
  request.delete({ url: buildOpenApiRuntimePath('deletePermissionSavedView', { viewId: id }) }).then(() => undefined);

export function getPermissionDetail(permissionId: number) {
  return request.get<PermissionDetailResponse>({
    url: buildOpenApiRuntimePath('getPermission', { id: permissionId }),
  });
}

export function getRolePermissionBindings(roleId: number) {
  return request.get<GetRolePermissionsData>({
    url: buildOpenApiRuntimePath('getRolePermissions', { id: roleId }),
  }) as Promise<RolePermissionBindingResponse>;
}

export function createRole(payload: PostRolesRequest & CreateRolePayload) {
  return request.post<RoleListItem>({
    url: OPENAPI_RUNTIME_PATH.postRoles,
    data: payload,
  });
}

export function cloneRole(roleId: number, payload: PostRoleCloneRequest & CloneRolePayload) {
  return request.post<RoleListItem>({
    url: buildOpenApiRuntimePath('postRoleClone', { id: roleId }),
    data: payload,
  });
}

export function updateRole(roleId: number, payload: PostRoleUpdateRequest & UpdateRolePayload) {
  return request.post<RoleListItem>({
    url: buildOpenApiRuntimePath('postRoleUpdate', { id: roleId }),
    data: payload,
  });
}

export function updateRoleStatus(roleId: number, payload: UpdateRoleStatusPayload) {
  return request.post<RoleDetailResponse>({
    url: buildOpenApiRuntimePath('postRoleStatus', { id: roleId }),
    data: payload,
  });
}

export function deleteRole(roleId: number) {
  return request.post<null>({
    url: buildOpenApiRuntimePath('postRoleDelete', { id: roleId }),
  });
}

export function replaceRolePermissions(
  roleId: number,
  payload: PostRolePermissionsReplaceRequest & ReplaceRolePermissionsPayload,
) {
  return request.post<null>({
    url: buildOpenApiRuntimePath('postRolePermissionsReplace', { id: roleId }),
    data: payload,
  });
}

export function addRolePermissions(roleId: number, payload: RolePermissionMutationPayload) {
  return request.post<null>({
    url: buildOpenApiRuntimePath('postRolePermissionsAdd', { id: roleId }),
    data: payload,
  });
}

export function removeRolePermissions(roleId: number, payload: RolePermissionMutationPayload) {
  return request.delete<DestructiveBatchResult>({
    url: buildOpenApiRuntimePath('deleteRolePermissions', { id: roleId }),
    data: payload,
  });
}
