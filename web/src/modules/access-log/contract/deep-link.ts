import type { LocationQuery, LocationQueryValue } from 'vue-router';

import { ACCESS_LOG_ROUTE_PATH } from './paths';

export type AccessLogRouteQuery = Partial<{
  request_id: string;
  trace_id: string;
  user_id: string;
  username: string;
  occurred_from: string;
  occurred_to: string;
}>;

const ACCESS_LOG_QUERY_KEYS = [
  'request_id',
  'trace_id',
  'user_id',
  'username',
  'occurred_from',
  'occurred_to',
] as const;
type AccessLogQueryKey = (typeof ACCESS_LOG_QUERY_KEYS)[number];

function readQueryString(source: LocationQuery | AccessLogRouteQuery, key: AccessLogQueryKey) {
  const rawValue = source[key] as LocationQueryValue | LocationQueryValue[] | undefined;
  const candidate = Array.isArray(rawValue) ? rawValue.find((item) => typeof item === 'string') : rawValue;

  return typeof candidate === 'string' ? candidate.trim() : '';
}

export function parseAccessLogRouteQuery(query: LocationQuery | AccessLogRouteQuery): AccessLogRouteQuery {
  const parsedQuery: AccessLogRouteQuery = {};

  for (const key of ACCESS_LOG_QUERY_KEYS) {
    parsedQuery[key] = readQueryString(query, key);
  }

  return parsedQuery;
}

export function buildAccessLogLocation(query: AccessLogRouteQuery) {
  const normalizedQuery: Record<string, string> = {};
  const parsedQuery = parseAccessLogRouteQuery(query);

  ACCESS_LOG_QUERY_KEYS.forEach((key) => {
    const value = parsedQuery[key];
    if (value) {
      normalizedQuery[key] = value;
    }
  });

  return {
    path: ACCESS_LOG_ROUTE_PATH.LIST,
    query: normalizedQuery,
  };
}

export function buildAccessLogRequestLocation(requestId: string) {
  return buildAccessLogLocation({ request_id: requestId });
}

export function buildAccessLogTraceLocation(traceId: string) {
  return buildAccessLogLocation({ trace_id: traceId });
}
