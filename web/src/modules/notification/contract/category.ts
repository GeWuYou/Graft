import type { components } from '@/contracts/openapi/generated/schema';

export type NotificationCategory = components['schemas']['notification-category'];

const NOTIFICATION_CATEGORY = {
  SECURITY: 'SECURITY',
  TASK: 'TASK',
  CONFIG: 'CONFIG',
  OPERATIONS: 'OPERATIONS',
  SYSTEM: 'SYSTEM',
} as const satisfies Record<string, NotificationCategory>;

export const NOTIFICATION_CATEGORY_VALUES = Object.values(NOTIFICATION_CATEGORY);
