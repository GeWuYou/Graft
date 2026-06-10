// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import { buildNotificationDeleteApiPath, buildNotificationReadApiPath, NOTIFICATION_API_PATH } from '../contract/paths';
import type {
  NotificationItem,
  NotificationListQuery,
  NotificationListResponse,
  NotificationReadAllRequest,
  NotificationReadAllResponse,
  NotificationUnreadCountResponse,
} from '../types/notification';

type NotificationListPath = (typeof NOTIFICATION_API_PATH)['LIST'];
type GetNotificationsOperation = paths[NotificationListPath]['get'];
type GetNotificationsEnvelope = GetNotificationsOperation['responses'][200]['content']['application/json'];
type GetNotificationsData = NonNullable<GetNotificationsEnvelope['data']>;
type GetNotificationsQuery = NonNullable<GetNotificationsOperation['parameters']['query']>;

type NotificationUnreadCountPath = (typeof NOTIFICATION_API_PATH)['UNREAD_COUNT'];
type GetNotificationUnreadCountOperation = paths[NotificationUnreadCountPath]['get'];
type GetNotificationUnreadCountEnvelope =
  GetNotificationUnreadCountOperation['responses'][200]['content']['application/json'];
type GetNotificationUnreadCountData = NonNullable<GetNotificationUnreadCountEnvelope['data']>;

type NotificationReadPath = (typeof NOTIFICATION_API_PATH)['READ'];
type PostNotificationReadOperation = paths[NotificationReadPath]['post'];
type PostNotificationReadEnvelope = PostNotificationReadOperation['responses'][200]['content']['application/json'];
type PostNotificationReadData = NonNullable<PostNotificationReadEnvelope['data']>;
type PostNotificationReadPathParams = PostNotificationReadOperation['parameters']['path'];

type NotificationReadAllPath = (typeof NOTIFICATION_API_PATH)['READ_ALL'];
type PostNotificationsReadAllOperation = paths[NotificationReadAllPath]['post'];
type PostNotificationsReadAllEnvelope =
  PostNotificationsReadAllOperation['responses'][200]['content']['application/json'];
type PostNotificationsReadAllData = NonNullable<PostNotificationsReadAllEnvelope['data']>;
type PostNotificationsReadAllBody = NonNullable<
  PostNotificationsReadAllOperation['requestBody']
>['content']['application/json'];

type NotificationDeletePath = (typeof NOTIFICATION_API_PATH)['DELETE'];
type DeleteNotificationOperation = paths[NotificationDeletePath]['delete'];
type DeleteNotificationPathParams = DeleteNotificationOperation['parameters']['path'];

export function getNotifications(query?: NotificationListQuery) {
  return request.get<GetNotificationsData>({
    url: NOTIFICATION_API_PATH.LIST,
    params: normalizeNotificationListQuery(query),
  }) as Promise<NotificationListResponse>;
}

export function getNotificationUnreadCount() {
  return request.get<GetNotificationUnreadCountData>({
    url: NOTIFICATION_API_PATH.UNREAD_COUNT,
  }) as Promise<NotificationUnreadCountResponse>;
}

export function markNotificationRead(deliveryId: PostNotificationReadPathParams['delivery_id']) {
  return request.post<PostNotificationReadData>({
    url: buildNotificationReadApiPath(deliveryId),
  }) as Promise<NotificationItem>;
}

export function markNotificationsReadAll(payload?: NotificationReadAllRequest) {
  return request.post<PostNotificationsReadAllData>({
    url: NOTIFICATION_API_PATH.READ_ALL,
    data: payload as PostNotificationsReadAllBody | undefined,
  }) as Promise<NotificationReadAllResponse>;
}

export function deleteNotification(deliveryId: DeleteNotificationPathParams['delivery_id']) {
  return request.delete<Record<string, never>>({
    url: buildNotificationDeleteApiPath(deliveryId),
  });
}

function normalizeNotificationListQuery(query?: NotificationListQuery): GetNotificationsQuery | undefined {
  if (!query) {
    return undefined;
  }

  const { status, ...params } = query;
  return {
    ...params,
    ...(status && status !== 'all' ? { status } : {}),
  } satisfies GetNotificationsQuery;
}
