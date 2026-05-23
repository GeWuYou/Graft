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

export function getRoles() {
  return request.get<RoleListResponse>({
    url: RBAC_API_PATH.ROLES,
  });
}

export function getPermissions() {
  return request.get<PermissionListResponse>({
    url: RBAC_API_PATH.PERMISSIONS,
  });
}

export function getRolePermissionBindings(roleId: number) {
  return request.get<RolePermissionBindingResponse>({
    url: RBAC_API_PATH.ROLE_PERMISSIONS(roleId),
  });
}

export function createRole(payload: CreateRolePayload) {
  return request.post<RoleListItem>({
    url: RBAC_API_PATH.ROLES,
    data: payload,
  });
}

export function updateRole(roleId: number, payload: UpdateRolePayload) {
  return request.post<RoleListItem>({
    url: RBAC_API_PATH.ROLE_UPDATE(roleId),
    data: payload,
  });
}

export function assignRolePermissions(roleId: number, payload: ReplaceRolePermissionsPayload) {
  return request.post<null>({
    url: RBAC_API_PATH.ROLE_PERMISSION_ASSIGN(roleId),
    data: payload,
  });
}
