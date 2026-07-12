export const MONITOR_ROUTE_PATH = {
  SYSTEM: '/observability',
  SYSTEM_OVERVIEW: '/observability/overview',
  SYSTEM_RUNTIME: '/observability/service-status',
  SYSTEM_DEPENDENCIES: '/observability/dependencies',
  SYSTEM_MODULES: '/observability/modules',
} as const;

export const MONITOR_API_PATH = {
  SERVER_STATUS: '/api/monitor/server-status',
  MODULE_RUNTIME: '/api/modules/runtime',
  MODULE_RUNTIME_DETAIL: '/api/modules/runtime/{module_key}',
} as const;

export function buildModuleRuntimeDetailApiPath(moduleKey: string) {
  return `${MONITOR_API_PATH.MODULE_RUNTIME}/${encodeURIComponent(moduleKey)}`;
}
