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

export function getUsers() {
  return request
    .get<GetUsersResponseData>({
      url: USER_API_PATH.USERS,
    })
    .then((response): UserListResponse => ({
      ...response,
      items: response.items.map(normalizeUserListItem),
    }));
}

export function getUserById(userId: number) {
  return request
    .get<GetUserByIdResponseData>({
      url: USER_API_PATH.USER_BY_ID(userId),
    })
    .then(normalizeUserListItem);
}

export function createUser(payload: PostUsersRequest) {
  return request
    .post<RawUserListItem>({
      url: USER_API_PATH.USERS,
      data: payload,
    })
    .then(normalizeUserListItem);
}

export function updateUser(userId: number, payload: PostUserUpdateRequest) {
  return request
    .post<RawUserListItem>({
      url: USER_API_PATH.USER_UPDATE(userId),
      data: payload,
    })
    .then(normalizeUserListItem);
}

export function updateUserStatus(userId: number, payload: UpdateUserStatusPayload) {
  return request
    .post<RawUserListItem>({
      url: USER_API_PATH.USER_STATUS(userId),
      data: payload satisfies PostUserStatusRequest,
    })
    .then(normalizeUserListItem);
}

export function resetUserPassword(userId: number, payload: ResetUserPasswordPayload) {
  return request.post<null>({
    url: USER_API_PATH.USER_RESET_PASSWORD(userId),
    data: payload satisfies PostUserResetPasswordRequest,
  });
}

export function deleteUser(userId: number) {
  return request.post<PostUserDeleteResponseData>({
    url: USER_API_PATH.USER_DELETE(userId),
  });
}
