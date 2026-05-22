import { request } from '@/utils/request';

import { USER_API_PATH } from '../contract/paths';
import type {
  CreateUserPayload,
  ResetUserPasswordPayload,
  UpdateUserPayload,
  UpdateUserStatusPayload,
  UserListItem,
  UserListResponse,
} from '../types/user';

export function getUsers() {
  return request.get<UserListResponse>({
    url: USER_API_PATH.USERS,
  });
}

export function createUser(payload: CreateUserPayload) {
  return request.post<UserListItem>({
    url: USER_API_PATH.USERS,
    data: payload,
  });
}

export function updateUser(userId: number, payload: UpdateUserPayload) {
  return request.post<UserListItem>({
    url: USER_API_PATH.USER_UPDATE(userId),
    data: payload,
  });
}

export function updateUserStatus(userId: number, payload: UpdateUserStatusPayload) {
  return request.post<UserListItem>({
    url: USER_API_PATH.USER_STATUS(userId),
    data: payload,
  });
}

export function resetUserPassword(userId: number, payload: ResetUserPasswordPayload) {
  return request.post<null>({
    url: USER_API_PATH.USER_RESET_PASSWORD(userId),
    data: payload,
  });
}

export function deleteUser(userId: number) {
  return request.post<null>({
    url: USER_API_PATH.USER_DELETE(userId),
  });
}
