import type { RouteLocationRaw } from 'vue-router';

import type { components } from '@/contracts/openapi/generated/schema';
import { buildAuditLogsLocation } from '@/modules/audit/contract/deep-link';
import { AUDIT_ROUTE_PATH } from '@/modules/audit/contract/paths';
import { SCHEDULED_TASK_ROUTE_PATH } from '@/modules/scheduled-task/contract/paths';

export type NotificationNavigationKind = components['schemas']['notification-navigation-kind'];
export type NotificationNavigation = components['schemas']['notification-navigation'];

export const NOTIFICATION_NAVIGATION_KIND = {
  AUDIT_INCIDENT: 'AUDIT_INCIDENT',
  AUDIT_LOG: 'AUDIT_LOG',
  SCHEDULER_RUN: 'SCHEDULER_RUN',
  SYSTEM_CONFIG_ITEM: 'SYSTEM_CONFIG_ITEM',
  MODULE_RUNTIME_ITEM: 'MODULE_RUNTIME_ITEM',
} as const satisfies Record<string, NotificationNavigationKind>;

function payloadText(payload: Record<string, unknown>, key: string) {
  const value = payload[key];
  if (typeof value === 'number' && Number.isFinite(value)) {
    return String(value);
  }
  return typeof value === 'string' ? value.trim() : '';
}

export function resolveNotificationNavigationLocation(navigation: NotificationNavigation): RouteLocationRaw | null {
  const payload = navigation.payload ?? {};

  switch (navigation.kind) {
    case NOTIFICATION_NAVIGATION_KIND.AUDIT_INCIDENT: {
      const incidentId = payloadText(payload, 'incident_id') || payloadText(payload, 'event_id');
      if (incidentId) {
        return {
          path: AUDIT_ROUTE_PATH.INCIDENT_DETAIL.replace(':event_id', encodeURIComponent(incidentId)),
        };
      }

      const auditLogId = payloadText(payload, 'audit_log_id');
      return auditLogId ? buildAuditLogsLocation({ keyword: auditLogId }) : buildAuditLogsLocation({});
    }

    case NOTIFICATION_NAVIGATION_KIND.AUDIT_LOG: {
      const requestId = payloadText(payload, 'request_id');
      if (requestId) {
        return buildAuditLogsLocation({ request_id: requestId });
      }

      const auditLogId = payloadText(payload, 'audit_log_id');
      return auditLogId ? buildAuditLogsLocation({ keyword: auditLogId }) : buildAuditLogsLocation({});
    }

    case NOTIFICATION_NAVIGATION_KIND.SCHEDULER_RUN: {
      const taskKey = payloadText(payload, 'task_id') || payloadText(payload, 'task_key');
      const runId = payloadText(payload, 'run_id');
      return {
        path: SCHEDULED_TASK_ROUTE_PATH.LIST,
        query: Object.fromEntries(
          [
            ['task_key', taskKey],
            ['run_id', runId],
          ].filter(([, value]) => Boolean(value)),
        ),
      };
    }

    case NOTIFICATION_NAVIGATION_KIND.SYSTEM_CONFIG_ITEM:
    case NOTIFICATION_NAVIGATION_KIND.MODULE_RUNTIME_ITEM:
      return null;

    default:
      return null;
  }
}
