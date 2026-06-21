import type { components } from '@/contracts/openapi/generated/schema';
import type { QuerySorter } from '@/shared/observability';

export type AuditLogListItem = components['schemas']['audit-log-list-item'];
export type AuditLogDetailResponse = components['schemas']['audit-log-detail-response'];
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
export type AppliedDrilldownScope = components['schemas']['applied-drilldown-scope'];
export type DrilldownScopeProjection = components['schemas']['drilldown-scope-projection'];
export type DrilldownScopeProjectionItem = components['schemas']['drilldown-scope-projection-item'];
export type AuditLogConvertibleFilters = components['schemas']['audit-log-convertible-filters'];

export type AuditTimePreset = components['schemas']['AuditOverviewResponse']['time_preset'];
export type AuditBusinessCategory = components['schemas']['AuditBusinessCategory'];
export type AuditDrilldownScope = components['schemas']['AuditDrilldownScope'];
export type AuditRiskLevel = 'LOW' | 'MEDIUM' | 'HIGH' | 'CRITICAL';
export type AuditResult = 'SUCCESS' | 'FAILED' | 'DENIED' | 'ERROR';
export type AuditSource = 'REQUEST' | 'SECURITY_EVENT' | 'DOMAIN_EVENT';
export type AuditSortBy = 'created_at';
export type AuditSortOrder = 'asc' | 'desc';
export type AuditSorter = QuerySorter<AuditSortBy>;

export type AuditLogQuery = {
  page?: number;
  page_size?: number;
  preset?: AuditTimePreset;
  scope?: components['schemas']['AuditDrilldownScope'];
  keyword?: string;
  actor?: string;
  actor_user_id?: number;
  action?: string;
  action_prefix?: string;
  action_prefixes?: string[];
  action_keywords?: string[];
  source?: AuditSource;
  business_category?: AuditBusinessCategory;
  resource_type?: string;
  resource_types?: string[];
  resource_id?: string;
  resource_name?: string;
  session_id?: string;
  request_id?: string;
  result?: AuditResult;
  results?: AuditResult[];
  risk_level?: AuditRiskLevel;
  risk_levels?: AuditRiskLevel[];
  success?: boolean;
  request_path_prefixes?: string[];
  created_from?: string;
  created_to?: string;
  sort?: string[];
};

export type AuditOverviewQuery = {
  preset?: AuditTimePreset;
};
