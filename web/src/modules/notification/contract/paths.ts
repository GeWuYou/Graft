// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

export const NOTIFICATION_ROUTE_PATH = {
  LIST: '/notifications',
} as const;

export const NOTIFICATION_API_PATH = {
  LIST: '/api/notifications',
  UNREAD_COUNT: '/api/notifications/unread-count',
  READ: '/api/notifications/{delivery_id}/read',
  READ_ALL: '/api/notifications/read-all',
  DELETE: '/api/notifications/{delivery_id}',
} as const;

export function buildNotificationReadApiPath(deliveryId: number) {
  return NOTIFICATION_API_PATH.READ.replace('{delivery_id}', String(deliveryId));
}

export function buildNotificationDeleteApiPath(deliveryId: number) {
  return NOTIFICATION_API_PATH.DELETE.replace('{delivery_id}', String(deliveryId));
}
