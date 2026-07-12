export const RUNTIME_TARGET_ROUTE_PATH = { LIST: '/infrastructure/runtime-targets' } as const;
export const RUNTIME_TARGET_API_PATH = {
  LIST: '/api/runtime-targets',
  DETAIL: '/api/runtime-targets/{id}',
  REFRESH: '/api/runtime-targets/{id}/refresh',
} as const;

/**
 * 构建运行目标详情接口路径。
 *
 * @param id - 运行目标的标识
 * @returns 包含指定运行目标标识的详情接口路径
 */
export function runtimeTargetDetailApiPath(id: number) {
  return RUNTIME_TARGET_API_PATH.DETAIL.replace('{id}', String(id));
}
/**
 * 构造运行目标的刷新接口路径。
 *
 * @param id - 运行目标的标识符
 * @returns 包含指定运行目标标识符的刷新接口路径
 */
export function runtimeTargetRefreshApiPath(id: number) {
  return RUNTIME_TARGET_API_PATH.REFRESH.replace('{id}', String(id));
}
