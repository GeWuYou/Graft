import type { components } from '@/contracts/openapi/generated/schema';
import type { QuerySorter } from '@/shared/observability';

export type AuditLogListItem = components['schemas']['audit-log-list-item'];
export type AuditLogListResponse = components['schemas']['audit-log-list-response'];
export type AuditOverviewItem = components['schemas']['AuditOverviewItem'];
export type AuditOverviewSummary = components['schemas']['AuditOverviewSummary'];
export type AuditOverviewResponse = components['schemas']['AuditOverviewResponse'];
export type AuditIncidentResponse = components['schemas']['AuditIncidentResponse'];
export type AuditIncidentSeed = AuditOverviewResponse['security_timeline'][number]['incident_seed'];
export type AuditIncidentSummary = AuditIncidentResponse['incident'];
export type AuditIncidentActor = AuditIncidentResponse['related_actors'][number];
export type AuditIncidentResource = AuditIncidentResponse['related_resources'][number];
export type AuditIncidentRequest = AuditIncidentResponse['related_requests'][number];
export type AuditIncidentMonitorContext = AuditIncidentResponse['monitor_context'];
export type EvidenceLink = components['schemas']['EvidenceLink'];

export type AuditOverviewWindow = '24h' | '7d' | '30d';
export type AuditRiskLevel = 'LOW' | 'MEDIUM' | 'HIGH' | 'CRITICAL';
export type AuditResult = 'SUCCESS' | 'FAILED' | 'DENIED' | 'ERROR';
export type AuditSource = 'REQUEST' | 'SECURITY_EVENT' | 'DOMAIN_EVENT';
export type AuditSortBy = 'created_at';
export type AuditSortOrder = 'asc' | 'desc';
export type AuditSorter = QuerySorter<AuditSortBy>;

export type AuditLogQuery = {
  page?: number;
  page_size?: number;
  actor_user_id?: number;
  action?: string;
  action_prefix?: string;
  source?: AuditSource;
  resource_type?: string;
  resource_id?: string;
  resource_name?: string;
  request_id?: string;
  result?: AuditResult;
  risk_level?: AuditRiskLevel;
  success?: boolean;
  created_from?: string;
  created_to?: string;
  sort_by?: AuditSortBy;
  sort_order?: AuditSortOrder;
};

export type AuditOverviewQuery = {
  window?: AuditOverviewWindow;
};
