import type { RouteRecordRaw } from 'vue-router';
import { createRouter, createWebHistory } from 'vue-router';

import { AUTH_ROUTE_NAME, AUTH_ROUTE_PATH } from '@/contracts/auth/routes';
import { BLANK_LAYOUT, PAGE_NOT_FOUND_ROUTE } from '@/utils/route/constant';

const env = import.meta.env.MODE || 'development';

export const ROOT_ENTRY_ROUTE_NAME = 'RootEntry';
export const RESTRICTED_SESSION_ROUTE_NAME = AUTH_ROUTE_NAME.RESTRICTED_SESSION;
export const RESTRICTED_SESSION_PATH = AUTH_ROUTE_PATH.RESTRICTED_SESSION;

const exceptionRouterList: Array<RouteRecordRaw> = [
  {
    path: '/result/403',
    name: 'Result403',
    component: () => import('@/app/result/403/index.vue'),
    meta: {
      hidden: true,
    },
  },
  {
    path: '/result/404',
    name: 'Result404',
    component: () => import('@/app/result/404/index.vue'),
    meta: {
      hidden: true,
    },
  },
  {
    path: '/result/500',
    name: 'Result500',
    component: () => import('@/app/result/500/index.vue'),
    meta: {
      hidden: true,
    },
  },
];

const defaultRouterList: Array<RouteRecordRaw> = [
  {
    path: '/login',
    name: 'login',
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
    component: BLANK_LAYOUT,
    meta: {
      hidden: true,
    },
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
