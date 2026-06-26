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

/**
 * 解析通知导航对应的路由位置。
 *
 * @param navigation - 通知导航信息
 * @returns 匹配到的路由位置；对于不支持的类型返回 `null`
 */
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

      return buildAuditLogLocation(payload);
    }

    case NOTIFICATION_NAVIGATION_KIND.AUDIT_LOG: {
      return buildAuditLogLocation(payload);
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

/**
 * 构建审计日志的跳转位置。
 *
 * @param payload - 用于提取审计日志标识的负载数据
 * @returns 根据 `audit_log_id`、`request_id` 或空条件生成的位置
 */
function buildAuditLogLocation(payload: Record<string, unknown>) {
  const auditLogId = payloadText(payload, 'audit_log_id');
  if (auditLogId) {
    return buildAuditLogsLocation({ audit_log_id: auditLogId });
  }

  const requestId = payloadText(payload, 'request_id');
  if (requestId) {
    return buildAuditLogsLocation({ request_id: requestId });
  }

  return buildAuditLogsLocation({});
}
