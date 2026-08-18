import type {
  ContainerDashboardAnomalyItem,
  ContainerDashboardHotspotItem,
  ContainerDashboardOverview,
} from '@/modules/container/contract/dashboard-summary';

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
/** 区分快捷入口激活与上下文 drill-down，只有前者可以影响浏览器本地的入口排序。 */
export type WorkbenchNavigationSource = 'contextual-action' | 'quick-entry';

export type WorkbenchAction =
  | { kind: 'navigate'; labelKey: string; labelFallback?: string; route: string }
  | { kind: 'retry'; labelKey: string; labelFallback?: string; route?: never };

export interface PresentationItem {
  id: string;
  region: PresentationRegion;
  status: PresentationStatus;
  evidenceState: EvidenceState;
  titleKey: string;
  titleFallback?: string;
  titleParams?: Record<string, string | number>;
  descriptionKey: string;
  descriptionFallback?: string;
  descriptionParams?: Record<string, string | number>;
  occurredAt?: string;
  actionable?: boolean;
  action?: WorkbenchAction;
  sourceWidgetId?: string;
  sourceMetadata?: {
    loadStatus?: 'normal' | 'error';
    displayState?: 'normal' | 'warning' | 'critical' | 'hidden';
    priority?: 'normal' | 'info' | 'warning' | 'critical';
  };
}

export interface QuickAction {
  id: string;
  iconKey?: string;
  kind: 'action' | 'navigation';
  titleKey: string;
  titleFallback?: string;
  descriptionKey: string;
  descriptionFallback?: string;
  route: string;
  showOnHome: boolean;
}

export type WorkbenchMetricTone = 'normal' | 'success' | 'warning' | 'error' | 'info';

export interface WorkbenchMetric {
  key: string;
  labelKey: string;
  labelFallback?: string;
  value: string;
  unitKey?: string;
  unitFallback?: string;
  descriptionKey?: string;
  descriptionFallback?: string;
  tone: WorkbenchMetricTone;
  route?: string;
}

export interface WorkbenchMetricGroup {
  id: string;
  titleKey: string;
  titleFallback?: string;
  descriptionKey?: string;
  descriptionFallback?: string;
  action?: WorkbenchAction;
  metrics: WorkbenchMetric[];
  sourceWidgetId: string;
}

export interface WorkbenchContextLink {
  key: string;
  labelKey: string;
  labelFallback?: string;
  descriptionKey?: string;
  descriptionFallback?: string;
  route: string;
  iconKey?: string;
  badgeKey?: string;
  badgeFallback?: string;
  disabled: boolean;
}

export interface WorkbenchContextLinkGroup {
  id: string;
  titleKey: string;
  titleFallback?: string;
  descriptionKey?: string;
  descriptionFallback?: string;
  action?: WorkbenchAction;
  links: WorkbenchContextLink[];
  sourceWidgetId: string;
}

export interface WorkbenchModuleCoverage {
  registeredModules: number;
  enabledModules: number;
  degradedModules: number;
  normalContributionSources: number;
  failedContributionSources: number;
}

export type WorkbenchResourceState = 'hidden' | 'loading' | 'failed' | 'no-sample' | 'loaded';

export type WorkbenchResourceOverview = Omit<ContainerDashboardOverview, 'collectedAt'> & {
  collectedAt: string;
};

export type WorkbenchResourceHotspot = ContainerDashboardHotspotItem;

export type WorkbenchResourceAnomaly = ContainerDashboardAnomalyItem;

export interface WorkbenchResourceSummary {
  state: WorkbenchResourceState;
  route?: string;
  overview?: WorkbenchResourceOverview;
  topCpu: WorkbenchResourceHotspot[];
  topMemory: WorkbenchResourceHotspot[];
  anomalies: WorkbenchResourceAnomaly[];
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
  moduleCoverage?: WorkbenchModuleCoverage;
  metricGroups?: WorkbenchMetricGroup[];
  contextLinkGroups?: WorkbenchContextLinkGroup[];
  resourceSummary?: WorkbenchResourceSummary;
}

export interface WorkbenchPresentation {
  generatedAt: string;
  operational: WorkbenchScenario['operational'] & {
    needsReview: number;
    attentionStatusCounts: Record<PresentationStatus, number>;
  };
  attention: PresentationItem[];
  health: PresentationItem[];
  activity: PresentationItem[];
  resources: PresentationItem[];
  moduleCoverage: WorkbenchModuleCoverage;
  metricGroups: WorkbenchMetricGroup[];
  contextLinkGroups: WorkbenchContextLinkGroup[];
  resourceSummary: WorkbenchResourceSummary;
  homeQuickActions: QuickAction[];
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

    const timeDifference = occurredAtTimestamp(right.occurredAt) - occurredAtTimestamp(left.occurredAt);
    return timeDifference || left.id.localeCompare(right.id);
  });
}

function occurredAtTimestamp(value?: string) {
  if (!value) {
    return 0;
  }

  const timestamp = Date.parse(value);
  return Number.isFinite(timestamp) ? timestamp : 0;
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
  const attentionStatusCounts = Object.fromEntries(
    Object.values(PRESENTATION_STATUS).map((status) => [
      status,
      attention.filter((item) => item.status === status).length,
    ]),
  ) as Record<PresentationStatus, number>;

  return {
    generatedAt: scenario.generatedAt,
    operational: {
      ...scenario.operational,
      needsReview: attention.length,
      attentionStatusCounts,
    },
    attention,
    health: sortedItems.filter((item) => item.region === 'health'),
    activity: sortedItems.filter((item) => item.region === 'activity'),
    resources: sortedItems.filter((item) => item.region === 'resources'),
    moduleCoverage: scenario.moduleCoverage ?? {
      registeredModules: scenario.operational.enabledModules,
      enabledModules: scenario.operational.enabledModules,
      degradedModules: 0,
      normalContributionSources: 0,
      failedContributionSources: 0,
    },
    metricGroups: scenario.metricGroups ?? [],
    contextLinkGroups: scenario.contextLinkGroups ?? [],
    resourceSummary: scenario.resourceSummary ?? {
      state: 'hidden',
      topCpu: [],
      topMemory: [],
      anomalies: [],
    },
    homeQuickActions: scenario.quickActions.filter((action) => action.showOnHome),
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
