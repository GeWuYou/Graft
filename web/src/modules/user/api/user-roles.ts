// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import { USER_API_PATH } from '../contract/paths';
import type {
  BatchUserRoleMutationPayload,
  ReplaceUserRolesPayload,
  RoleListResponse,
  UserRoleBindingResponse,
  UserRoleMutation,
} from '../types/role';

type RolesPath = (typeof USER_API_PATH)['ROLES'];
type UserRolesPath = (typeof USER_API_PATH)['USER_ROLES_TEMPLATE'];
type GetRolesOperation = paths[RolesPath]['get'];
type GetUserRolesOperation = paths[UserRolesPath]['get'];
type GetRolesEnvelope = GetRolesOperation['responses'][200]['content']['application/json'];
type GetUserRolesEnvelope = GetUserRolesOperation['responses'][200]['content']['application/json'];
type GetRolesData = NonNullable<GetRolesEnvelope['data']>;
type GetUserRolesData = NonNullable<GetUserRolesEnvelope['data']>;

const singleUserRoleMutationPathMap: Record<UserRoleMutation, (userId: number) => string> = {
  replace: USER_API_PATH.USER_ROLE_REPLACE,
  add: USER_API_PATH.USER_ROLE_ADD,
  remove: USER_API_PATH.USER_ROLE_REMOVE,
};

const batchUserRoleMutationPathMap: Record<UserRoleMutation, string> = {
  replace: USER_API_PATH.BATCH_USER_ROLE_REPLACE,
  add: USER_API_PATH.BATCH_USER_ROLE_ADD,
  remove: USER_API_PATH.BATCH_USER_ROLE_REMOVE,
};

export function getRoles() {
  return request.get<GetRolesData>({
    url: USER_API_PATH.ROLES,
  }) as Promise<RoleListResponse>;
}

export function getUserRoleBindings(userId: number) {
  return request.get<GetUserRolesData>({
    url: USER_API_PATH.USER_ROLES(userId),
  }) as Promise<UserRoleBindingResponse>;
}

export function mutateUserRoles(userId: number, operation: UserRoleMutation, payload: ReplaceUserRolesPayload) {
  return request.post<null>({
    url: singleUserRoleMutationPathMap[operation](userId),
    data: payload,
  });
}

export function mutateBatchUserRoles(operation: UserRoleMutation, payload: BatchUserRoleMutationPayload) {
  return request.post<null>({
    url: batchUserRoleMutationPathMap[operation],
    data: payload,
  });
}
