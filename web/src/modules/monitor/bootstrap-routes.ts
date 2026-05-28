import type { BootstrapRouteRegistration } from '@/modules/types';

import { MONITOR_ROUTE_PATH } from './contract/paths';

export const monitorBootstrapRouteRegistrations: BootstrapRouteRegistration[] = [
  {
    menuPath: MONITOR_ROUTE_PATH.SERVER_OVERVIEW,
    routeName: 'MonitorServerStatusOverview',
    loadPage: () => import('./pages/overview/index.vue'),
    meta: {
      domain: 'monitor',
      tabGroup: 'monitor',
      dashboard: true,
      pageKind: 'overview',
      semanticTitle: {
        'zh-CN': '服务管理 - 概览',
        'en-US': 'Service Management - Overview',
      },
      breadcrumbTitle: {
        'zh-CN': '概览',
        'en-US': 'Overview',
      },
      tabTitle: {
        'zh-CN': '服务管理 - 概览',
        'en-US': 'Service Management - Overview',
      },
    },
  },
  {
    menuPath: MONITOR_ROUTE_PATH.SERVER_RUNTIME,
    routeName: 'MonitorServerStatusRuntime',
    loadPage: () => import('./pages/runtime/index.vue'),
    meta: {
      domain: 'monitor',
      tabGroup: 'monitor',
      dashboard: true,
      pageKind: 'runtime',
      semanticTitle: {
        'zh-CN': '服务管理 - 运行时',
        'en-US': 'Service Management - Runtime',
      },
      breadcrumbTitle: {
        'zh-CN': '运行时',
        'en-US': 'Runtime',
      },
      tabTitle: {
        'zh-CN': '服务管理 - 运行时',
        'en-US': 'Service Management - Runtime',
      },
    },
  },
  {
    menuPath: MONITOR_ROUTE_PATH.SERVER_DEPENDENCIES,
    routeName: 'MonitorServerStatusDependencies',
    loadPage: () => import('./pages/dependencies/index.vue'),
    meta: {
      domain: 'monitor',
      tabGroup: 'monitor',
      dashboard: true,
      pageKind: 'overview',
      semanticTitle: {
        'zh-CN': '服务管理 - 依赖服务',
        'en-US': 'Service Management - Dependencies',
      },
      breadcrumbTitle: {
        'zh-CN': '依赖服务',
        'en-US': 'Dependencies',
      },
      tabTitle: {
        'zh-CN': '服务管理 - 依赖服务',
        'en-US': 'Service Management - Dependencies',
      },
    },
  },
];
