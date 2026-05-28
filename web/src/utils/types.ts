import type { Component, FunctionalComponent, VNodeChild } from 'vue';
import type { LocationQueryRaw, RouteRecordName, RouteRecordRaw } from 'vue-router';

import type { LocalizedTitle } from '@/contracts/i18n/locales';

export type ModeType = 'light' | 'dark';
export type PageFooterContent = string | LocalizedTitle;
export type GovernanceDomain = 'rbac' | 'audit' | 'monitor';
export type AppRoutePageKind = 'overview' | 'list' | 'detail' | 'runtime' | 'investigation';

export interface PageFooterMeta {
  visible?: boolean;
  content?: PageFooterContent;
}

/**
 * AppRouteMeta describes the stable route metadata consumed by the `web` shell.
 *
 * Use `title` as the rendered localized title payload. `titleKey` is the
 * backend/bootstrap contract key that can be preserved alongside `title` for
 * diagnostics or future re-localization, but current runtime renderers still
 * read `title`.
 *
 * New static routes should keep defining `title` directly. Backend-driven menu
 * routes should prefer flowing `titleKey` through the bootstrap transformer,
 * which resolves `title` from locale catalogs first and falls back to the
 * bootstrap label when no translation exists.
 */
export interface AppRouteMeta {
  title?: LocalizedTitle;
  titleKey?: string;
  domain?: GovernanceDomain;
  domainTitle?: LocalizedTitle;
  semanticTitle?: LocalizedTitle;
  breadcrumbTitle?: LocalizedTitle;
  tabTitle?: LocalizedTitle;
  tabGroup?: string;
  dashboard?: boolean;
  pageKind?: AppRoutePageKind;
  investigationSurface?: boolean;
  icon?: string | Component | FunctionalComponent | (() => VNodeChild);
  orderNo?: number;
  hidden?: boolean;
  hiddenBreadcrumb?: boolean;
  single?: boolean;
  expanded?: boolean;
  frameSrc?: string;
  frameBlank?: boolean;
  keepAlive?: boolean;
  footer?: false | PageFooterMeta;
}

export interface MenuRoute extends Omit<RouteRecordRaw, 'children' | 'meta'> {
  children?: MenuRoute[];
  meta?: AppRouteMeta;
  title?: LocalizedTitle;
  icon?: AppRouteMeta['icon'];
}

export interface TRouterInfo {
  path: string;
  routeIdx?: number;
  title?: LocalizedTitle;
  name?: RouteRecordName | null;
  isHome?: boolean;
  isAlive?: boolean;
  query?: LocationQueryRaw;
  meta?: AppRouteMeta;
}

export interface TTabRouterType {
  tabRouterList: TRouterInfo[];
  isRefreshing: boolean;
}

export interface TTabRemoveOptions {
  value: string | number;
  index: number;
}

export interface NotificationItem {
  id: string;
  content: string;
  type: string;
  status: boolean;
  collected: boolean;
  date: string;
  quality: string;
}

export interface UserInfo {
  name: string;
  username: string;
  roles: string[];
  permissions: string[];
}
