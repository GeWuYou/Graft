import type { RouteRecordRaw } from 'vue-router';

// 开发构建才装配此路由，隔离本地更新中心预览与正式用户路由树。
export const developmentRouterList: RouteRecordRaw[] = [
  {
    path: '/mock',
    name: 'DevelopmentUpdatePreviewShell',
    component: () => import('@/modules/update/pages/preview/shell.vue'),
    meta: { hidden: true },
    children: [
      {
        path: 'platform/updates',
        name: 'DevelopmentUpdatePreview',
        component: () => import('@/modules/update/pages/preview/index.vue'),
        meta: { hidden: true },
      },
    ],
  },
];
