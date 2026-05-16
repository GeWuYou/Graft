import type { BootstrapResponse, ChangePasswordPayload, LoginPayload, LoginResponse } from '@/api/model/authModel';
import { request } from '@/utils/request';

const Api = {
  Bootstrap: '/api/auth/bootstrap',
  ChangePassword: '/api/auth/change-password',
  Login: '/api/auth/login',
  Logout: '/api/auth/logout',
  Refresh: '/api/auth/refresh',
} as const;

export function login(payload: LoginPayload) {
  return request.post<LoginResponse>({
    url: Api.Login,
    data: payload,
  });
}

export function refresh() {
  return request.post<LoginResponse>({
    url: Api.Refresh,
  });
}

export function logout() {
  return request.post<void>({
    url: Api.Logout,
  });
}

export function changePassword(payload: ChangePasswordPayload) {
  return request.post<void>({
    url: Api.ChangePassword,
    data: payload,
  });
}

export function getBootstrap() {
  return request.get<BootstrapResponse>({
    url: Api.Bootstrap,
  });
}
