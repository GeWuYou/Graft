import type { RouteRecordRaw } from 'vue-router';

import type { AppRouteMeta } from '@/utils/types';

// BootstrapRouteRegistration 描述一个前端模块向壳层声明的 bootstrap 动态路由入口。
//
// 当前阶段只收敛最小运行面：模块显式声明菜单 path、稳定 route name 和页面加载器，
// 壳层再据此把后端 bootstrap 菜单装配成真实动态路由。
export type BootstrapRouteRegistration = {
  menuPath: string;
  routeName: string;
  loadPage: RouteRecordRaw['component'];
  meta?: Partial<AppRouteMeta>;
};

// GlobalRouteRegistration describes module-owned pages that live in the shell route runtime
// but are intentionally not sourced from the sidebar bootstrap menu tree.
export type GlobalRouteRegistration = {
  path: string;
  routeName: string;
  loadPage: RouteRecordRaw['component'];
  meta: AppRouteMeta;
};

// WebModuleRegistration 描述一个前端模块对壳层暴露的最小公共注册面。
//
// 当前阶段开放模块标识、bootstrap 动态路由声明，以及少量菜单外全局页面声明，
// 避免共享壳层直接依赖模块内部实现文件。
export type WebModuleRegistration = {
  moduleId: string;
  bootstrapRoutes: BootstrapRouteRegistration[];
  globalRoutes?: GlobalRouteRegistration[];
};
