import { ACCESS_LOG_ROUTE_PATH } from '@/modules/access-log/contract/paths';
import { AUDIT_ROUTE_PATH } from '@/modules/audit/contract/paths';
import { BUILD_ROUTE_PATH } from '@/modules/build/contract/paths';
import { MONITOR_ROUTE_PATH } from '@/modules/monitor/contract/paths';
import { NETWORK_ROUTE_PATH } from '@/modules/network/contract/paths';
import { SCHEDULED_TASK_ROUTE_PATH } from '@/modules/scheduled-task/contract/paths';
import type { MenuIconKey } from '@/shared/icons/menu-icon';

export const PRESENTATION_STATUS = {
  ERROR: 'error',
  WARNING: 'warning',
  UNKNOWN: 'unknown',
  INFO: 'info',
  HEALTHY: 'healthy',
} as const;

export type PresentationStatus = (typeof PRESENTATION_STATUS)[keyof typeof PRESENTATION_STATUS];

export const EVIDENCE_STATE = {
  CONFIRMED: 'confirmed',
  SOURCE_FAILED: 'source-failed',
  MISSING: 'missing',
  NOT_APPLICABLE: 'not-applicable',
} as const;

export type EvidenceState = (typeof EVIDENCE_STATE)[keyof typeof EVIDENCE_STATE];

export type PresentationRegion = 'attention' | 'health' | 'activity' | 'resources';

export interface WorkbenchAction {
  labelKey: string;
  route?: string;
  kind: 'navigate' | 'retry';
}

export interface PresentationItem {
  id: string;
  region: PresentationRegion;
  status: PresentationStatus;
  evidenceState: EvidenceState;
  titleKey: string;
  descriptionKey: string;
  occurredAt?: string;
  actionable?: boolean;
  action?: WorkbenchAction;
  sourceMetadata?: {
    loadStatus?: 'normal' | 'error';
    displayState?: 'normal' | 'warning' | 'critical' | 'hidden';
    priority?: 'normal' | 'info' | 'warning' | 'critical';
  };
}

export interface QuickAction {
  id: string;
  iconKey: MenuIconKey;
  titleKey: string;
  descriptionKey: string;
  route: string;
}

export interface WorkbenchScenario {
  generatedAt: string;
  operational: {
    enabledModules: number;
    failedTasks: number;
    highRiskEvents: number;
  };
  items: PresentationItem[];
  quickActions: QuickAction[];
}

export interface WorkbenchPresentation {
  generatedAt: string;
  operational: WorkbenchScenario['operational'] & {
    needsReview: number;
    statusCounts: Record<PresentationStatus, number>;
  };
  attention: PresentationItem[];
  health: PresentationItem[];
  activity: PresentationItem[];
  resources: PresentationItem[];
  quickActions: QuickAction[];
}

const STATUS_RANK: Record<PresentationStatus, number> = {
  error: 0,
  warning: 1,
  unknown: 2,
  info: 3,
  healthy: 4,
};

const ATTENTION_STATUSES = new Set<PresentationStatus>([
  PRESENTATION_STATUS.ERROR,
  PRESENTATION_STATUS.WARNING,
  PRESENTATION_STATUS.UNKNOWN,
]);

export function sortPresentationItems<T extends PresentationItem>(items: readonly T[]): T[] {
  return [...items].sort((left, right) => {
    const statusDifference = STATUS_RANK[left.status] - STATUS_RANK[right.status];
    if (statusDifference !== 0) {
      return statusDifference;
    }

    const actionDifference = Number(Boolean(right.actionable)) - Number(Boolean(left.actionable));
    if (actionDifference !== 0) {
      return actionDifference;
    }

    const timeDifference = (right.occurredAt ?? '').localeCompare(left.occurredAt ?? '');
    return timeDifference || left.id.localeCompare(right.id);
  });
}

export function presentationStatusFromAlertLevel(level: 'error' | 'warning' | 'info'): PresentationStatus {
  return level;
}

export function presentationStatusFromHealthStatus(
  status: 'healthy' | 'degraded' | 'unknown' | 'disabled',
): PresentationStatus {
  const mapping = {
    healthy: PRESENTATION_STATUS.HEALTHY,
    degraded: PRESENTATION_STATUS.WARNING,
    unknown: PRESENTATION_STATUS.UNKNOWN,
    disabled: PRESENTATION_STATUS.INFO,
  } as const satisfies Record<string, PresentationStatus>;

  return mapping[status];
}

export function presentationStatusFromRequestAttentionKind(
  kind: 'server-error' | 'client-error' | 'slow-request',
): PresentationStatus {
  return kind === 'server-error' ? PRESENTATION_STATUS.ERROR : PRESENTATION_STATUS.WARNING;
}

export function projectWorkbenchScenario(scenario: WorkbenchScenario): WorkbenchPresentation {
  scenario.items.forEach(assertPresentationEvidence);
  const sortedItems = sortPresentationItems(scenario.items);
  const attention = sortedItems.filter((item) => item.region === 'attention' && ATTENTION_STATUSES.has(item.status));
  const statusCounts = Object.fromEntries(
    Object.values(PRESENTATION_STATUS).map((status) => [
      status,
      scenario.items.filter((item) => item.status === status).length,
    ]),
  ) as Record<PresentationStatus, number>;

  return {
    generatedAt: scenario.generatedAt,
    operational: {
      ...scenario.operational,
      needsReview: attention.length,
      statusCounts,
    },
    attention,
    health: sortedItems.filter((item) => item.region === 'health'),
    activity: sortedItems.filter((item) => item.region === 'activity'),
    resources: sortedItems.filter((item) => item.region === 'resources'),
    quickActions: scenario.quickActions,
  };
}

function assertPresentationEvidence(item: PresentationItem) {
  if (
    (item.status === PRESENTATION_STATUS.ERROR || item.status === PRESENTATION_STATUS.HEALTHY) &&
    item.evidenceState !== EVIDENCE_STATE.CONFIRMED
  ) {
    throw new Error(`Presentation item ${item.id} requires confirmed evidence for status ${item.status}`);
  }

  if (
    item.status === PRESENTATION_STATUS.UNKNOWN &&
    item.evidenceState !== EVIDENCE_STATE.MISSING &&
    item.evidenceState !== EVIDENCE_STATE.SOURCE_FAILED
  ) {
    throw new Error(`Presentation item ${item.id} requires missing evidence for unknown status`);
  }
}

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
      titleKey: 'dashboard.workbench.attention.accessLog.title',
      descriptionKey: 'dashboard.workbench.attention.accessLog.description',
      actionable: true,
      action: { labelKey: 'dashboard.workbench.attention.accessLog.action', kind: 'retry' },
      sourceMetadata: { loadStatus: 'error', displayState: 'critical', priority: 'critical' },
    },
    {
      id: 'outbound-network',
      region: 'attention',
      status: PRESENTATION_STATUS.UNKNOWN,
      evidenceState: EVIDENCE_STATE.MISSING,
      titleKey: 'dashboard.workbench.attention.outbound.title',
      descriptionKey: 'dashboard.workbench.attention.outbound.description',
      actionable: true,
      action: {
        labelKey: 'dashboard.workbench.attention.outbound.action',
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
      titleKey: 'dashboard.workbench.health.postgresql.title',
      descriptionKey: 'dashboard.workbench.health.postgresql.description',
    },
    {
      id: 'health-redis',
      region: 'health',
      status: PRESENTATION_STATUS.HEALTHY,
      evidenceState: EVIDENCE_STATE.CONFIRMED,
      titleKey: 'dashboard.workbench.health.redis.title',
      descriptionKey: 'dashboard.workbench.health.redis.description',
    },
    {
      id: 'activity-build',
      region: 'activity',
      status: PRESENTATION_STATUS.INFO,
      evidenceState: EVIDENCE_STATE.CONFIRMED,
      titleKey: 'dashboard.workbench.activity.build.title',
      descriptionKey: 'dashboard.workbench.activity.build.description',
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
      titleKey: 'dashboard.workbench.activity.scheduledTask.title',
      descriptionKey: 'dashboard.workbench.activity.scheduledTask.description',
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
      titleKey: 'dashboard.workbench.quickActions.createBuild.title',
      descriptionKey: 'dashboard.workbench.quickActions.createBuild.description',
      route: BUILD_ROUTE_PATH.CREATE,
    },
    {
      id: 'observability',
      iconKey: 'observability-overview',
      titleKey: 'dashboard.workbench.quickActions.observability.title',
      descriptionKey: 'dashboard.workbench.quickActions.observability.description',
      route: MONITOR_ROUTE_PATH.OBSERVABILITY_OVERVIEW,
    },
    {
      id: 'audit-logs',
      iconKey: 'audit-trail',
      titleKey: 'dashboard.workbench.quickActions.audit.title',
      descriptionKey: 'dashboard.workbench.quickActions.audit.description',
      route: AUDIT_ROUTE_PATH.LOGS,
    },
    {
      id: 'artifacts',
      iconKey: 'image-artifact',
      titleKey: 'dashboard.workbench.quickActions.artifacts.title',
      descriptionKey: 'dashboard.workbench.quickActions.artifacts.description',
      route: BUILD_ROUTE_PATH.ARTIFACTS,
    },
    {
      id: 'scheduled-tasks',
      iconKey: 'scheduled-automation',
      titleKey: 'dashboard.workbench.quickActions.scheduledTasks.title',
      descriptionKey: 'dashboard.workbench.quickActions.scheduledTasks.description',
      route: SCHEDULED_TASK_ROUTE_PATH.LIST,
    },
    {
      id: 'access-logs',
      iconKey: 'access-log',
      titleKey: 'dashboard.workbench.quickActions.accessLogs.title',
      descriptionKey: 'dashboard.workbench.quickActions.accessLogs.description',
      route: ACCESS_LOG_ROUTE_PATH.LIST,
    },
  ],
} as const satisfies WorkbenchScenario;

export const DASHBOARD_PREVIEW_PRESENTATION = projectWorkbenchScenario(DASHBOARD_PREVIEW_SCENARIO);
