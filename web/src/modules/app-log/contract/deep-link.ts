import type { LocationQuery, LocationQueryValue } from 'vue-router';

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

function readQueryString(source: LocationQuery | AppLogRouteQuery, key: AppLogQueryKey) {
  const values = ([] as LocationQueryValue[]).concat(source[key] ?? []);
  const candidate = values.find((item) => typeof item === 'string');

  return typeof candidate === 'string' ? candidate.trim() : '';
}

export function parseAppLogRouteQuery(query: LocationQuery | AppLogRouteQuery): AppLogRouteQuery {
  return Object.fromEntries(APP_LOG_QUERY_KEYS.map((key) => [key, readQueryString(query, key)])) as AppLogRouteQuery;
}

export function buildAppLogLocation(query: AppLogRouteQuery) {
  const normalizedQuery: Record<string, string> = {};
  const parsedQuery = parseAppLogRouteQuery(query);

  APP_LOG_QUERY_KEYS.forEach((key) => {
    const value = parsedQuery[key];
    if (value) {
      normalizedQuery[key] = value;
    }
  });

  return {
    path: APP_LOG_ROUTE_PATH.LIST,
    query: normalizedQuery,
  };
}
