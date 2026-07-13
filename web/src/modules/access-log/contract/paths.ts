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

export function buildAccessLogSavedViewApiPath(viewId: number) {
  return ACCESS_LOG_API_PATH.SAVED_VIEW.replace('{viewId}', String(viewId));
}
