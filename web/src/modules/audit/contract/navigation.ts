import type { LocationQuery, RouteLocationAsPathGeneric } from 'vue-router';

import { buildAccessLogRequestLocation } from '@/modules/access-log/contract/deep-link';
import {
  buildMonitorLocationFromOrigin,
  buildMonitorOriginQuery,
  type MonitorOriginContext,
  normalizeMonitorOriginContext,
  parseMonitorOriginQuery,
} from '@/modules/monitor/contract/navigation';

import type { AuditLogListItem } from '../types/audit';
import { buildAuditLogsLocation, buildAuditRequestLocation } from './deep-link';

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

function buildAuditRequestLocationWithOrigin(
  requestId: string,
  monitorOrigin?: MonitorOriginContext | null,
): RouteLocationWithQuery {
  return withMonitorOrigin(buildAuditRequestLocation(requestId) as RouteLocationWithQuery, monitorOrigin);
}

export function buildAccessLogRequestLocationWithOrigin(
  requestId: string,
  monitorOrigin?: MonitorOriginContext | null,
): RouteLocationWithQuery {
  return withMonitorOrigin(buildAccessLogRequestLocation(requestId) as RouteLocationWithQuery, monitorOrigin);
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
  void actorUserId;
  return buildAuditLogsLocationWithOrigin(
    {
      actor,
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
  return buildAuditLogsLocationWithOrigin(
    { resource_type: resourceType, resource_id: resourceId, resource_name: resourceName },
    monitorOrigin,
  );
}

/**
 * 根据审计日志记录生成对应的审计导航位置。
 *
 * 当记录包含多个可用字段时，按优先级选择 `audit_log_id`、`request_id`、资源信息或操作者信息来构造目标位置，并附加监控来源上下文。
 *
 * @param row - 审计日志列表项
 * @param monitorOrigin - 要附加到目标位置的监控来源上下文
 * @returns 生成的审计相关路由位置
 */
export function buildAuditRelatedRecordLocation(
  row: AuditLogListItem,
  monitorOrigin?: MonitorOriginContext | null,
): RouteLocationWithQuery {
  if (row.id) {
    return buildAuditLogsLocationWithOrigin(
      {
        audit_log_id: String(row.id),
      },
      monitorOrigin,
    );
  }
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
