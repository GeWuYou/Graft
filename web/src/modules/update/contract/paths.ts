export const UPDATE_ROUTE_PATH = {
  CENTER: '/platform/updates',
} as const;

export const UPDATE_API_PATH = {
  STATUS: '/api/platform/updates/status',
  CHECK: '/api/platform/updates/check',
  OPERATIONS: '/api/platform/updates/operations',
  OPERATION_DIAGNOSTIC: '/api/platform/updates/operations/{operationID}/diagnostic',
} as const;
