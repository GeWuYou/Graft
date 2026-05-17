import type {
  CreateRolePayload,
  PermissionListResponse,
  ReplaceRolePermissionsPayload,
  ReplaceUserRolesPayload,
  RoleListItem,
  RoleListResponse,
  RolePermissionBindingResponse,
  UpdateRolePayload,
} from '@/api/model/rbacModel';
import { RBAC_API_PATH } from '@/contracts/rbac/paths';
import { request } from '@/utils/request';

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

export function assignUserRoles(userId: number, payload: ReplaceUserRolesPayload) {
  return request.post<null>({
    url: RBAC_API_PATH.USER_ROLE_ASSIGN(userId),
    data: payload,
  });
}
