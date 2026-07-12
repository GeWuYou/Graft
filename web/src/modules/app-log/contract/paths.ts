export const APP_LOG_ROUTE_PATH = {
  ROOT: '/observability',
  LIST: '/observability/application-logs',
  DETAIL: '/observability/application-logs/:id',
} as const;

export const APP_LOG_API_PATH = {
  LIST: '/api/app-log',
  DETAIL: '/api/app-log/{id}',
  BATCH_DELETE: '/api/app-log/batch-delete',
} as const;
