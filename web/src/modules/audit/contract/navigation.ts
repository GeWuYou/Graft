import type { LocationQuery, RouteLocationAsPathGeneric } from 'vue-router';

import {
  buildMonitorLocationFromOrigin,
  buildMonitorOriginQuery,
  type MonitorOriginContext,
  normalizeMonitorOriginContext,
  parseMonitorOriginQuery,
} from '@/modules/monitor/contract/navigation';

import type { AuditLogListItem } from '../types/audit';
import { buildAuditIncidentLocation, buildAuditLogsLocation, buildAuditRequestLocation } from './deep-link';

export type AuditNavigationContext = {
  monitorOrigin: MonitorOriginContext | null;
};

type RouteLocationWithQuery = RouteLocationAsPathGeneric;

export function resolveAuditNavigationContext(query: LocationQuery | Record<string, unknown>): AuditNavigationContext {
  return {
    monitorOrigin: parseMonitorOriginQuery(query as Record<string, unknown>),
  };
}

export function withMonitorOrigin(
  location: RouteLocationWithQuery,
  monitorOrigin?: MonitorOriginContext | null,
): RouteLocationWithQuery {
  if (!monitorOrigin) {
    return location;
  }

  const normalized = normalizeMonitorOriginContext(monitorOrigin);
  const query = location.query ? { ...location.query } : {};

  return {
    ...location,
    query: {
      ...query,
      ...buildMonitorOriginQuery(normalized),
    },
  };
}

export function buildAuditIncidentLocationWithOrigin(
  eventId: number | string,
  monitorOrigin?: MonitorOriginContext | null,
): RouteLocationWithQuery {
  return withMonitorOrigin(buildAuditIncidentLocation(eventId) as RouteLocationWithQuery, monitorOrigin);
}

export function buildAuditRequestLocationWithOrigin(
  requestId: string,
  monitorOrigin?: MonitorOriginContext | null,
): RouteLocationWithQuery {
  return withMonitorOrigin(buildAuditRequestLocation(requestId) as RouteLocationWithQuery, monitorOrigin);
}

export function buildAuditLogsLocationWithOrigin(
  query: Parameters<typeof buildAuditLogsLocation>[0],
  monitorOrigin?: MonitorOriginContext | null,
): RouteLocationWithQuery {
  return withMonitorOrigin(buildAuditLogsLocation(query) as RouteLocationWithQuery, monitorOrigin);
}

export function buildAuditRelatedActorLocation(
  actor: string,
  actorUserId?: number | string | null,
  monitorOrigin?: MonitorOriginContext | null,
): RouteLocationWithQuery {
  return buildAuditLogsLocationWithOrigin(
    {
      actor,
      actorUserId: actorUserId === null || actorUserId === undefined ? '' : String(actorUserId),
    },
    monitorOrigin,
  );
}

export function buildAuditRelatedResourceLocation(
  resourceType: string,
  resourceId: string,
  resourceName?: string,
  monitorOrigin?: MonitorOriginContext | null,
): RouteLocationWithQuery {
  return buildAuditLogsLocationWithOrigin({ resourceType, resourceId, resourceName }, monitorOrigin);
}

export function buildAuditRelatedRecordLocation(
  row: AuditLogListItem,
  monitorOrigin?: MonitorOriginContext | null,
): RouteLocationWithQuery {
  if (row.request_id) {
    return buildAuditRequestLocationWithOrigin(row.request_id, monitorOrigin);
  }

  if (row.resource_type && row.resource_id) {
    return buildAuditRelatedResourceLocation(row.resource_type, row.resource_id, row.resource_name, monitorOrigin);
  }

  if (row.actor_display_name || row.actor_username) {
    return buildAuditRelatedActorLocation(
      row.actor_username || row.actor_display_name || '',
      row.actor_user_id,
      monitorOrigin,
    );
  }

  return buildAuditLogsLocationWithOrigin({}, monitorOrigin);
}

export function buildMonitorReturnLocation(
  query: LocationQuery | Record<string, unknown>,
): RouteLocationWithQuery | null {
  const monitorOrigin = resolveAuditNavigationContext(query).monitorOrigin;
  return monitorOrigin ? (buildMonitorLocationFromOrigin(monitorOrigin) as RouteLocationWithQuery) : null;
}
