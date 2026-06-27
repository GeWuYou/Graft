export const AUDIT_PERMISSION_CODE = {
  READ: 'audit.read',
  MANAGE: 'audit.manage',
} as const;

export type AuditPermissionCode = (typeof AUDIT_PERMISSION_CODE)[keyof typeof AUDIT_PERMISSION_CODE];
