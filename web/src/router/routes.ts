import type { RouteRecordRaw } from 'vue-router';

/**
 * 这些静态路由在后端菜单返回前先提供壳层入口。
 * 后续接入服务端菜单与权限元数据时，应在这层壳上合并动态路由，
 * 而不是替换登录页和基础仪表盘等保底入口。
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
          titleKey: 'routes.login',
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
          titleKey: 'routes.unauthorized',
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
      titleKey: 'routes.workspace',
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
          titleKey: 'navigation.dashboard',
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
      titleKey: 'routes.notFound',
      hideInMenu: true,
    },
  },
];
