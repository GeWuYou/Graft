export const BUILD_PERMISSION_CODE = {
  READ: 'build.read',
  CREATE: 'build.create',
  CANCEL: 'build.cancel',
  RETRY: 'build.retry',
} as const;

export type BuildPermissionCode = (typeof BUILD_PERMISSION_CODE)[keyof typeof BUILD_PERMISSION_CODE];
