// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { RouteRecordRaw } from 'vue-router';
import { createRouter, createWebHistory } from 'vue-router';

import { APP_RESULT_ROUTE_NAME, APP_RESULT_ROUTE_PATH } from '@/contracts/app/routes';
import { AUTH_ROUTE_NAME, AUTH_ROUTE_PATH } from '@/modules/auth/contract/routes';
import { PAGE_NOT_FOUND_ROUTE } from '@/utils/route/constant';
import { localizeRouteTitle } from '@/utils/route/title';

const env = import.meta.env.MODE || 'development';

const ROOT_ENTRY_ROUTE_NAME = 'RootEntry';
const ROOT_ENTRY_TITLE_KEY = 'app.home.title';
const ROOT_ENTRY_TITLE = localizeRouteTitle('Workspace', ROOT_ENTRY_TITLE_KEY);
const RESTRICTED_SESSION_ROUTE_NAME = AUTH_ROUTE_NAME.RESTRICTED_SESSION;
const RESTRICTED_SESSION_PATH = AUTH_ROUTE_PATH.RESTRICTED_SESSION;

const exceptionRouterList: Array<RouteRecordRaw> = [
  {
    path: APP_RESULT_ROUTE_PATH.FORBIDDEN,
    name: APP_RESULT_ROUTE_NAME.FORBIDDEN,
    component: () => import('@/app/result/403/index.vue'),
    meta: {
      hidden: true,
    },
  },
  {
    path: APP_RESULT_ROUTE_PATH.NOT_FOUND,
    name: APP_RESULT_ROUTE_NAME.NOT_FOUND,
    component: () => import('@/app/result/404/index.vue'),
    meta: {
      hidden: true,
    },
  },
  {
    path: APP_RESULT_ROUTE_PATH.SERVER_ERROR,
    name: APP_RESULT_ROUTE_NAME.SERVER_ERROR,
    component: () => import('@/app/result/500/index.vue'),
    meta: {
      hidden: true,
    },
  },
  {
    path: APP_RESULT_ROUTE_PATH.SUCCESS,
    name: APP_RESULT_ROUTE_NAME.SUCCESS,
    component: () => import('@/app/result/success/index.vue'),
    meta: {
      hidden: true,
    },
  },
  {
    path: APP_RESULT_ROUTE_PATH.FAIL,
    name: APP_RESULT_ROUTE_NAME.FAIL,
    component: () => import('@/app/result/fail/index.vue'),
    meta: {
      hidden: true,
    },
  },
  {
    path: APP_RESULT_ROUTE_PATH.NETWORK_ERROR,
    name: APP_RESULT_ROUTE_NAME.NETWORK_ERROR,
    component: () => import('@/app/result/network-error/index.vue'),
    meta: {
      hidden: true,
    },
  },
  {
    path: APP_RESULT_ROUTE_PATH.MAINTENANCE,
    name: APP_RESULT_ROUTE_NAME.MAINTENANCE,
    component: () => import('@/app/result/maintenance/index.vue'),
    meta: {
      hidden: true,
    },
  },
  {
    path: APP_RESULT_ROUTE_PATH.BROWSER_INCOMPATIBLE,
    name: APP_RESULT_ROUTE_NAME.BROWSER_INCOMPATIBLE,
    component: () => import('@/app/result/browser-incompatible/index.vue'),
    meta: {
      hidden: true,
    },
  },
];

const defaultRouterList: Array<RouteRecordRaw> = [
  {
    path: AUTH_ROUTE_PATH.LOGIN,
    name: AUTH_ROUTE_NAME.LOGIN,
    component: () => import('@/app/auth/index.vue'),
  },
  {
    path: RESTRICTED_SESSION_PATH,
    name: RESTRICTED_SESSION_ROUTE_NAME,
    component: () => import('@/layouts/index.vue'),
    meta: {
      hidden: true,
      hiddenBreadcrumb: true,
      keepAlive: false,
    },
  },
  {
    path: '/',
    name: ROOT_ENTRY_ROUTE_NAME,
    component: () => import('@/layouts/index.vue'),
    meta: {
      keepAlive: false,
      pageKind: 'overview',
      semanticTitle: ROOT_ENTRY_TITLE,
      tabTitle: ROOT_ENTRY_TITLE,
      title: ROOT_ENTRY_TITLE_KEY,
      titleKey: ROOT_ENTRY_TITLE_KEY,
    },
    children: [
      {
        path: '',
        name: `${ROOT_ENTRY_ROUTE_NAME}Index`,
        component: () => import('@/app/home/index.vue'),
        meta: {
          hidden: true,
          hiddenBreadcrumb: true,
          keepAlive: false,
          pageKind: 'overview',
          semanticTitle: ROOT_ENTRY_TITLE,
          tabTitle: ROOT_ENTRY_TITLE,
          title: ROOT_ENTRY_TITLE_KEY,
          titleKey: ROOT_ENTRY_TITLE_KEY,
        },
      },
    ],
  },
  PAGE_NOT_FOUND_ROUTE,
];

const staticRouterList = [...exceptionRouterList, ...defaultRouterList];

export const getActive = (maxLevel = 3): string => {
  // 非组件内调用必须通过Router实例获取当前路由
  const route = router.currentRoute.value;

  if (!route.path) {
    return '';
  }

  return route.path
    .split('/')
    .filter((_item: string, index: number) => index <= maxLevel && index > 0)
    .map((item: string) => `/${item}`)
    .join('');
};

const router = createRouter({
  history: createWebHistory(env === 'site' ? '/starter/vue-next/' : import.meta.env.VITE_BASE_URL),
  routes: staticRouterList,
  scrollBehavior() {
    return {
      el: '#app',
      top: 0,
      behavior: 'smooth',
    };
  },
});

export default router;
