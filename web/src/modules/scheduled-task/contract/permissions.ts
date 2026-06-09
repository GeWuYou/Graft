// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

export const SCHEDULED_TASK_PERMISSION_CODE = {
  READ: 'scheduled-task.read',
  CREATE: 'scheduled-task.create',
  UPDATE: 'scheduled-task.update',
  DELETE: 'scheduled-task.delete',
  RUN: 'scheduled-task.run',
  ENABLE: 'scheduled-task.enable',
} as const;

export type ScheduledTaskPermissionCode =
  (typeof SCHEDULED_TASK_PERMISSION_CODE)[keyof typeof SCHEDULED_TASK_PERMISSION_CODE];
