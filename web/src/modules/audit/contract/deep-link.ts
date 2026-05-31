import type { LocationQuery, LocationQueryValue } from 'vue-router';

import type { components } from '@/contracts/openapi/generated/schema';
import type { MonitorOriginContext } from '@/modules/monitor/contract/navigation';

import { withMonitorOrigin } from './navigation';
import { AUDIT_ROUTE_PATH } from './paths';

type AuditEvidenceContext = components['schemas']['AuditEvidenceContext'];

export type AuditLogsRouteQuery = Partial<{
  preset: string;
  keyword: string;
  actor: string;
  user_id: string;
  username: string;
  success: string;
  action: string;
  action_prefix: string;
  action_prefixes: string;
  action_keywords: string;
  source: string;
  created_from: string;
  created_to: string;
  occurred_from: string;
  occurred_to: string;
  resource_type: string;
  resource_types: string;
  resource_name: string;
  resource_id: string;
  result: string;
  results: string;
  risk_level: string;
  risk_levels: string;
  session: string;
  request_id: string;
  request_path_prefixes: string;
  sort_by: string;
  sort_order: string;
}>;

function trimQueryValue(value: unknown) {
  return typeof value === 'string' ? value.trim() : '';
}

function firstQueryValue(value: LocationQueryValue | LocationQueryValue[] | undefined) {
  return Array.isArray(value) ? value[0] : value;
}

export function parseAuditLogsRouteQuery(query: LocationQuery | AuditLogsRouteQuery): AuditLogsRouteQuery {
  return {
    preset: trimQueryValue(firstQueryValue(query.preset)),
    keyword: trimQueryValue(firstQueryValue(query.keyword)),
    actor: trimQueryValue(firstQueryValue(query.actor)),
    user_id: trimQueryValue(firstQueryValue(query.user_id)),
    username: trimQueryValue(firstQueryValue(query.username)),
    success: trimQueryValue(firstQueryValue(query.success)),
    action: trimQueryValue(firstQueryValue(query.action)),
    action_prefix: trimQueryValue(firstQueryValue(query.action_prefix)),
    action_prefixes: trimQueryValue(firstQueryValue(query.action_prefixes)),
    action_keywords: trimQueryValue(firstQueryValue(query.action_keywords)),
    source: trimQueryValue(firstQueryValue(query.source)),
    created_from: trimQueryValue(firstQueryValue(query.created_from ?? query.occurred_from)),
    created_to: trimQueryValue(firstQueryValue(query.created_to ?? query.occurred_to)),
    resource_type: trimQueryValue(firstQueryValue(query.resource_type)),
    resource_types: trimQueryValue(firstQueryValue(query.resource_types)),
    resource_name: trimQueryValue(firstQueryValue(query.resource_name)),
    resource_id: trimQueryValue(firstQueryValue(query.resource_id)),
    result: trimQueryValue(firstQueryValue(query.result)),
    results: trimQueryValue(firstQueryValue(query.results)),
    risk_level: trimQueryValue(firstQueryValue(query.risk_level)),
    risk_levels: trimQueryValue(firstQueryValue(query.risk_levels)),
    session: trimQueryValue(firstQueryValue(query.session)),
    request_id: trimQueryValue(firstQueryValue(query.request_id)),
    request_path_prefixes: trimQueryValue(firstQueryValue(query.request_path_prefixes)),
    sort_by: trimQueryValue(firstQueryValue(query.sort_by)),
    sort_order: trimQueryValue(firstQueryValue(query.sort_order)),
  };
}

function normalizeAuditLogsRouteQuery(query: AuditLogsRouteQuery) {
  return Object.fromEntries(
    Object.entries(parseAuditLogsRouteQuery(query))
      .map(([key, value]) => [key, trimQueryValue(value)])
      .filter(([, value]) => value !== ''),
  ) as Record<string, string>;
}

export function buildAuditLogsLocation(query: AuditLogsRouteQuery) {
  return {
    path: AUDIT_ROUTE_PATH.LOGS,
    query: normalizeAuditLogsRouteQuery(query),
  };
}

export function buildAuditResourceLocation(resourceType: string, resourceId: string, resourceName?: string) {
  return buildAuditLogsLocation({
    resource_name: resourceName,
    resource_type: resourceType,
    resource_id: resourceId,
  });
}

export function buildAuditRequestLocation(requestId: string) {
  return buildAuditLogsLocation({
    request_id: requestId,
  });
}

function buildAuditIncidentLocation(eventId: number | string) {
  return {
    path: AUDIT_ROUTE_PATH.INCIDENT_DETAIL.replace(':event_id', String(eventId)),
  };
}

function buildAuditEvidenceLocation(context: AuditEvidenceContext) {
  return buildAuditLogsLocation({
    action: context.action,
    action_prefix: context.action_prefix,
    source: context.source,
    resource_type: context.resource_type,
    resource_id: context.resource_id,
    resource_name: context.resource_name,
    request_id: context.request_id,
    result: context.result,
    risk_level: context.risk_level,
    created_from: context.created_from,
    created_to: context.created_to,
  });
}

type EvidenceLink = components['schemas']['EvidenceLink'];

export function buildAuditEvidenceTargetLocation(link: EvidenceLink, monitorOrigin?: MonitorOriginContext | null) {
  if (link.link_state !== 'available') {
    return null;
  }

  if (link.target_kind === 'audit_incident' && link.incident_seed?.event_id) {
    return withMonitorOrigin(buildAuditIncidentLocation(link.incident_seed.event_id), monitorOrigin);
  }

  if (link.audit_context) {
    return withMonitorOrigin(buildAuditEvidenceLocation(link.audit_context), monitorOrigin);
  }

  return null;
}
