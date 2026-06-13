// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

export const ANNOUNCEMENT_PERMISSION_CODE = {
  READ: 'announcement.read',
  CREATE: 'announcement.create',
  UPDATE: 'announcement.update',
  PUBLISH: 'announcement.publish',
  DELETE: 'announcement.delete',
} as const;

export type AnnouncementPermissionCode =
  (typeof ANNOUNCEMENT_PERMISSION_CODE)[keyof typeof ANNOUNCEMENT_PERMISSION_CODE];
