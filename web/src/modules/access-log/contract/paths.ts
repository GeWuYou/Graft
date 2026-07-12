export const ACCESS_LOG_ROUTE_PATH = {
  ROOT: '/observability',
  LIST: '/observability/access-logs',
  DETAIL: '/observability/access-logs/:id',
} as const;

export const ACCESS_LOG_API_PATH = {
  LIST: '/api/access-log',
  DETAIL: '/api/access-log/{id}',
} as const;
