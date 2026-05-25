import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import { RBAC_API_PATH } from '../contract/paths';
import type { RoleListItem, RoleListResponse } from '../contract/role';
import type { PermissionListResponse } from '../types/permission';
import type {
  CreateRolePayload,
  ReplaceRolePermissionsPayload,
  RolePermissionBindingResponse,
  UpdateRolePayload,
} from '../types/rbac';

type PermissionsPath = (typeof RBAC_API_PATH)['PERMISSIONS'];
type RolesPath = (typeof RBAC_API_PATH)['ROLES'];
type RolePermissionsPath = '/api/roles/{id}/permissions';
type RolePermissionAssignPath = '/api/roles/{id}/permissions/assign';
type GetPermissionsOperation = paths[PermissionsPath]['get'];
type GetRolesOperation = paths[RolesPath]['get'];
type GetRolePermissionsOperation = paths[RolePermissionsPath]['get'];
type PostRolesOperation = paths[RolesPath]['post'];
type PostRoleUpdateOperation = paths['/api/roles/{id}/update']['post'];
type PostRolePermissionAssignOperation = paths[RolePermissionAssignPath]['post'];
type GetPermissionsEnvelope = GetPermissionsOperation['responses'][200]['content']['application/json'];
type GetRolesEnvelope = GetRolesOperation['responses'][200]['content']['application/json'];
type GetRolePermissionsEnvelope = GetRolePermissionsOperation['responses'][200]['content']['application/json'];
type GetPermissionsData = NonNullable<GetPermissionsEnvelope['data']>;
type GetRolesData = NonNullable<GetRolesEnvelope['data']>;
type GetRolePermissionsData = NonNullable<GetRolePermissionsEnvelope['data']>;
type PostRolesRequest = PostRolesOperation['requestBody']['content']['application/json'];
type PostRoleUpdateRequest = PostRoleUpdateOperation['requestBody']['content']['application/json'];
type PostRolePermissionAssignRequest = PostRolePermissionAssignOperation['requestBody']['content']['application/json'];

export function getRoles() {
  return request.get<GetRolesData>({
    url: RBAC_API_PATH.ROLES,
  }) as Promise<RoleListResponse>;
}

export function getPermissions() {
  return request.get<GetPermissionsData>({
    url: RBAC_API_PATH.PERMISSIONS,
  }) as Promise<PermissionListResponse>;
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

export function assignRolePermissions(
  roleId: number,
  payload: PostRolePermissionAssignRequest & ReplaceRolePermissionsPayload,
) {
  return request.post<null>({
    url: RBAC_API_PATH.ROLE_PERMISSION_ASSIGN(roleId),
    data: payload,
  });
}
