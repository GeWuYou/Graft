import type { LocationQuery, LocationQueryValue } from 'vue-router';

import { ACCESS_LOG_ROUTE_PATH } from './paths';

export type AccessLogRouteQuery = Partial<{
  request_id: string;
  user_id: string;
  username: string;
  started_from: string;
  started_to: string;
  sort_by: string;
  sort_order: string;
}>;

const ACCESS_LOG_QUERY_KEYS = [
  'request_id',
  'user_id',
  'username',
  'started_from',
  'started_to',
  'sort_by',
  'sort_order',
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
