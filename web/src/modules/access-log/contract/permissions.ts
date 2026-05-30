export const ACCESS_LOG_PERMISSION_CODE = {
  READ: 'access_log.read',
} as const;

export type AccessLogPermissionCode = (typeof ACCESS_LOG_PERMISSION_CODE)[keyof typeof ACCESS_LOG_PERMISSION_CODE];
