import type {
  BootstrapResponse,
  CompleteRequiredPasswordChangePayload,
  LoginPayload,
  LoginResponse,
} from '@/api/model/authModel';
import { AUTH_API_PATH } from '@/contracts/auth/paths';
import { request } from '@/utils/request';

export function login(payload: LoginPayload) {
  return request.post<LoginResponse>({
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
  return request.get<BootstrapResponse>({
    url: AUTH_API_PATH.BOOTSTRAP,
  });
}
