export const APP_LOG_ROUTE_PATH = {
  ROOT: '/observability',
  LIST: '/observability/application-logs',
  DETAIL: '/observability/application-logs/:id',
} as const;

export const APP_LOG_API_PATH = {
  LIST: '/api/app-log',
  DETAIL: '/api/app-log/{id}',
  BATCH_DELETE: '/api/app-log/batch-delete',
  SAVED_VIEWS: '/api/app-log/saved-views',
  SAVED_VIEW: '/api/app-log/saved-views/{viewId}',
} as const;

/**
 * 构建指定应用日志保存视图的 API 路径。
 *
 * @param viewId - 保存视图的数字标识
 * @returns 包含指定保存视图标识的 API 路径
 */
export function buildAppLogSavedViewApiPath(viewId: number) {
  return APP_LOG_API_PATH.SAVED_VIEW.replace('{viewId}', String(viewId));
}
