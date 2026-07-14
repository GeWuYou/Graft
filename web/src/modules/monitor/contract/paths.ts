export const MONITOR_ROUTE_PATH = {
  OBSERVABILITY: '/observability',
  OBSERVABILITY_OVERVIEW: '/observability/overview',
  OBSERVABILITY_SERVICE_STATUS: '/observability/service-status',
  OBSERVABILITY_DEPENDENCIES: '/observability/dependencies',
  OBSERVABILITY_REQUEST_PERFORMANCE: '/observability/request-performance',
  OBSERVABILITY_MODULES: '/observability/modules',
} as const;

export const MONITOR_API_PATH = {
  SERVER_STATUS: '/api/monitor/server-status',
  REQUEST_PERFORMANCE: '/api/monitor/request-performance',
  MODULE_RUNTIME: '/api/modules/runtime',
  MODULE_RUNTIME_DETAIL: '/api/modules/runtime/{module_key}',
} as const;

export function buildModuleRuntimeDetailApiPath(moduleKey: string) {
  return `${MONITOR_API_PATH.MODULE_RUNTIME}/${encodeURIComponent(moduleKey)}`;
}
