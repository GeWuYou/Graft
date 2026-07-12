export const RUNTIME_TARGET_ROUTE_PATH = { LIST: '/runtime-targets' } as const;
export const RUNTIME_TARGET_API_PATH = {
  LIST: '/api/runtime-targets',
  DETAIL: '/api/runtime-targets/{id}',
  REFRESH: '/api/runtime-targets/{id}/refresh',
} as const;

export function runtimeTargetDetailApiPath(id: number) {
  return RUNTIME_TARGET_API_PATH.DETAIL.replace('{id}', String(id));
}
export function runtimeTargetRefreshApiPath(id: number) {
  return RUNTIME_TARGET_API_PATH.REFRESH.replace('{id}', String(id));
}
