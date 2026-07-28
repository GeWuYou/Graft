import type { LocationQuery } from 'vue-router';

import { buildMonitorOriginQuery, type MonitorOriginContext } from '@/modules/monitor/contract/navigation';
import { buildLogListLocation, parseLogRouteQuery } from '@/shared/observability';

import { ACCESS_LOG_ROUTE_PATH } from './paths';

export type AccessLogRouteQuery = Partial<{
  quick_preset: string;
  keyword: string;
  request_id: string;
  user_id: string;
  username: string;
  method: string;
  path: string;
  path_match: string;
  route: string;
  status_code: string;
  status_group: string;
  duration_min_ms: string;
  duration_max_ms: string;
  started_from: string;
  started_to: string;
  occurred_from: string;
  occurred_to: string;
  sort: string | string[];
}>;

const ACCESS_LOG_QUERY_KEYS = [
  'quick_preset',
  'keyword',
  'request_id',
  'user_id',
  'username',
  'method',
  'path',
  'path_match',
  'route',
  'status_code',
  'status_group',
  'duration_min_ms',
  'duration_max_ms',
  'started_from',
  'started_to',
  'occurred_from',
  'occurred_to',
] as const;
type AccessLogQueryKey = (typeof ACCESS_LOG_QUERY_KEYS)[number];

/** 只保留已批准的日志筛选字段和 UI 快捷筛选上下文，避免 URL 查询参数成为第二套 API 契约。 */
export function parseAccessLogRouteQuery(query: LocationQuery | AccessLogRouteQuery): AccessLogRouteQuery {
  return parseLogRouteQuery<AccessLogRouteQuery>(query, ACCESS_LOG_QUERY_KEYS);
}

export function buildAccessLogLocation(query: AccessLogRouteQuery, monitorOrigin?: MonitorOriginContext | null) {
  const location = buildLogListLocation(ACCESS_LOG_ROUTE_PATH.LIST, ACCESS_LOG_QUERY_KEYS, query);
  return {
    ...location,
    query: {
      ...location.query,
      ...(monitorOrigin ? buildMonitorOriginQuery(monitorOrigin) : {}),
    },
  };
}

export function buildAccessLogRequestLocation(requestId: string, monitorOrigin?: MonitorOriginContext | null) {
  return buildAccessLogLocation({ request_id: requestId }, monitorOrigin);
}

void (null as unknown as AccessLogQueryKey);
