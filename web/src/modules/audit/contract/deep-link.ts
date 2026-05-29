import type { LocationQuery, LocationQueryValue } from 'vue-router';

import { AUDIT_ROUTE_PATH } from './paths';

export type AuditLogsRouteQuery = Partial<{
  preset: string;
  keyword: string;
  actor: string;
  action: string;
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
