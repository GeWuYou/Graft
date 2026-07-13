export const ACCESS_LOG_ROUTE_PATH = {
  ROOT: '/observability',
  LIST: '/observability/access-logs',
  DETAIL: '/observability/access-logs/:id',
} as const;

export const ACCESS_LOG_API_PATH = {
  LIST: '/api/access-log',
  DETAIL: '/api/access-log/{id}',
  SAVED_VIEWS: '/api/access-log/saved-views',
  SAVED_VIEW: '/api/access-log/saved-views/{viewId}',
} as const;

/**
 * 构建指定已保存视图的访问日志 API 路径。
 *
 * @param viewId - 已保存视图的数字 ID
 * @returns 替换视图 ID 后的 API 路径
 */
export function buildAccessLogSavedViewApiPath(viewId: number) {
  return ACCESS_LOG_API_PATH.SAVED_VIEW.replace('{viewId}', String(viewId));
}
