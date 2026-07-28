import type { LocationQuery } from 'vue-router';

import { buildLogListLocation, parseLogRouteQuery } from '@/shared/observability';

import { APP_LOG_ROUTE_PATH } from './paths';

export type AppLogRouteQuery = Partial<{
  quick_preset: string;
  keyword: string;
  occurred_from: string;
  occurred_to: string;
  severity: string;
  category: string;
  component: string;
  operation: string;
  request_id: string;
  message: string;
  error: string;
  sort: string | string[];
}>;

const APP_LOG_QUERY_KEYS = [
  'quick_preset',
  'keyword',
  'occurred_from',
  'occurred_to',
  'severity',
  'category',
  'component',
  'operation',
  'request_id',
  'message',
  'error',
] as const;

type AppLogQueryKey = (typeof APP_LOG_QUERY_KEYS)[number];

/** 深链只接收应用日志允许的筛选字段和快捷筛选上下文，并交给共享日志解析器完成规范化。 */
export function parseAppLogRouteQuery(query: LocationQuery | AppLogRouteQuery): AppLogRouteQuery {
  return parseLogRouteQuery<AppLogRouteQuery>(query, APP_LOG_QUERY_KEYS);
}

export function buildAppLogLocation(query: AppLogRouteQuery) {
  return buildLogListLocation(APP_LOG_ROUTE_PATH.LIST, APP_LOG_QUERY_KEYS, query);
}

void (null as unknown as AppLogQueryKey);
