export const NOTIFICATION_PERMISSION_CODE = {
  VIEW: 'notification.view',
  READ: 'notification.read',
  MANAGE: 'notification.manage',
} as const;

export type NotificationPermissionCode =
  (typeof NOTIFICATION_PERMISSION_CODE)[keyof typeof NOTIFICATION_PERMISSION_CODE];
