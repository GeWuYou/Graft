export const BACKUP_ROUTE_PATH = {
  LIST: '/platform/backups',
} as const;

export const BACKUP_API_PATH = {
  LIST: '/api/platform/backups',
  DETAIL: '/api/platform/backups/:id',
} as const;
