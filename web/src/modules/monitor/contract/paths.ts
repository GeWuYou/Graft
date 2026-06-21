export const MONITOR_ROUTE_PATH = {
  SERVER: '/server',
  SERVER_OVERVIEW: '/server/overview',
  SERVER_RUNTIME: '/server/runtime',
  SERVER_DEPENDENCIES: '/server/dependencies',
  SERVER_MODULES: '/server/modules',
} as const;

export const MONITOR_API_PATH = {
  SERVER_STATUS: '/api/monitor/server-status',
  MODULE_RUNTIME: '/api/modules/runtime',
  MODULE_RUNTIME_DETAIL: '/api/modules/runtime/{module_key}',
} as const;

export function buildModuleRuntimeDetailApiPath(moduleKey: string) {
  return `${MONITOR_API_PATH.MODULE_RUNTIME}/${encodeURIComponent(moduleKey)}`;
}
