import type { components } from '@/contracts/openapi/generated/schema';

export type AuditLogListItem = components['schemas']['audit-log-list-item'];
export type AuditLogListResponse = components['schemas']['audit-log-list-response'];
export type AuditOverviewItem = components['schemas']['AuditOverviewItem'];
export type AuditOverviewSummary = components['schemas']['AuditOverviewSummary'];
export type AuditOverviewResponse = components['schemas']['AuditOverviewResponse'];

export type AuditOverviewWindow = '24h' | '7d' | '30d';

export type AuditLogQuery = {
  page?: number;
  page_size?: number;
  actor_user_id?: number;
  action?: string;
  resource_type?: string;
  resource_id?: string;
  resource_name?: string;
  request_id?: string;
  success?: boolean;
  created_from?: string;
  created_to?: string;
};

export type AuditOverviewQuery = {
  window?: AuditOverviewWindow;
};
