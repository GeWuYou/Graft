import type { BootstrapRouteRegistration } from '@/modules/types';
import { localizeRouteTitleKey } from '@/utils/route/title';

import { MONITOR_ROUTE_PATH } from './contract/paths';

const overviewRouteTitle = localizeRouteTitleKey('monitor.route.overview.title');
const overviewBreadcrumbTitle = localizeRouteTitleKey('monitor.route.overview.breadcrumb');
const runtimeRouteTitle = localizeRouteTitleKey('monitor.route.runtime.title');
const runtimeBreadcrumbTitle = localizeRouteTitleKey('monitor.route.runtime.breadcrumb');
const dependenciesRouteTitle = localizeRouteTitleKey('monitor.route.dependencies.title');
const dependenciesBreadcrumbTitle = localizeRouteTitleKey('monitor.route.dependencies.breadcrumb');
const requestPerformanceRouteTitle = localizeRouteTitleKey('monitor.route.requestPerformance.title');
const requestPerformanceBreadcrumbTitle = localizeRouteTitleKey('monitor.route.requestPerformance.breadcrumb');
const moduleRuntimeRouteTitle = localizeRouteTitleKey('monitor.route.moduleRuntime.title');
const moduleRuntimeBreadcrumbTitle = localizeRouteTitleKey('monitor.route.moduleRuntime.breadcrumb');

export const monitorBootstrapRouteRegistrations: BootstrapRouteRegistration[] = [
  {
    menuPath: MONITOR_ROUTE_PATH.OBSERVABILITY_OVERVIEW,
    routeName: 'MonitorServerStatusOverview',
    loadPage: () => import('./pages/overview/index.vue'),
    meta: {
      domain: 'monitor',
      tabGroup: 'monitor',
      dashboard: true,
      pageKind: 'overview',
      semanticTitle: overviewRouteTitle,
      breadcrumbTitle: overviewBreadcrumbTitle,
      tabTitle: overviewRouteTitle,
    },
  },
  {
    menuPath: MONITOR_ROUTE_PATH.OBSERVABILITY_SERVICE_STATUS,
    routeName: 'MonitorServerStatusRuntime',
    loadPage: () => import('./pages/runtime/index.vue'),
    meta: {
      domain: 'monitor',
      tabGroup: 'monitor',
      dashboard: true,
      pageKind: 'runtime',
      semanticTitle: runtimeRouteTitle,
      breadcrumbTitle: runtimeBreadcrumbTitle,
      tabTitle: runtimeRouteTitle,
    },
  },
  {
    menuPath: MONITOR_ROUTE_PATH.OBSERVABILITY_DEPENDENCIES,
    routeName: 'MonitorServerStatusDependencies',
    loadPage: () => import('./pages/dependencies/index.vue'),
    meta: {
      domain: 'monitor',
      tabGroup: 'monitor',
      dashboard: true,
      pageKind: 'overview',
      semanticTitle: dependenciesRouteTitle,
      breadcrumbTitle: dependenciesBreadcrumbTitle,
      tabTitle: dependenciesRouteTitle,
    },
  },
  {
    menuPath: MONITOR_ROUTE_PATH.OBSERVABILITY_REQUEST_PERFORMANCE,
    routeName: 'MonitorRequestPerformance',
    loadPage: () => import('./pages/request-performance/index.vue'),
    meta: {
      domain: 'monitor',
      tabGroup: 'monitor',
      dashboard: true,
      pageKind: 'overview',
      semanticTitle: requestPerformanceRouteTitle,
      breadcrumbTitle: requestPerformanceBreadcrumbTitle,
      tabTitle: requestPerformanceRouteTitle,
    },
  },
  {
    menuPath: MONITOR_ROUTE_PATH.OBSERVABILITY_MODULES,
    routeName: 'MonitorModuleRuntimeOverview',
    loadPage: () => import('./pages/modules/index.vue'),
    meta: {
      domain: 'monitor',
      tabGroup: 'monitor',
      dashboard: true,
      pageKind: 'overview',
      pageSurface: 'paged-table',
      sidebarMotion: 'wide-table',
      semanticTitle: moduleRuntimeRouteTitle,
      breadcrumbTitle: moduleRuntimeBreadcrumbTitle,
      tabTitle: moduleRuntimeRouteTitle,
    },
  },
];
