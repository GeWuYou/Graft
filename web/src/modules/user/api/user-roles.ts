import type { components } from '@/contracts/openapi/generated/schema';
import { RBAC_API_PATH } from '@/modules/rbac/contract/paths';
import type { RoleListResponse, UserRoleBindingResponse } from '@/modules/rbac/contract/role';
import { request } from '@/utils/request';

export type ReplaceUserRolesPayload = components['schemas']['ReplaceUserRolesRequest'];

export function getRoles() {
  return request.get<RoleListResponse>({
    url: RBAC_API_PATH.ROLES,
  });
}

export function getUserRoleBindings(userId: number) {
  return request.get<UserRoleBindingResponse>({
    url: RBAC_API_PATH.USER_ROLES(userId),
  });
}

export function assignUserRoles(userId: number, payload: ReplaceUserRolesPayload) {
  return request.post<null>({
    url: RBAC_API_PATH.USER_ROLE_ASSIGN(userId),
    data: payload,
  });
}
