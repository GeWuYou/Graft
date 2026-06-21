import type { components } from '@/contracts/openapi/generated/schema';
import type { QuerySorter } from '@/shared/observability';

export type AppLogSeverity = components['schemas']['app-log-detail-response']['severity'];
export type AppLogItem = components['schemas']['app-log-detail-response'];
export type AppLogListResponse = components['schemas']['app-log-list-response'];
export type AppLogDetailResponse = components['schemas']['app-log-detail-response'];
export type AppLogBatchDeleteRequest = components['schemas']['app-log-batch-delete-request'];
export type AppLogSortBy = 'occurred_at' | 'severity' | 'component';
export type AppLogSortOrder = 'asc' | 'desc';
export type AppLogSorter = QuerySorter<AppLogSortBy>;

export type AppLogQuery = {
  page?: number;
  page_size?: number;
  occurred_from?: string;
  occurred_to?: string;
  severity?: AppLogSeverity;
  component?: string;
  operation?: string;
  request_id?: string;
  keyword?: string;
  message?: string;
  error?: string;
  sort?: string[];
};

export type AppLogFilterState = {
  keyword: string;
  occurredRange: string[];
  severity: '' | AppLogSeverity;
  component: string;
  operation: string;
  requestId: string;
  message: string;
  error: string;
  sorters: AppLogSorter[];
};
