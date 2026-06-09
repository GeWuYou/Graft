// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { components } from '@/contracts/openapi/generated/schema';
import type { QuerySorter } from '@/shared/observability';

export type AccessLogItem = components['schemas']['access-log-detail-response'];
export type AccessLogListResponse = components['schemas']['access-log-list-response'];
export type AccessLogDetailResponse = components['schemas']['AccessLogDetailResponse'];

export type AccessLogSortBy = 'started_at' | 'occurred_at' | 'duration_ms' | 'status_code';
export type AccessLogSortOrder = 'asc' | 'desc';
export type AccessLogPathMatch = 'exact' | 'prefix';
export type AccessLogSorter = QuerySorter<AccessLogSortBy>;

export type AccessLogQuery = {
  page?: number;
  page_size?: number;
  keyword?: string;
  request_id?: string;
  user_id?: number;
  username?: string;
  method?: string;
  path?: string;
  path_match?: AccessLogPathMatch;
  route?: string;
  status_code?: number;
  status_group?: '4xx' | '5xx';
  duration_min_ms?: number;
  duration_max_ms?: number;
  started_from?: string;
  started_to?: string;
  occurred_from?: string;
  occurred_to?: string;
  sort?: string[];
};

export type AccessLogFilterState = {
  keyword: string;
  requestId: string;
  userId: string;
  username: string;
  method: string;
  path: string;
  pathMatch: AccessLogPathMatch;
  route: string;
  statusCode: string;
  durationMinMs: string;
  durationMaxMs: string;
  startedRange: string[];
  occurredRange: string[];
  sorters: AccessLogSorter[];
};
