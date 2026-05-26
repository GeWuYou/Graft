export const RBAC_PERMISSION_CODE = {
  ROLE_READ: 'role.read',
  ROLE_CREATE: 'role.create',
  ROLE_UPDATE: 'role.update',
  ROLE_STATUS_UPDATE: 'role.status.update',
  ROLE_DELETE: 'role.delete',
  ROLE_PERMISSION_ASSIGN: 'role.permission.assign',
  ROLE_PERMISSION_MANAGE: 'role.permission.assign',
  PERMISSION_READ: 'permission.read',
  USER_ROLE_READ: 'user.role.read',
  USER_ROLE_ASSIGN: 'user.role.assign',
} as const;

export type RbacPermissionCode = (typeof RBAC_PERMISSION_CODE)[keyof typeof RBAC_PERMISSION_CODE];
