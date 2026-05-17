import { API_CODE, type ApiCode, type ApiResponseCode } from '@/contracts/api/codes';
import type { LocalizedTitle } from '@/contracts/i18n/locales';

export { API_CODE };
export type { ApiCode, ApiResponseCode };

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
  code: ApiResponseCode;
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
  title_key?: string;
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
  roles: string[];
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

export interface CompleteRequiredPasswordChangePayload {
  new_password: string;
}

export interface AppBootstrapRouteMeta {
  title: LocalizedTitle;
  titleKey?: string;
  icon?: string;
  permission?: string;
}
