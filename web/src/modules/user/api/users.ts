import { buildOpenApiRuntimePath, OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import type { components, paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import { USER_STATUS } from '../contract/status';
import type {
  RawUserListItem,
  ResetUserPasswordPayload,
  UpdateUserStatusPayload,
  UserListItem,
  UserListResponse,
} from '../types/user';

type UsersPath = typeof OPENAPI_RUNTIME_PATH.getUsers;
type GetUserByIdPath = typeof OPENAPI_RUNTIME_PATH.getUserById;
type PostUserUpdatePath = typeof OPENAPI_RUNTIME_PATH.postUserUpdate;
type PostUserStatusPath = typeof OPENAPI_RUNTIME_PATH.postUserStatus;
type PostUserResetPasswordPath = typeof OPENAPI_RUNTIME_PATH.postUserResetPassword;
type PostUserDeletePath = typeof OPENAPI_RUNTIME_PATH.postUserDelete;
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
type UserSavedViewsPath = typeof OPENAPI_RUNTIME_PATH.getUserSavedViews;
type GetUserSavedViewsOperation = paths[UserSavedViewsPath]['get'];
type GetUserSavedViewsResponse = GetUserSavedViewsOperation['responses'][200]['content']['application/json'];
type GetUserSavedViewsResponseData = NonNullable<GetUserSavedViewsResponse['data']>;
type PostUserSavedViewOperation = paths[UserSavedViewsPath]['post'];
type PostUserSavedViewResponse = PostUserSavedViewOperation['responses'][201]['content']['application/json'];
type PostUserSavedViewResponseData = NonNullable<PostUserSavedViewResponse['data']>;
type UserSavedViewPath = typeof OPENAPI_RUNTIME_PATH.putUserSavedView;
type PutUserSavedViewOperation = paths[UserSavedViewPath]['put'];
type PutUserSavedViewResponse = PutUserSavedViewOperation['responses'][200]['content']['application/json'];
type PutUserSavedViewResponseData = NonNullable<PutUserSavedViewResponse['data']>;

export type UserSavedView = components['schemas']['saved-view'];
type UserSavedViewRequest = components['schemas']['saved-view-request'];

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
      url: OPENAPI_RUNTIME_PATH.getUsers,
    })
    .then((response): UserListResponse => ({
      ...response,
      items: response.items.map(normalizeUserListItem),
    }));
}

export function getUserById(userId: number) {
  return request
    .get<GetUserByIdResponseData>({
      url: buildOpenApiRuntimePath('getUserById', { id: userId }),
    })
    .then(normalizeUserListItem);
}

export function createUser(payload: PostUsersRequest) {
  return request
    .post<RawUserListItem>({
      url: OPENAPI_RUNTIME_PATH.postUsers,
      data: payload,
    })
    .then(normalizeUserListItem);
}

export function updateUser(userId: number, payload: PostUserUpdateRequest) {
  return request
    .post<RawUserListItem>({
      url: buildOpenApiRuntimePath('postUserUpdate', { id: userId }),
      data: payload,
    })
    .then(normalizeUserListItem);
}

export function updateUserStatus(userId: number, payload: UpdateUserStatusPayload) {
  return request
    .post<RawUserListItem>({
      url: buildOpenApiRuntimePath('postUserStatus', { id: userId }),
      data: payload satisfies PostUserStatusRequest,
    })
    .then(normalizeUserListItem);
}

export function resetUserPassword(userId: number, payload: ResetUserPasswordPayload) {
  return request.post<null>({
    url: buildOpenApiRuntimePath('postUserResetPassword', { id: userId }),
    data: payload satisfies PostUserResetPasswordRequest,
  });
}

export function deleteUser(userId: number) {
  return request.post<PostUserDeleteResponseData>({
    url: buildOpenApiRuntimePath('postUserDelete', { id: userId }),
  });
}

export async function getUserSavedViews(): Promise<UserSavedView[]> {
  const data = await request.get<GetUserSavedViewsResponseData>({ url: OPENAPI_RUNTIME_PATH.getUserSavedViews });
  return data.items;
}

export function postUserSavedView(payload: UserSavedViewRequest): Promise<UserSavedView> {
  return request.post<PostUserSavedViewResponseData>({
    url: OPENAPI_RUNTIME_PATH.postUserSavedView,
    data: payload,
  }) as Promise<UserSavedView>;
}

export function putUserSavedView(viewId: number, payload: UserSavedViewRequest): Promise<UserSavedView> {
  return request.put<PutUserSavedViewResponseData>({
    url: buildOpenApiRuntimePath('putUserSavedView', { viewId }),
    data: payload,
  }) as Promise<UserSavedView>;
}

export async function deleteUserSavedView(viewId: number): Promise<void> {
  await request.delete({ url: buildOpenApiRuntimePath('deleteUserSavedView', { viewId }) });
}
