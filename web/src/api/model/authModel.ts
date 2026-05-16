import type { LocalizedTitle } from '@/locales';

export const API_CODE = {
  OK: 'OK',
  AUTH_INVALID_CREDENTIALS: 'AUTH_INVALID_CREDENTIALS',
  AUTH_TOKEN_MISSING: 'AUTH_TOKEN_MISSING',
  AUTH_TOKEN_EXPIRED: 'AUTH_TOKEN_EXPIRED',
  AUTH_TOKEN_INVALID: 'AUTH_TOKEN_INVALID',
  AUTH_FORBIDDEN: 'AUTH_FORBIDDEN',
  AUTH_PASSWORD_CHANGE_REQUIRED: 'AUTH_PASSWORD_CHANGE_REQUIRED',
  AUTH_PASSWORD_POLICY_VIOLATION: 'AUTH_PASSWORD_POLICY_VIOLATION',
  AUTH_PASSWORD_REUSE_FORBIDDEN: 'AUTH_PASSWORD_REUSE_FORBIDDEN',
  AUTH_CURRENT_PASSWORD_INVALID: 'AUTH_CURRENT_PASSWORD_INVALID',
  COMMON_INVALID_ARGUMENT: 'COMMON_INVALID_ARGUMENT',
  COMMON_INTERNAL_ERROR: 'COMMON_INTERNAL_ERROR',
} as const;

export type ApiCode = (typeof API_CODE)[keyof typeof API_CODE] | (string & {});

export interface ApiSuccessEnvelope<T> {
  success: true;
  code: ApiCode;
  message: string;
  traceId: string;
  data: T;
  messageKey?: string;
  locale?: string;
}

export interface ApiErrorEnvelope {
  success: false;
  code: ApiCode;
  message: string;
  traceId: string;
  data?: null;
  messageKey?: string;
  locale?: string;
}

export type ApiEnvelope<T> = ApiSuccessEnvelope<T> | ApiErrorEnvelope;

export interface LoginUser {
  id: number;
  username: string;
  display_name: string;
}

export interface LoginResponse {
  access_token: string;
  expires_at: string;
  must_change_password: boolean;
  user: LoginUser;
}

export interface BootstrapMenu {
  code: string;
  title: string;
  path: string;
  icon: string;
  permission: string;
}

export interface BootstrapLocale {
  current_locale: string;
  default_locale: string;
  fallback_locale: string;
  supported_locales: string[];
}

export interface BootstrapResponse {
  user: LoginUser;
  must_change_password: boolean;
  permissions: string[];
  menus: BootstrapMenu[];
  locale: BootstrapLocale;
}

export interface LoginPayload {
  username: string;
  password: string;
}

export interface ChangePasswordPayload {
  current_password: string;
  new_password: string;
}

export interface AppBootstrapRouteMeta {
  title: LocalizedTitle;
  icon?: string;
  permission?: string;
}
