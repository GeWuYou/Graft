export const RUNTIME_TARGET_ROUTE_PATH = { LIST: '/infrastructure/runtime-targets' } as const;
export const RUNTIME_TARGET_API_PATH = {
  LIST: '/api/runtime-targets',
  DETAIL: '/api/runtime-targets/{id}',
  REFRESH: '/api/runtime-targets/{id}/refresh',
  DISCOVER_LOCAL: '/api/runtime-targets/discover-local',
} as const;
