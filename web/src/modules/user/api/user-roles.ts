import type { paths } from '@/contracts/openapi/generated/schema';
import { RBAC_API_PATH } from '@/modules/rbac/contract/paths';
import type { RoleListResponse, UserRoleBindingResponse } from '@/modules/rbac/contract/role';
import { request } from '@/utils/request';

type RolesPath = (typeof RBAC_API_PATH)['ROLES'];
type UserRolesPath = '/api/users/{id}/roles';
type UserRolesAssignPath = '/api/users/{id}/roles/assign';
type GetRolesOperation = paths[RolesPath]['get'];
type GetUserRolesOperation = paths[UserRolesPath]['get'];
type PostUserRolesAssignOperation = paths[UserRolesAssignPath]['post'];
type GetRolesEnvelope = GetRolesOperation['responses'][200]['content']['application/json'];
type GetUserRolesEnvelope = GetUserRolesOperation['responses'][200]['content']['application/json'];
type GetRolesData = NonNullable<GetRolesEnvelope['data']>;
type GetUserRolesData = NonNullable<GetUserRolesEnvelope['data']>;

export type ReplaceUserRolesPayload = PostUserRolesAssignOperation['requestBody']['content']['application/json'];

export function getRoles() {
  return request.get<GetRolesData>({
    url: RBAC_API_PATH.ROLES,
  }) as Promise<RoleListResponse>;
}

export function getUserRoleBindings(userId: number) {
  return request.get<GetUserRolesData>({
    url: RBAC_API_PATH.USER_ROLES(userId),
  }) as Promise<UserRoleBindingResponse>;
}

export function assignUserRoles(userId: number, payload: ReplaceUserRolesPayload) {
  return request.post<null>({
    url: RBAC_API_PATH.USER_ROLE_ASSIGN(userId),
    data: payload,
  });
}
