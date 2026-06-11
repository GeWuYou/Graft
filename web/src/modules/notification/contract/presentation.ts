// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

export const NOTIFICATION_SOURCE_LABEL_KEY_BY_MODULE: Record<string, string> = {
  'access-log': 'notification.source.accessLog',
  audit: 'notification.source.audit',
  notification: 'notification.source.notification',
  scheduler: 'notification.source.scheduler',
  'system-config': 'notification.source.systemConfig',
};

export const NOTIFICATION_CATEGORY_LABEL_KEY_BY_VALUE: Record<string, string> = {
  CONFIG: 'notification.category.config',
  OPERATIONS: 'notification.category.operations',
  SECURITY: 'notification.category.security',
  SYSTEM: 'notification.category.system',
  TASK: 'notification.category.task',
};

export const NOTIFICATION_RESOURCE_TYPE_LABEL_KEY_BY_VALUE: Record<string, string> = {
  scheduled_task_run: 'notification.resourceType.scheduledTaskRun',
};
