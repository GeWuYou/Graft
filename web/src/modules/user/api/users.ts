// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import { USER_API_PATH } from '../contract/paths';
import { USER_STATUS } from '../contract/status';
import type {
  RawUserListItem,
  ResetUserPasswordPayload,
  UpdateUserStatusPayload,
  UserListItem,
  UserListResponse,
} from '../types/user';

type UsersPath = (typeof USER_API_PATH)['USERS'];
type GetUserByIdPath = (typeof USER_API_PATH)['USER_BY_ID_TEMPLATE'];
type PostUserUpdatePath = (typeof USER_API_PATH)['USER_UPDATE_TEMPLATE'];
type PostUserStatusPath = (typeof USER_API_PATH)['USER_STATUS_TEMPLATE'];
type PostUserResetPasswordPath = (typeof USER_API_PATH)['USER_RESET_PASSWORD_TEMPLATE'];
type PostUserDeletePath = (typeof USER_API_PATH)['USER_DELETE_TEMPLATE'];
type GetUsersOperation = paths[UsersPath]['get'];
type GetUserByIdOperation = paths[GetUserByIdPath]['get'];
type PostUsersOperation = paths[UsersPath]['post'];
type PostUserUpdateOperation = paths[PostUserUpdatePath]['post'];
type PostUserStatusOperation = paths[PostUserStatusPath]['post'];
type PostUserResetPasswordOperation = paths[PostUserResetPasswordPath]['post'];
type PostUserDeleteOperation = paths[PostUserDeletePath]['post'];
type GetUsersResponse = GetUsersOperation['responses']['200']['content']['application/json'];
type GetUserByIdResponse = GetUserByIdOperation['responses']['200']['content']['application/json'];
type PostUsersRequest = PostUsersOperation['requestBody']['content']['application/json'];
type PostUserUpdateRequest = PostUserUpdateOperation['requestBody']['content']['application/json'];
type PostUserStatusRequest = PostUserStatusOperation['requestBody']['content']['application/json'];
type PostUserResetPasswordRequest = PostUserResetPasswordOperation['requestBody']['content']['application/json'];
type PostUserDeleteResponse = PostUserDeleteOperation['responses']['200']['content']['application/json'];
type GetUsersResponseData = NonNullable<GetUsersResponse['data']>;
type GetUserByIdResponseData = NonNullable<GetUserByIdResponse['data']>;
type PostUserDeleteResponseData = NonNullable<PostUserDeleteResponse['data']>;

function normalizeUserStatus(status?: string | null) {
  return status === USER_STATUS.DISABLED ? USER_STATUS.DISABLED : USER_STATUS.ENABLED;
}

function normalizeUserListItem(item: RawUserListItem): UserListItem {
  return {
    ...item,
    status: normalizeUserStatus(item.status),
    roles: item.roles ?? [],
  };
}

/**
 * getUsers 获取用户管理列表数据。
 *
 * 该请求调用 `USER_API_PATH.USERS`，用于读取当前用户管理页所需的用户集合快照。
 *
 * @returns 返回 `UserListResponse` 约定的用户列表结果。
 */
export function getUsers() {
  return request
    .get<GetUsersResponseData>({
      url: USER_API_PATH.USERS,
    })
    .then(
      (response): UserListResponse => ({
        ...response,
        items: response.items.map(normalizeUserListItem),
      }),
    );
}

/**
 * getUserById 读取指定用户的详情快照。
 *
 * 该请求调用 `USER_API_PATH.USER_BY_ID(userId)`，用于获取单个用户的现有管理视图。
 *
 * @param userId 需要读取的用户 ID。
 * @returns 返回规范化后的 `UserListItem`。
 */
export function getUserById(userId: number) {
  return request
    .get<GetUserByIdResponseData>({
      url: USER_API_PATH.USER_BY_ID(userId),
    })
    .then(normalizeUserListItem);
}

/**
 * createUser 创建新的后台用户。
 *
 * 该请求调用 `USER_API_PATH.USERS`，请求体需要满足 `CreateUserPayload` 约定。
 *
 * @param payload 创建用户所需的请求体。
 * @returns 返回新建后的 `UserListItem`。
 */
export function createUser(payload: PostUsersRequest) {
  return request
    .post<RawUserListItem>({
      url: USER_API_PATH.USERS,
      data: payload,
    })
    .then(normalizeUserListItem);
}

/**
 * updateUser 更新指定用户的基础资料。
 *
 * 该请求调用 `USER_API_PATH.USER_UPDATE(userId)`，请求体遵循 `UpdateUserPayload`。
 *
 * @param userId 需要更新的用户 ID。
 * @param payload 更新用户资料所需的请求体。
 * @returns 返回更新后的 `UserListItem`。
 */
export function updateUser(userId: number, payload: PostUserUpdateRequest) {
  return request
    .post<RawUserListItem>({
      url: USER_API_PATH.USER_UPDATE(userId),
      data: payload,
    })
    .then(normalizeUserListItem);
}

/**
 * updateUserStatus 更新指定用户的启停状态。
 *
 * 该请求调用 `USER_API_PATH.USER_STATUS(userId)`，请求体遵循 `UpdateUserStatusPayload`。
 *
 * @param userId 需要更新状态的用户 ID。
 * @param payload 用户状态更新请求体。
 * @returns 返回更新后的 `UserListItem`。
 */
export function updateUserStatus(userId: number, payload: UpdateUserStatusPayload) {
  return request
    .post<RawUserListItem>({
      url: USER_API_PATH.USER_STATUS(userId),
      data: payload satisfies PostUserStatusRequest,
    })
    .then(normalizeUserListItem);
}

/**
 * resetUserPassword 重置指定用户的密码。
 *
 * 该请求调用 `USER_API_PATH.USER_RESET_PASSWORD(userId)`，请求体遵循 `ResetUserPasswordPayload`。
 *
 * @param userId 需要重置密码的用户 ID。
 * @param payload 重置密码所需的新密码请求体。
 * @returns 成功时返回 `null`。
 */
export function resetUserPassword(userId: number, payload: ResetUserPasswordPayload) {
  return request.post<null>({
    url: USER_API_PATH.USER_RESET_PASSWORD(userId),
    data: payload satisfies PostUserResetPasswordRequest,
  });
}

/**
 * deleteUser 删除指定用户。
 *
 * 该请求调用 `USER_API_PATH.USER_DELETE(userId)`，用于执行用户删除动作。
 *
 * @param userId 需要删除的用户 ID。
 * @returns 成功时返回 `null`。
 */
export function deleteUser(userId: number) {
  return request.post<PostUserDeleteResponseData>({
    url: USER_API_PATH.USER_DELETE(userId),
  });
}
