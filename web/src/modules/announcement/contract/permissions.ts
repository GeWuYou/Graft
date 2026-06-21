export const ANNOUNCEMENT_PERMISSION_CODE = {
  READ: 'announcement.read',
  CREATE: 'announcement.create',
  UPDATE: 'announcement.update',
  PUBLISH: 'announcement.publish',
  DELETE: 'announcement.delete',
} as const;

export type AnnouncementPermissionCode =
  (typeof ANNOUNCEMENT_PERMISSION_CODE)[keyof typeof ANNOUNCEMENT_PERMISSION_CODE];
