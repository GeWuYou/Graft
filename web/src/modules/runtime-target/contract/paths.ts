export const RUNTIME_TARGET_ROUTE_PATH = {
  LIST: '/infrastructure/runtime-targets',
  DETAIL: '/infrastructure/runtime-targets/:id',
} as const;

export function runtimeTargetDetailPath(id: number) {
  return '/infrastructure/runtime-targets/' + id;
}

export const RUNTIME_TARGET_API_PATH = {
  LIST: '/api/runtime-targets',
  DETAIL: '/api/runtime-targets/{id}',
  REFRESH: '/api/runtime-targets/{id}/refresh',
  DISCOVER_LOCAL_DOCKER: '/api/runtime-targets/discover-local-docker',
} as const;

export function runtimeTargetDetailApiPath(id: number) {
  return '/api/runtime-targets/' + id;
}

export function runtimeTargetRefreshApiPath(id: number) {
  return '/api/runtime-targets/' + id + '/refresh';
}
