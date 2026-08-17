import type { RouteRecordRaw } from 'vue-router';

import { localizeRouteTitleKey } from '@/utils/route/title';

// 开发构建才装配此路由文件，隔离本地设计预览与正式用户路由树。
export const developmentRouterList: RouteRecordRaw[] = [
  {
    path: '/mock/dashboard-preview',
    name: 'DevelopmentDashboardWorkbenchPreviewShell',
    component: () => import('@/layouts/index.vue'),
    meta: { hidden: true },
    children: [
      {
        path: '',
        name: 'DevelopmentDashboardWorkbenchPreview',
        component: () => import('@/modules/dashboard/pages/preview/index.vue'),
        meta: {
          hidden: true,
          hiddenBreadcrumb: true,
          keepAlive: false,
          pageKind: 'overview',
          semanticTitle: localizeRouteTitleKey('dashboard.workbench.routeTitle'),
          tabTitle: localizeRouteTitleKey('dashboard.workbench.routeTitle'),
        },
      },
    ],
  },
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
