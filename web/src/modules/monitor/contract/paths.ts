export const MONITOR_ROUTE_PATH = {
  SERVER: '/system',
  SERVER_OVERVIEW: '/system/overview',
  SERVER_RUNTIME: '/system/runtime',
  SERVER_DEPENDENCIES: '/system/dependencies',
  SERVER_MODULES: '/system/modules',
} as const;

export const MONITOR_API_PATH = {
  SERVER_STATUS: '/api/monitor/server-status',
  MODULE_RUNTIME: '/api/modules/runtime',
  MODULE_RUNTIME_DETAIL: '/api/modules/runtime/{module_key}',
} as const;

export function buildModuleRuntimeDetailApiPath(moduleKey: string) {
  return `${MONITOR_API_PATH.MODULE_RUNTIME}/${encodeURIComponent(moduleKey)}`;
}
