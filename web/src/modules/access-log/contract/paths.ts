export const ACCESS_LOG_ROUTE_PATH = {
  ROOT: '/logs',
  LIST: '/logs/access',
  DETAIL: '/logs/access/:id',
} as const;

export const ACCESS_LOG_API_PATH = {
  LIST: '/api/access-log',
  DETAIL: '/api/access-log/{id}',
} as const;
