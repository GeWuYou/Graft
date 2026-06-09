// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { LocationQuery } from 'vue-router';

import { buildLogListLocation, parseLogRouteQuery } from '@/shared/observability';

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

export function parseAccessLogRouteQuery(query: LocationQuery | AccessLogRouteQuery): AccessLogRouteQuery {
  return parseLogRouteQuery<AccessLogRouteQuery>(query, ACCESS_LOG_QUERY_KEYS);
}

export function buildAccessLogLocation(query: AccessLogRouteQuery) {
  return buildLogListLocation(ACCESS_LOG_ROUTE_PATH.LIST, ACCESS_LOG_QUERY_KEYS, query);
}

export function buildAccessLogRequestLocation(requestId: string) {
  return buildAccessLogLocation({ request_id: requestId });
}

void (null as unknown as AccessLogQueryKey);
