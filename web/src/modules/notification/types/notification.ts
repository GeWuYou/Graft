import type { components } from '@/contracts/openapi/generated/schema';

export type NotificationItem = components['schemas']['notification-item'];
export type NotificationListResponse = components['schemas']['notification-list-response'];
export type NotificationUnreadCountResponse = components['schemas']['notification-unread-count-response'];
export type NotificationReadAllRequest = components['schemas']['notification-read-all-request'];
export type NotificationReadAllResponse = components['schemas']['notification-read-all-response'];
export type NotificationStatus = components['schemas']['notification-status'];
export type NotificationStatusFilter = 'all' | NotificationStatus;

export type NotificationListQuery = {
  category?: NotificationItem['category'];
  occurred_from?: string;
  occurred_to?: string;
  page?: number;
  page_size?: number;
  severity?: NotificationItem['severity'];
  source_module?: string;
  status?: NotificationStatusFilter;
};

export type NotificationFilterState = {
  category: '' | NotificationItem['category'];
  occurredRange: string[];
  severity: '' | NotificationItem['severity'];
  sourceModule: string;
  status: NotificationStatusFilter;
};
