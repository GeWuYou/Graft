export const MONITOR_ROUTE_PATH = {
  SERVER_STATUS: '/monitor/server-status',
  SERVER_STATUS_OVERVIEW: '/monitor/server-status/overview',
  SERVER_STATUS_RUNTIME: '/monitor/server-status/runtime',
  SERVER_STATUS_DEPENDENCIES: '/monitor/server-status/dependencies',
} as const;

export const MONITOR_API_PATH = {
  SERVER_STATUS: '/api/monitor/server-status',
} as const;
