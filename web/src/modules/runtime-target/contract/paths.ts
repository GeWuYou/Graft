export const RUNTIME_TARGET_ROUTE_PATH = {
  LIST: '/infrastructure/runtime-targets',
  DETAIL: '/infrastructure/runtime-targets/:id',
} as const;

/**
 * 构建运行时目标的详情路由路径。
 *
 * @param id - 运行时目标的标识符
 * @returns 包含运行时目标标识符的详情路由路径
 */
export function runtimeTargetDetailPath(id: number) {
  return '/infrastructure/runtime-targets/' + id;
}

export const RUNTIME_TARGET_API_PATH = {
  LIST: '/api/runtime-targets',
  DETAIL: '/api/runtime-targets/{id}',
  REFRESH: '/api/runtime-targets/{id}/refresh',
  DISCOVER_LOCAL_DOCKER: '/api/runtime-targets/discover-local-docker',
} as const;

/**
 * 构建运行时目标详情接口的路径。
 *
 * @param id - 运行时目标的标识符
 * @returns 包含运行时目标标识符的详情接口路径
 */
export function runtimeTargetDetailApiPath(id: number) {
  return '/api/runtime-targets/' + id;
}

/**
 * 构建运行时目标的刷新接口路径。
 *
 * @param id - 运行时目标的标识符
 * @returns 对应运行时目标的刷新接口路径
 */
export function runtimeTargetRefreshApiPath(id: number) {
  return '/api/runtime-targets/' + id + '/refresh';
}
