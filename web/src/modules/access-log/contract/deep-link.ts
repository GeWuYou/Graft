import type { LocationQuery, LocationQueryValue } from 'vue-router';

import { ACCESS_LOG_ROUTE_PATH } from './paths';

export type AccessLogRouteQuery = Partial<{
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

  const rawSort = query.sort as LocationQueryValue | LocationQueryValue[] | undefined;
  if (Array.isArray(rawSort)) {
    parsedQuery.sort = rawSort
      .filter((item): item is string => typeof item === 'string')
      .map((item) => item.trim())
      .filter(Boolean);
  } else if (typeof rawSort === 'string' && rawSort.trim()) {
    parsedQuery.sort = rawSort.trim();
  }

  return parsedQuery;
}

export function buildAccessLogLocation(query: AccessLogRouteQuery) {
  const normalizedQuery: Record<string, string | string[]> = {};
  const parsedQuery = parseAccessLogRouteQuery(query);

  ACCESS_LOG_QUERY_KEYS.forEach((key) => {
    const value = parsedQuery[key];
    if (value) {
      normalizedQuery[key] = value;
    }
  });

  if (Array.isArray(parsedQuery.sort)) {
    const sortValues = parsedQuery.sort.filter((item): item is string => Boolean(item));
    if (sortValues.length) {
      normalizedQuery.sort = sortValues;
    }
  } else if (typeof parsedQuery.sort === 'string' && parsedQuery.sort) {
    normalizedQuery.sort = [parsedQuery.sort];
  }

  return {
    path: ACCESS_LOG_ROUTE_PATH.LIST,
    query: normalizedQuery,
  };
}

export function buildAccessLogRequestLocation(requestId: string) {
  return buildAccessLogLocation({ request_id: requestId });
}
