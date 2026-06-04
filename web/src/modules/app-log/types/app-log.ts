import type { components } from '@/contracts/openapi/generated/schema';

export type AppLogSeverity = components['schemas']['app-log-detail-response']['severity'];
export type AppLogItem = components['schemas']['app-log-detail-response'];
export type AppLogListResponse = components['schemas']['app-log-list-response'];
export type AppLogDetailResponse = components['schemas']['AppLogDetailResponse'];

export type AppLogQuery = {
  page?: number;
  page_size?: number;
  occurred_from?: string;
  occurred_to?: string;
  severity?: AppLogSeverity;
  component?: string;
  operation?: string;
  request_id?: string;
  trace_id?: string;
  keyword?: string;
  message?: string;
  error?: string;
};

export type AppLogFilterState = {
  keyword: string;
  occurredRange: string[];
  severity: '' | AppLogSeverity;
  component: string;
  operation: string;
  requestId: string;
  traceId: string;
  message: string;
  error: string;
};
