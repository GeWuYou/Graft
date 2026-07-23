export const UPDATE_PERMISSION_CODE = {
  READ: 'platform-update.read',
  CHECK: 'platform-update.check',
  MANAGE: 'platform-update.manage',
} as const;

export type UpdatePermissionCode = (typeof UPDATE_PERMISSION_CODE)[keyof typeof UPDATE_PERMISSION_CODE];
