import type { BootstrapRouteRegistration } from '@/modules/types';

import { MONITOR_ROUTE_PATH } from './contract/paths';

export const monitorBootstrapRouteRegistrations: BootstrapRouteRegistration[] = [
  {
    menuPath: MONITOR_ROUTE_PATH.SERVER_OVERVIEW,
    routeName: 'MonitorServerStatusOverview',
    loadPage: () => import('./pages/overview/index.vue'),
  },
  {
    menuPath: MONITOR_ROUTE_PATH.SERVER_RUNTIME,
    routeName: 'MonitorServerStatusRuntime',
    loadPage: () => import('./pages/runtime/index.vue'),
  },
  {
    menuPath: MONITOR_ROUTE_PATH.SERVER_DEPENDENCIES,
    routeName: 'MonitorServerStatusDependencies',
    loadPage: () => import('./pages/dependencies/index.vue'),
  },
];
