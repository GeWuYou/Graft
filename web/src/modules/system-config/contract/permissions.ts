export const SYSTEM_CONFIG_PERMISSION_CODE = {
  READ: 'system-config.read',
  WRITE: 'system-config.write',
} as const;

export type SystemConfigPermissionCode =
  (typeof SYSTEM_CONFIG_PERMISSION_CODE)[keyof typeof SYSTEM_CONFIG_PERMISSION_CODE];
