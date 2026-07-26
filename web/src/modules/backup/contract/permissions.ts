export const BACKUP_PERMISSION_CODE = {
  READ: 'platform-backup.read',
  CREATE: 'platform-backup.create',
} as const;

export type BackupPermissionCode = (typeof BACKUP_PERMISSION_CODE)[keyof typeof BACKUP_PERMISSION_CODE];
