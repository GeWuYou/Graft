export const UPDATE_ROUTE_PATH = {
  CENTER: '/platform/updates',
} as const;

export const UPDATE_API_PATH = {
  STATUS: '/api/platform/updates/status',
  CHECK: '/api/platform/updates/check',
  OPERATIONS: '/api/platform/updates/operations',
  ACTIVE_OPERATION: '/api/platform/updates/active-operation',
  OPERATION_EVENTS: '/api/platform/updates/operations/{operationID}/events',
  OPERATION_DIAGNOSTIC: '/api/platform/updates/operations/{operationID}/diagnostic',
} as const;
