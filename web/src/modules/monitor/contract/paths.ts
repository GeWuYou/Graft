export const MONITOR_ROUTE_PATH = {
  SYSTEM: '/system',
  SYSTEM_OVERVIEW: '/system/overview',
  SYSTEM_RUNTIME: '/system/runtime',
  SYSTEM_DEPENDENCIES: '/system/dependencies',
  SYSTEM_MODULES: '/system/modules',
} as const;

export const MONITOR_API_PATH = {
  SERVER_STATUS: '/api/monitor/server-status',
  MODULE_RUNTIME: '/api/modules/runtime',
  MODULE_RUNTIME_DETAIL: '/api/modules/runtime/{module_key}',
} as const;

export function buildModuleRuntimeDetailApiPath(moduleKey: string) {
  return `${MONITOR_API_PATH.MODULE_RUNTIME}/${encodeURIComponent(moduleKey)}`;
}
