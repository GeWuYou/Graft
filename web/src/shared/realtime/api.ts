import { request } from '@/utils/request';

export const REALTIME_API_PATH = {
  SUBSCRIPTIONS: '/api/realtime/subscriptions',
} as const;

export type RealtimeSubscriptionRequest = {
  topic: string;
};

export type RealtimeSubscriptionResponse = {
  topic: string;
  ticket: string;
  websocket_url: string;
  expires_at: string;
};

/**
 * 创建实时订阅。
 *
 * @param body - 订阅请求体，包含要订阅的主题
 * @returns 订阅信息，包括主题、ticket、websocket 地址和过期时间
 */
export function postRealtimeSubscription(body: RealtimeSubscriptionRequest) {
  return request.post<RealtimeSubscriptionResponse>({
    url: REALTIME_API_PATH.SUBSCRIPTIONS,
    data: body,
  }) as Promise<RealtimeSubscriptionResponse>;
}
