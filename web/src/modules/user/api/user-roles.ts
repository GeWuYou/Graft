import { buildOpenApiRuntimePath, OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import type { components, paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import type {
  BatchUserRoleMutationPayload,
  ReplaceUserRolesPayload,
  RoleListResponse,
  UserRoleBindingResponse,
  UserRoleMutation,
} from '../types/role';

type RolesPath = typeof OPENAPI_RUNTIME_PATH.getRoles;
type UserRolesPath = typeof OPENAPI_RUNTIME_PATH.getUserRoles;
type GetRolesOperation = paths[RolesPath]['get'];
type GetUserRolesOperation = paths[UserRolesPath]['get'];
type GetRolesEnvelope = GetRolesOperation['responses'][200]['content']['application/json'];
type GetUserRolesEnvelope = GetUserRolesOperation['responses'][200]['content']['application/json'];
type GetRolesData = NonNullable<GetRolesEnvelope['data']>;
type GetUserRolesData = NonNullable<GetUserRolesEnvelope['data']>;

const singleUserRoleMutationOperationMap: Record<
  Exclude<UserRoleMutation, 'remove'>,
  'postUserRolesReplace' | 'postUserRolesAdd'
> = {
  replace: 'postUserRolesReplace',
  add: 'postUserRolesAdd',
};

const batchUserRoleMutationPathMap: Record<Exclude<UserRoleMutation, 'remove'>, string> = {
  replace: OPENAPI_RUNTIME_PATH.postUsersRolesReplace,
  add: OPENAPI_RUNTIME_PATH.postUsersRolesAdd,
};

type DestructiveBatchResult = components['schemas']['DestructiveBatchResult'];

export function getRoles() {
  return request.get<GetRolesData>({
    url: OPENAPI_RUNTIME_PATH.getRoles,
  }) as Promise<RoleListResponse>;
}

export function getUserRoleBindings(userId: number) {
  return request.get<GetUserRolesData>({
    url: buildOpenApiRuntimePath('getUserRoles', { id: userId }),
  }) as Promise<UserRoleBindingResponse>;
}

export function mutateUserRoles(userId: number, operation: UserRoleMutation, payload: ReplaceUserRolesPayload) {
  if (operation === 'remove') {
    return request.delete<DestructiveBatchResult>({
      url: buildOpenApiRuntimePath('deleteUserRoles', { id: userId }),
      data: payload,
    });
  }
  return request.post<null>({
    url: buildOpenApiRuntimePath(singleUserRoleMutationOperationMap[operation], { id: userId }),
    data: payload,
  });
}

export function mutateBatchUserRoles(operation: UserRoleMutation, payload: BatchUserRoleMutationPayload) {
  if (operation === 'remove') {
    return request.delete<DestructiveBatchResult>({
      url: OPENAPI_RUNTIME_PATH.deleteUsersRoles,
      data: payload,
    });
  }
  return request.post<DestructiveBatchResult>({
    url: batchUserRoleMutationPathMap[operation],
    data: payload,
  });
}
