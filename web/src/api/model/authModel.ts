import type { LocalizedTitle } from '@/locales';

export interface LoginUser {
  id: number;
  username: string;
  display_name: string;
}

export interface LoginResponse {
  access_token: string;
  expires_at: string;
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
  permissions: string[];
  menus: BootstrapMenu[];
  locale: BootstrapLocale;
}

export interface LoginPayload {
  username: string;
  password: string;
}

export interface AppBootstrapRouteMeta {
  title: LocalizedTitle;
  icon?: string;
  permission?: string;
}
