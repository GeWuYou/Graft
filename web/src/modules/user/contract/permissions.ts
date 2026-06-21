export const USER_PERMISSION_CODE = {
  READ: 'user.read',
  CREATE: 'user.create',
  UPDATE: 'user.update',
  DISABLE: 'user.disable',
  SESSION_READ: 'user.session.read',
  SESSION_REVOKE: 'user.session.revoke',
} as const;

export type UserPermissionCode = (typeof USER_PERMISSION_CODE)[keyof typeof USER_PERMISSION_CODE];
