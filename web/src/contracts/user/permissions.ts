export const USER_PERMISSION_CODE = {
  CREATE: 'user.create',
  UPDATE: 'user.update',
  DISABLE: 'user.disable',
} as const;

export type UserPermissionCode = (typeof USER_PERMISSION_CODE)[keyof typeof USER_PERMISSION_CODE];
