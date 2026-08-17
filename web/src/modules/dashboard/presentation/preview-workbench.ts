import { ACCESS_LOG_ROUTE_PATH } from '@/modules/access-log/contract/paths';
import { AUDIT_ROUTE_PATH } from '@/modules/audit/contract/paths';
import { BUILD_ROUTE_PATH } from '@/modules/build/contract/paths';
import { MONITOR_ROUTE_PATH } from '@/modules/monitor/contract/paths';
import { NETWORK_ROUTE_PATH } from '@/modules/network/contract/paths';
import { SCHEDULED_TASK_ROUTE_PATH } from '@/modules/scheduled-task/contract/paths';

import { EVIDENCE_STATE, PRESENTATION_STATUS, projectWorkbenchScenario, type WorkbenchScenario } from './workbench';

/** 预览场景只服务 development-only 设计验收，删除预览路由时应整体移除。 */
export const DASHBOARD_PREVIEW_SCENARIO = {
  generatedAt: '2026-08-17T03:20:00.000Z',
  operational: {
    enabledModules: 22,
    failedTasks: 0,
    highRiskEvents: 0,
  },
  items: [
    {
      id: 'access-log-source',
      region: 'attention',
      status: PRESENTATION_STATUS.WARNING,
      evidenceState: EVIDENCE_STATE.SOURCE_FAILED,
      titleKey: 'dashboard.previewWorkbench.attention.accessLog.title',
      descriptionKey: 'dashboard.previewWorkbench.attention.accessLog.description',
      actionable: true,
      action: { labelKey: 'dashboard.previewWorkbench.attention.accessLog.action', kind: 'retry' },
      sourceMetadata: { loadStatus: 'error', displayState: 'critical', priority: 'critical' },
    },
    {
      id: 'outbound-network',
      region: 'attention',
      status: PRESENTATION_STATUS.UNKNOWN,
      evidenceState: EVIDENCE_STATE.MISSING,
      titleKey: 'dashboard.previewWorkbench.attention.outbound.title',
      descriptionKey: 'dashboard.previewWorkbench.attention.outbound.description',
      actionable: true,
      action: {
        labelKey: 'dashboard.previewWorkbench.attention.outbound.action',
        kind: 'navigate',
        route: NETWORK_ROUTE_PATH.OUTBOUND,
      },
      sourceMetadata: { loadStatus: 'normal', displayState: 'critical', priority: 'critical' },
    },
    {
      id: 'health-postgresql',
      region: 'health',
      status: PRESENTATION_STATUS.HEALTHY,
      evidenceState: EVIDENCE_STATE.CONFIRMED,
      titleKey: 'dashboard.previewWorkbench.health.postgresql.title',
      descriptionKey: 'dashboard.previewWorkbench.health.postgresql.description',
    },
    {
      id: 'health-redis',
      region: 'health',
      status: PRESENTATION_STATUS.HEALTHY,
      evidenceState: EVIDENCE_STATE.CONFIRMED,
      titleKey: 'dashboard.previewWorkbench.health.redis.title',
      descriptionKey: 'dashboard.previewWorkbench.health.redis.description',
    },
    {
      id: 'activity-build',
      region: 'activity',
      status: PRESENTATION_STATUS.INFO,
      evidenceState: EVIDENCE_STATE.CONFIRMED,
      titleKey: 'dashboard.previewWorkbench.activity.build.title',
      descriptionKey: 'dashboard.previewWorkbench.activity.build.description',
      occurredAt: '2026-08-17T03:12:00.000Z',
      actionable: true,
      action: {
        labelKey: 'dashboard.workbench.activity.view',
        kind: 'navigate',
        route: BUILD_ROUTE_PATH.JOBS,
      },
    },
    {
      id: 'activity-scheduled-task',
      region: 'activity',
      status: PRESENTATION_STATUS.INFO,
      evidenceState: EVIDENCE_STATE.CONFIRMED,
      titleKey: 'dashboard.previewWorkbench.activity.scheduledTask.title',
      descriptionKey: 'dashboard.previewWorkbench.activity.scheduledTask.description',
      occurredAt: '2026-08-17T02:54:00.000Z',
      actionable: true,
      action: {
        labelKey: 'dashboard.workbench.activity.view',
        kind: 'navigate',
        route: SCHEDULED_TASK_ROUTE_PATH.LIST,
      },
    },
    {
      id: 'resources-no-sample',
      region: 'resources',
      status: PRESENTATION_STATUS.INFO,
      evidenceState: EVIDENCE_STATE.MISSING,
      titleKey: 'dashboard.workbench.resources.noSample.title',
      descriptionKey: 'dashboard.workbench.resources.noSample.description',
      actionable: true,
      action: {
        labelKey: 'dashboard.workbench.resources.noSample.action',
        kind: 'navigate',
        route: MONITOR_ROUTE_PATH.OBSERVABILITY_OVERVIEW,
      },
    },
  ],
  quickActions: [
    {
      id: 'create-build',
      iconKey: 'build',
      kind: 'action',
      titleKey: 'dashboard.previewWorkbench.quickActions.createBuild.title',
      descriptionKey: 'dashboard.previewWorkbench.quickActions.createBuild.description',
      route: BUILD_ROUTE_PATH.CREATE,
      showOnHome: true,
    },
    {
      id: 'observability',
      iconKey: 'observability-overview',
      kind: 'navigation',
      titleKey: 'dashboard.previewWorkbench.quickActions.observability.title',
      descriptionKey: 'dashboard.previewWorkbench.quickActions.observability.description',
      route: MONITOR_ROUTE_PATH.OBSERVABILITY_OVERVIEW,
      showOnHome: true,
    },
    {
      id: 'audit-logs',
      iconKey: 'audit-trail',
      kind: 'navigation',
      titleKey: 'dashboard.previewWorkbench.quickActions.audit.title',
      descriptionKey: 'dashboard.previewWorkbench.quickActions.audit.description',
      route: AUDIT_ROUTE_PATH.LOGS,
      showOnHome: true,
    },
    {
      id: 'artifacts',
      iconKey: 'image-artifact',
      kind: 'navigation',
      titleKey: 'dashboard.previewWorkbench.quickActions.artifacts.title',
      descriptionKey: 'dashboard.previewWorkbench.quickActions.artifacts.description',
      route: BUILD_ROUTE_PATH.ARTIFACTS,
      showOnHome: true,
    },
    {
      id: 'scheduled-tasks',
      iconKey: 'scheduled-automation',
      kind: 'navigation',
      titleKey: 'dashboard.previewWorkbench.quickActions.scheduledTasks.title',
      descriptionKey: 'dashboard.previewWorkbench.quickActions.scheduledTasks.description',
      route: SCHEDULED_TASK_ROUTE_PATH.LIST,
      showOnHome: false,
    },
    {
      id: 'access-logs',
      iconKey: 'access-log',
      kind: 'navigation',
      titleKey: 'dashboard.previewWorkbench.quickActions.accessLogs.title',
      descriptionKey: 'dashboard.previewWorkbench.quickActions.accessLogs.description',
      route: ACCESS_LOG_ROUTE_PATH.LIST,
      showOnHome: false,
    },
  ],
} as const satisfies WorkbenchScenario;

export const DASHBOARD_PREVIEW_PRESENTATION = projectWorkbenchScenario(DASHBOARD_PREVIEW_SCENARIO);
