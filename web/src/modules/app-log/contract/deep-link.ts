import type { LocationQuery } from 'vue-router';

import { buildLogListLocation, parseLogRouteQuery } from '@/shared/observability';

import { APP_LOG_ROUTE_PATH } from './paths';

export type AppLogRouteQuery = Partial<{
  keyword: string;
  occurred_from: string;
  occurred_to: string;
  severity: string;
  component: string;
  operation: string;
  request_id: string;
  trace_id: string;
  message: string;
  error: string;
  sort: string | string[];
}>;

const APP_LOG_QUERY_KEYS = [
  'keyword',
  'occurred_from',
  'occurred_to',
  'severity',
  'component',
  'operation',
  'request_id',
  'trace_id',
  'message',
  'error',
] as const;

type AppLogQueryKey = (typeof APP_LOG_QUERY_KEYS)[number];

export function parseAppLogRouteQuery(query: LocationQuery | AppLogRouteQuery): AppLogRouteQuery {
  return parseLogRouteQuery<AppLogRouteQuery>(query, APP_LOG_QUERY_KEYS);
}

export function buildAppLogLocation(query: AppLogRouteQuery) {
  return buildLogListLocation(APP_LOG_ROUTE_PATH.LIST, APP_LOG_QUERY_KEYS, query);
}

void (null as unknown as AppLogQueryKey);
