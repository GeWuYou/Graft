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
  action: string;
  actionPrefix: string;
  source: string;
  createdFrom: string;
  createdTo: string;
  resourceType: string;
  resourceName: string;
  resourceId: string;
  result: string;
  riskLevel: string;
  session: string;
  requestId: string;
  traceId: string;
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
    action: trimQueryValue(firstQueryValue(query.action)),
    actionPrefix: trimQueryValue(firstQueryValue(query.actionPrefix)),
    source: trimQueryValue(firstQueryValue(query.source)),
    createdFrom: trimQueryValue(firstQueryValue(query.createdFrom)),
    createdTo: trimQueryValue(firstQueryValue(query.createdTo)),
    resourceType: trimQueryValue(firstQueryValue(query.resourceType)),
    resourceName: trimQueryValue(firstQueryValue(query.resourceName)),
    resourceId: trimQueryValue(firstQueryValue(query.resourceId)),
    result: trimQueryValue(firstQueryValue(query.result)),
    riskLevel: trimQueryValue(firstQueryValue(query.riskLevel)),
    session: trimQueryValue(firstQueryValue(query.session)),
    requestId: trimQueryValue(firstQueryValue(query.requestId)),
    traceId: trimQueryValue(firstQueryValue(query.traceId)),
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
    resourceType,
    resourceName,
    resourceId,
  });
}

export function buildAuditRequestLocation(requestId: string) {
  return buildAuditLogsLocation({
    requestId,
  });
}

export function buildAuditIncidentLocation(eventId: number | string) {
  return {
    path: AUDIT_ROUTE_PATH.INCIDENT_DETAIL.replace(':eventId', String(eventId)),
  };
}

function buildAuditEvidenceLocation(context: AuditEvidenceContext) {
  return buildAuditLogsLocation({
    action: context.action,
    actionPrefix: context.action_prefix,
    source: context.source,
    resourceType: context.resource_type,
    resourceId: context.resource_id,
    resourceName: context.resource_name,
    requestId: context.request_id,
    result: context.result,
    riskLevel: context.risk_level,
    createdFrom: context.created_from,
    createdTo: context.created_to,
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
