export const SCHEDULED_TASK_PERMISSION_CODE = {
  READ: 'scheduled-task.read',
  RUN: 'scheduled-task.run',
} as const;

export type ScheduledTaskPermissionCode =
  (typeof SCHEDULED_TASK_PERMISSION_CODE)[keyof typeof SCHEDULED_TASK_PERMISSION_CODE];
