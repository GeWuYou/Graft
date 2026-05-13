import type { RouteRecordRaw } from 'vue-router';

/**
 * Static routes exist before backend menus are available.
 * Once the server exposes menu + permission metadata, dynamic routes should be merged
 * on top of this shell instead of replacing the login and baseline dashboard entries.
 */
export const staticRoutes: RouteRecordRaw[] = [
  {
    path: '/login',
    component: () => import('@/layouts/AuthLayout.vue'),
    children: [
      {
        path: '',
        name: 'login',
        component: () => import('@/pages/LoginPage.vue'),
        meta: {
          title: '登录',
          hideInMenu: true,
        },
      },
    ],
  },
  {
    path: '/unauthorized',
    component: () => import('@/layouts/AuthLayout.vue'),
    children: [
      {
        path: '',
        name: 'unauthorized',
        component: () => import('@/pages/UnauthorizedPage.vue'),
        meta: {
          title: '无权限访问',
          hideInMenu: true,
          requiresAuth: true,
        },
      },
    ],
  },
  {
    path: '/',
    component: () => import('@/layouts/BasicLayout.vue'),
    meta: {
      title: '工作台',
      requiresAuth: true,
      hideInMenu: true,
    },
    children: [
      {
        path: '',
        redirect: {
          name: 'dashboard',
        },
      },
      {
        path: 'dashboard',
        name: 'dashboard',
        component: () => import('@/pages/DashboardPage.vue'),
        meta: {
          title: '仪表盘',
          requiresAuth: true,
          icon: 'dashboard',
          permission: 'dashboard.view',
          plugin: 'core',
        },
      },
    ],
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'not-found',
    component: () => import('@/pages/NotFoundPage.vue'),
    meta: {
      title: '页面不存在',
      hideInMenu: true,
    },
  },
];
