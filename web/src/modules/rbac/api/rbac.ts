// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import { RBAC_API_PATH } from '../contract/paths';
import type { RoleListItem, RoleListResponse } from '../contract/role';
import type { PermissionDetailResponse, PermissionFilters, PermissionListResponse } from '../types/permission';
import type {
  CreateRolePayload,
  ReplaceRolePermissionsPayload,
  RoleDetailResponse,
  RolePermissionBindingResponse,
  RolePermissionMutationPayload,
  UpdateRolePayload,
  UpdateRoleStatusPayload,
} from '../types/rbac';

type PermissionsPath = (typeof RBAC_API_PATH)['PERMISSIONS'];
type RolesPath = (typeof RBAC_API_PATH)['ROLES'];
type RolePermissionsPath = (typeof RBAC_API_PATH)['ROLE_PERMISSIONS_TEMPLATE'];
type RolePermissionsReplacePath = (typeof RBAC_API_PATH)['ROLE_PERMISSIONS_REPLACE_TEMPLATE'];
type GetPermissionsOperation = paths[PermissionsPath]['get'];
type GetRolesOperation = paths[RolesPath]['get'];
type GetRolePermissionsOperation = paths[RolePermissionsPath]['get'];
type PostRolesOperation = paths[RolesPath]['post'];
type PostRoleUpdateOperation = paths[(typeof RBAC_API_PATH)['ROLE_UPDATE_TEMPLATE']]['post'];
type PostRolePermissionsReplaceOperation = paths[RolePermissionsReplacePath]['post'];
type GetPermissionsEnvelope = GetPermissionsOperation['responses'][200]['content']['application/json'];
type GetRolesEnvelope = GetRolesOperation['responses'][200]['content']['application/json'];
type GetRolePermissionsEnvelope = GetRolePermissionsOperation['responses'][200]['content']['application/json'];
type GetPermissionsData = NonNullable<GetPermissionsEnvelope['data']>;
type GetRolesData = NonNullable<GetRolesEnvelope['data']>;
type GetRolePermissionsData = NonNullable<GetRolePermissionsEnvelope['data']>;
type PostRolesRequest = PostRolesOperation['requestBody']['content']['application/json'];
type PostRoleUpdateRequest = PostRoleUpdateOperation['requestBody']['content']['application/json'];
type PostRolePermissionsReplaceRequest =
  PostRolePermissionsReplaceOperation['requestBody']['content']['application/json'];

export function getRoles() {
  return request.get<GetRolesData>({
    url: RBAC_API_PATH.ROLES,
  }) as Promise<RoleListResponse>;
}

export function getRoleDetail(roleId: number) {
  return request.get<RoleDetailResponse>({
    url: RBAC_API_PATH.ROLE_DETAIL(roleId),
  });
}

export function getPermissions(filters?: PermissionFilters) {
  return request.get<GetPermissionsData>({
    url: RBAC_API_PATH.PERMISSIONS,
    params: filters,
  }) as Promise<PermissionListResponse>;
}

export function getPermissionDetail(permissionId: number) {
  return request.get<PermissionDetailResponse>({
    url: RBAC_API_PATH.PERMISSION_DETAIL(permissionId),
  });
}

export function getRolePermissionBindings(roleId: number) {
  return request.get<GetRolePermissionsData>({
    url: RBAC_API_PATH.ROLE_PERMISSIONS(roleId),
  }) as Promise<RolePermissionBindingResponse>;
}

export function createRole(payload: PostRolesRequest & CreateRolePayload) {
  return request.post<RoleListItem>({
    url: RBAC_API_PATH.ROLES,
    data: payload,
  });
}

export function updateRole(roleId: number, payload: PostRoleUpdateRequest & UpdateRolePayload) {
  return request.post<RoleListItem>({
    url: RBAC_API_PATH.ROLE_UPDATE(roleId),
    data: payload,
  });
}

export function updateRoleStatus(roleId: number, payload: UpdateRoleStatusPayload) {
  return request.post<RoleDetailResponse>({
    url: RBAC_API_PATH.ROLE_STATUS(roleId),
    data: payload,
  });
}

export function deleteRole(roleId: number) {
  return request.post<null>({
    url: RBAC_API_PATH.ROLE_DELETE(roleId),
  });
}

export function replaceRolePermissions(
  roleId: number,
  payload: PostRolePermissionsReplaceRequest & ReplaceRolePermissionsPayload,
) {
  return request.post<null>({
    url: RBAC_API_PATH.ROLE_PERMISSIONS_REPLACE(roleId),
    data: payload,
  });
}

export function addRolePermissions(roleId: number, payload: RolePermissionMutationPayload) {
  return request.post<null>({
    url: RBAC_API_PATH.ROLE_PERMISSIONS_ADD(roleId),
    data: payload,
  });
}

export function removeRolePermissions(roleId: number, payload: RolePermissionMutationPayload) {
  return request.post<null>({
    url: RBAC_API_PATH.ROLE_PERMISSIONS_REMOVE(roleId),
    data: payload,
  });
}
