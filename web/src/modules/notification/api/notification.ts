import { buildOpenApiRuntimePath, OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import type {
  NotificationItem,
  NotificationListQuery,
  NotificationListResponse,
  NotificationReadAllRequest,
  NotificationReadAllResponse,
  NotificationUnreadCountResponse,
} from '../types/notification';

type NotificationListPath = typeof OPENAPI_RUNTIME_PATH.getNotifications;
type GetNotificationsOperation = paths[NotificationListPath]['get'];
type GetNotificationsEnvelope = GetNotificationsOperation['responses'][200]['content']['application/json'];
type GetNotificationsData = NonNullable<GetNotificationsEnvelope['data']>;
type GetNotificationsQuery = NonNullable<GetNotificationsOperation['parameters']['query']>;

type NotificationUnreadCountPath = typeof OPENAPI_RUNTIME_PATH.getNotificationsUnreadCount;
type GetNotificationUnreadCountOperation = paths[NotificationUnreadCountPath]['get'];
type GetNotificationUnreadCountEnvelope =
  GetNotificationUnreadCountOperation['responses'][200]['content']['application/json'];
type GetNotificationUnreadCountData = NonNullable<GetNotificationUnreadCountEnvelope['data']>;

type NotificationReadPath = typeof OPENAPI_RUNTIME_PATH.postNotificationRead;
type PostNotificationReadOperation = paths[NotificationReadPath]['post'];
type PostNotificationReadEnvelope = PostNotificationReadOperation['responses'][200]['content']['application/json'];
type PostNotificationReadData = NonNullable<PostNotificationReadEnvelope['data']>;
type PostNotificationReadPathParams = PostNotificationReadOperation['parameters']['path'];

type NotificationReadAllPath = typeof OPENAPI_RUNTIME_PATH.postNotificationsReadAll;
type PostNotificationsReadAllOperation = paths[NotificationReadAllPath]['post'];
type PostNotificationsReadAllEnvelope =
  PostNotificationsReadAllOperation['responses'][200]['content']['application/json'];
type PostNotificationsReadAllData = NonNullable<PostNotificationsReadAllEnvelope['data']>;
type PostNotificationsReadAllBody = NonNullable<
  PostNotificationsReadAllOperation['requestBody']
>['content']['application/json'];

type NotificationDeletePath = typeof OPENAPI_RUNTIME_PATH.deleteNotification;
type DeleteNotificationOperation = paths[NotificationDeletePath]['delete'];
type DeleteNotificationPathParams = DeleteNotificationOperation['parameters']['path'];

export function getNotifications(query?: NotificationListQuery) {
  return request.get<GetNotificationsData>({
    url: OPENAPI_RUNTIME_PATH.getNotifications,
    params: normalizeNotificationListQuery(query),
  }) as Promise<NotificationListResponse>;
}

export function getNotificationUnreadCount() {
  return request.get<GetNotificationUnreadCountData>({
    url: OPENAPI_RUNTIME_PATH.getNotificationsUnreadCount,
  }) as Promise<NotificationUnreadCountResponse>;
}

export function markNotificationRead(deliveryId: PostNotificationReadPathParams['delivery_id']) {
  return request.post<PostNotificationReadData>({
    url: buildOpenApiRuntimePath('postNotificationRead', { delivery_id: deliveryId }),
  }) as Promise<NotificationItem>;
}

export function markNotificationsReadAll(payload?: NotificationReadAllRequest) {
  return request.post<PostNotificationsReadAllData>({
    url: OPENAPI_RUNTIME_PATH.postNotificationsReadAll,
    data: payload as PostNotificationsReadAllBody | undefined,
  }) as Promise<NotificationReadAllResponse>;
}

export function deleteNotification(deliveryId: DeleteNotificationPathParams['delivery_id']) {
  return request.delete<Record<string, never>>({
    url: buildOpenApiRuntimePath('deleteNotification', { delivery_id: deliveryId }),
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
