import type { LocalizedTitle } from '@/contracts/i18n/locales';

export interface ApiSuccessEnvelope<T> {
  success: true;
  code: string;
  message: string;
  traceId: string;
  data: T;
  messageKey?: string;
  locale?: string;
}

export interface ApiErrorEnvelope {
  success: false;
  code: string;
  message: string;
  traceId: string;
  data?: Record<string, unknown> | null;
  messageKey?: string;
  locale?: string;
}

export type ApiEnvelope<T> = ApiSuccessEnvelope<T> | ApiErrorEnvelope;

export interface AppBootstrapRouteMeta {
  title: LocalizedTitle;
  titleKey?: string;
  semanticTitle?: LocalizedTitle;
  breadcrumbTitle?: LocalizedTitle;
  tabTitle?: LocalizedTitle;
  domain?: string;
  tabGroup?: string;
  dashboard?: boolean;
  pageKind?: string;
  icon?: string;
  permission?: string;
}
