// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

export const NOTIFICATION_PERMISSION_CODE = {
  VIEW: 'notification.view',
  READ: 'notification.read',
  MANAGE: 'notification.manage',
} as const;

export type NotificationPermissionCode =
  (typeof NOTIFICATION_PERMISSION_CODE)[keyof typeof NOTIFICATION_PERMISSION_CODE];
