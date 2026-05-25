import type { paths } from '@/contracts/openapi/generated/schema';
import { AUTH_API_PATH } from '@/modules/auth/contract/paths';
import type { CompleteRequiredPasswordChangePayload, LoginPayload } from '@/modules/auth/contract/types';
import { request } from '@/utils/request';

type LoginPath = (typeof AUTH_API_PATH)['LOGIN'];
type BootstrapPath = (typeof AUTH_API_PATH)['BOOTSTRAP'];
type PostAuthLoginOperation = paths[LoginPath]['post'];
type GetAuthBootstrapOperation = paths[BootstrapPath]['get'];
type PostAuthLoginRequest = PostAuthLoginOperation['requestBody']['content']['application/json'];
type PostAuthLoginResponse = PostAuthLoginOperation['responses']['200']['content']['application/json'];
type GetAuthBootstrapResponse = GetAuthBootstrapOperation['responses']['200']['content']['application/json'];

export function login(payload: LoginPayload & PostAuthLoginRequest) {
  return request.post<PostAuthLoginResponse['data']>({
    url: AUTH_API_PATH.LOGIN,
    data: payload,
  });
}

export function refresh() {
  return request.post<LoginResponse>({
    url: AUTH_API_PATH.REFRESH,
  });
}

export function logout() {
  return request.post<void>({
    url: AUTH_API_PATH.LOGOUT,
  });
}

export function completeRequiredPasswordChange(payload: CompleteRequiredPasswordChangePayload) {
  return request.post<void>({
    url: AUTH_API_PATH.COMPLETE_REQUIRED_PASSWORD_CHANGE,
    data: payload,
  });
}

export function getBootstrap() {
  return request.get<GetAuthBootstrapResponse['data']>({
    url: AUTH_API_PATH.BOOTSTRAP,
  });
}
