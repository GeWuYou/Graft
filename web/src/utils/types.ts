import type { Component, FunctionalComponent, VNodeChild } from 'vue';
import type { LocationQueryRaw, RouteParamsRaw, RouteRecordName, RouteRecordRaw } from 'vue-router';

import type { LocalizedTitle } from '@/contracts/i18n/locales';

export type ModeType = 'light' | 'dark';
export type PageFooterContent = string | LocalizedTitle;
export type GovernanceDomain = 'rbac' | 'audit' | 'monitor' | 'security';
export type AppRoutePageKind = 'overview' | 'list' | 'detail' | 'runtime' | 'investigation';
export type AppRoutePageSurface = 'shell' | 'overview-dashboard' | 'paged-table' | 'form-detail' | 'editor';
export type AppRouteSidebarMotion = 'default' | 'wide-table';

export interface PageFooterMeta {
  visible?: boolean;
  content?: PageFooterContent;
}

/**
 * AppRouteMeta 描述 `web` 壳层消费的稳定路由元数据。
 *
 * `title` 是当前渲染使用的本地化标题；`titleKey` 是后端/bootstrap 契约键，
 * 可与 `title` 并存以支持诊断或后续重新本地化，但当前运行时渲染器仍读取 `title`。
 *
 * 新增静态路由应继续直接定义 `title`；后端驱动的菜单路由应优先让 `titleKey`
 * 经过 bootstrap 转换器，由 locale catalog 解析 `title`，无翻译时才回退到 bootstrap 标签。
 */
export interface AppRouteMeta {
  /** 显式声明 bootstrap 导航图元数据，不从 URL 层级关系推断。 */
  navigationCode?: string;
  navigationKind?: 'group' | 'entry';
  /** 仅用于侧栏视觉分组，不会创建导航节点。 */
  navigationSection?: NavigationSection;
  navigationTargetPath?: string;
  /** 显式声明供壳层面包屑和标签页使用的本地化导航祖先。 */
  navigationAncestors?: NavigationAncestor[];
  /** 标签页出现同名项时用于消歧的本地化导航路径，不作为持久化标签状态。 */
  navigationTitle?: LocalizedTitle;
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
  pageSurface?: AppRoutePageSurface;
  /** 为需要保留可用宽度的宽表非列表页显式声明壳层动效。 */
  sidebarMotion?: AppRouteSidebarMotion;
  investigationSurface?: boolean;
  icon?: string | Component | FunctionalComponent | (() => VNodeChild);
  orderNo?: number;
  hidden?: boolean;
  hiddenMenu?: boolean;
  hiddenBreadcrumb?: boolean;
  single?: boolean;
  expanded?: boolean;
  frameSrc?: string;
  frameBlank?: boolean;
  keepAlive?: boolean;
  footer?: false | PageFooterMeta;
}

export interface NavigationAncestor {
  code: string;
  path: string;
  title: LocalizedTitle;
}

export interface NavigationSection {
  key: string;
  title: LocalizedTitle;
}

export interface MenuRoute extends Omit<RouteRecordRaw, 'children' | 'meta'> {
  children?: MenuRoute[];
  meta?: AppRouteMeta;
  title?: LocalizedTitle;
  icon?: AppRouteMeta['icon'];
}

export interface TRouterInfo {
  tabKey?: string;
  path: string;
  fullPath?: string;
  routeIdx?: number;
  title?: LocalizedTitle;
  /** `route` 标题随实时路由元数据刷新，`runtime` 标题由页面生成并跨同路由更新保留。 */
  titleSource?: 'route' | 'runtime';
  name?: RouteRecordName | null;
  isHome?: boolean;
  isAlive?: boolean;
  isPinned?: boolean;
  isDuplicate?: boolean;
  duplicatedFrom?: string;
  query?: LocationQueryRaw;
  params?: RouteParamsRaw;
  meta?: AppRouteMeta;
}

export type TabPageSnapshot = Record<string, unknown>;

export interface TTabRouterType {
  tabRouterList: TRouterInfo[];
  closedTabStack: TRouterInfo[];
  activeTabKey: string;
  refreshingTabKey?: string;
  refreshNonceByTabKey: Record<string, number>;
  pageSnapshots: Record<string, TabPageSnapshot>;
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
