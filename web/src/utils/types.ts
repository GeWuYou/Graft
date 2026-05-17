import type { Component, FunctionalComponent, VNodeChild } from 'vue';
import type { LocationQueryRaw, RouteRecordName, RouteRecordRaw } from 'vue-router';

import type { LocalizedTitle } from '@/contracts/i18n/locales';

export type ModeType = 'light' | 'dark';

export interface AppRouteMeta {
  title?: LocalizedTitle;
  titleKey?: string;
  icon?: string | Component | FunctionalComponent | (() => VNodeChild);
  orderNo?: number;
  hidden?: boolean;
  hiddenBreadcrumb?: boolean;
  single?: boolean;
  expanded?: boolean;
  frameSrc?: string;
  frameBlank?: boolean;
  keepAlive?: boolean;
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
