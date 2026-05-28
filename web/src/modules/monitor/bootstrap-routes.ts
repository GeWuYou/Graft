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
        'zh-CN': '服务监控 · 概览',
        'en-US': 'Server Overview',
      },
      tabTitle: {
        'zh-CN': '服务监控 · 概览',
        'en-US': 'Server Overview',
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
        'zh-CN': '服务监控 · 运行时',
        'en-US': 'Server Runtime',
      },
      tabTitle: {
        'zh-CN': '服务监控 · 运行时',
        'en-US': 'Server Runtime',
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
        'zh-CN': '服务监控 · 依赖服务',
        'en-US': 'Server Dependencies',
      },
      tabTitle: {
        'zh-CN': '服务监控 · 依赖服务',
        'en-US': 'Server Dependencies',
      },
    },
  },
];
