import { API_CODE, type ApiCode, type ApiResponseCode } from '@/contracts/api/codes';
import type { LocalizedTitle } from '@/contracts/i18n/locales';
import type { components } from '@/contracts/openapi/generated/schema';

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
  data?: Record<string, unknown> | null;
  messageKey?: string;
  locale?: string;
}

export type ApiEnvelope<T> = ApiSuccessEnvelope<T> | ApiErrorEnvelope;

type AuthSchemas = components['schemas'];

export type LoginUser = AuthSchemas['LoginUser'];
export type LoginResponse = AuthSchemas['LoginResponse'];
export type BootstrapMenu = AuthSchemas['BootstrapMenu'];
export type BootstrapLocale = AuthSchemas['BootstrapLocale'];
export type BootstrapResponse = AuthSchemas['BootstrapResponse'];
export type LoginPayload = AuthSchemas['LoginRequest'];

export interface CompleteRequiredPasswordChangePayload {
  new_password: string;
}

export interface AppBootstrapRouteMeta {
  title: LocalizedTitle;
  titleKey?: string;
  icon?: string;
  permission?: string;
}
