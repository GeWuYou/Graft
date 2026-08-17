import type { ContainerDashboardSummary } from '@/modules/container/contract/dashboard-summary';

import {
  asAlertListPayload,
  asHealthPayload,
  asLinkListPayload,
  asStatGroupPayload,
  asTimelinePayload,
} from '../components/widgets/payload';
import type { DashboardQuickActionLink } from '../contract/quick-action-links';
import type { DashboardQuickActionConfig } from '../contract/quick-actions';
import type {
  DashboardLinkListPayload,
  DashboardStatGroupPayload,
  DashboardSummaryResponse,
  DashboardWidget,
} from '../types/dashboard';
import {
  EVIDENCE_STATE,
  PRESENTATION_STATUS,
  type PresentationItem,
  presentationStatusFromAlertLevel,
  presentationStatusFromHealthStatus,
  projectWorkbenchScenario,
  type QuickAction,
  type WorkbenchAction,
  type WorkbenchContextLinkGroup,
  type WorkbenchMetricGroup,
  type WorkbenchModuleCoverage,
  type WorkbenchPresentation,
  type WorkbenchResourceSummary,
} from './workbench';

export type DashboardResourceLoadState = 'hidden' | 'loading' | 'failed' | 'loaded';

export type DashboardResourceProjection = {
  route?: string;
  state: DashboardResourceLoadState;
  summary?: ContainerDashboardSummary;
};

export type ProductionWorkbenchInput = {
  generatedAt: string;
  quickActionConfig: DashboardQuickActionConfig;
  rankedQuickLinks: DashboardQuickActionLink[];
  resources: DashboardResourceProjection;
  summary: DashboardSummaryResponse;
};

export type DashboardWidgetProjection = {
  items: PresentationItem[];
  metricGroups: WorkbenchMetricGroup[];
  contextLinkGroups: WorkbenchContextLinkGroup[];
};

const DETAILS_ACTION_KEY = 'dashboard.actions.details';

export function projectDashboardSummaryToWorkbench(input: ProductionWorkbenchInput): WorkbenchPresentation {
  const widgetProjections = input.summary.widgets.map(projectWidget);
  const items = widgetProjections.flatMap((projection) => projection.items);
  items.push(...projectResources(input.resources));

  return projectWorkbenchScenario({
    generatedAt: input.generatedAt,
    operational: {
      enabledModules: input.summary.system_summary.modules.enabled_modules,
      failedTasks: input.summary.system_summary.failed_tasks,
      highRiskEvents: input.summary.system_summary.high_risk_events,
    },
    items,
    quickActions: projectQuickActions(input.rankedQuickLinks, input.quickActionConfig),
    moduleCoverage: projectModuleCoverage(input.summary),
    metricGroups: widgetProjections.flatMap((projection) => projection.metricGroups),
    contextLinkGroups: widgetProjections.flatMap((projection) => projection.contextLinkGroups),
    resourceSummary: projectResourceSummary(input.resources),
  });
}

export function projectWidget(widget: DashboardWidget): DashboardWidgetProjection {
  if (widget.status === 'error') {
    return widgetProjection({ items: [projectWidgetLoadFailure(widget)] });
  }
  if (widget.status === 'disabled') {
    return widgetProjection();
  }

  switch (widget.type) {
    case 'alert-list':
      return widgetProjection({ items: projectAlertWidget(widget) });
    case 'health':
      return widgetProjection({ items: projectHealthWidget(widget) });
    case 'timeline':
      return widgetProjection({ items: projectTimelineWidget(widget) });
    case 'link-list': {
      const group = projectContextLinkGroup(widget);
      return group
        ? widgetProjection({ contextLinkGroups: [group] })
        : widgetProjection({ items: [projectInvalidWidgetPayload(widget)] });
    }
    case 'stat-group': {
      const group = projectMetricGroup(widget);
      return group
        ? widgetProjection({ metricGroups: [group] })
        : widgetProjection({ items: [projectInvalidWidgetPayload(widget)] });
    }
    default:
      return widgetProjection();
  }
}

export function projectModuleCoverage(summary: DashboardSummaryResponse): WorkbenchModuleCoverage {
  return {
    registeredModules: summary.system_summary.modules.total_modules,
    enabledModules: summary.system_summary.modules.enabled_modules,
    degradedModules: summary.system_summary.modules.degraded_modules,
    normalContributionSources: summary.widgets.filter((widget) => widget.status === 'normal').length,
    failedContributionSources: summary.widgets.filter((widget) => widget.status === 'error').length,
  };
}

export function projectQuickActions(
  rankedLinks: readonly DashboardQuickActionLink[],
  config: DashboardQuickActionConfig,
): QuickAction[] {
  if (!config.enabled) {
    return [];
  }

  return rankedLinks.map((link, index) => ({
    id: link.id,
    iconKey: link.icon,
    kind: 'navigation',
    titleKey: link.title_key || '',
    titleFallback: link.title || link.full_label || link.id,
    descriptionKey: link.description_key || '',
    descriptionFallback: link.description || link.group || link.section || '',
    route: link.route_location,
    showOnHome: index < config.maxItems,
  }));
}

export function projectResources(resources: DashboardResourceProjection): PresentationItem[] {
  const action = resources.route
    ? {
        kind: 'navigate' as const,
        labelKey: DETAILS_ACTION_KEY,
        route: resources.route,
      }
    : undefined;

  if (resources.state === 'hidden') {
    return [];
  }
  if (resources.state === 'loading') {
    return [
      {
        id: 'container-resources-loading',
        region: 'resources',
        status: PRESENTATION_STATUS.INFO,
        evidenceState: EVIDENCE_STATE.MISSING,
        titleKey: 'dashboard.workbench.resources.loading.title',
        descriptionKey: 'dashboard.workbench.resources.loading.description',
      },
    ];
  }
  if (resources.state === 'failed') {
    return [
      {
        id: 'container-resources-failed',
        region: 'resources',
        status: PRESENTATION_STATUS.WARNING,
        evidenceState: EVIDENCE_STATE.SOURCE_FAILED,
        titleKey: 'dashboard.workbench.resources.unavailable.title',
        descriptionKey: 'dashboard.workbench.resources.unavailable.description',
        actionable: Boolean(action),
        action,
      },
    ];
  }

  const summary = resources.summary;
  if (!summary?.overview.collectedAt) {
    return [
      {
        id: 'container-resources-no-sample',
        region: 'resources',
        status: PRESENTATION_STATUS.INFO,
        evidenceState: EVIDENCE_STATE.MISSING,
        titleKey: 'dashboard.workbench.resources.noSample.title',
        descriptionKey: 'dashboard.workbench.resources.noSample.description',
        actionable: Boolean(action),
        action,
      },
    ];
  }

  return [
    {
      id: 'container-resources-current',
      region: 'resources',
      status: PRESENTATION_STATUS.INFO,
      evidenceState: EVIDENCE_STATE.CONFIRMED,
      titleKey: 'dashboard.workbench.resources.sample.title',
      titleParams: { count: summary.overview.runningContainers },
      descriptionKey: 'dashboard.workbench.resources.sample.description',
      descriptionParams: {
        abnormal: summary.overview.abnormalContainers,
        cpu: summary.overview.cpuTotalPercent.toFixed(1),
      },
      actionable: Boolean(action),
      action,
    },
  ];
}

export function projectResourceSummary(resources: DashboardResourceProjection): WorkbenchResourceSummary {
  const emptySummary = {
    route: resources.route,
    topCpu: [],
    topMemory: [],
    anomalies: [],
  } satisfies Omit<WorkbenchResourceSummary, 'state'>;

  if (resources.state !== 'loaded') {
    return { ...emptySummary, state: resources.state };
  }

  const summary = resources.summary;
  if (!summary?.overview.collectedAt) {
    return { ...emptySummary, state: 'no-sample' };
  }

  return {
    state: 'loaded',
    route: resources.route,
    overview: {
      ...summary.overview,
      collectedAt: summary.overview.collectedAt,
    },
    topCpu: summary.hotspots.cpu.slice(0, 3).map((item) => ({ ...item })),
    topMemory: summary.hotspots.memory.slice(0, 3).map((item) => ({ ...item })),
    anomalies: summary.anomalies.slice(0, 5).map((item) => ({ ...item })),
  };
}

function widgetProjection(overrides: Partial<DashboardWidgetProjection> = {}): DashboardWidgetProjection {
  return {
    items: [],
    metricGroups: [],
    contextLinkGroups: [],
    ...overrides,
  };
}

function projectWidgetGroupMetadata(widget: DashboardWidget) {
  return {
    titleKey: widget.title_key || '',
    titleFallback: widget.title,
    descriptionKey: widget.description_key,
    descriptionFallback: widget.description,
    action: projectWidgetNavigationAction(widget),
    sourceWidgetId: widget.id,
  };
}

function projectMetricGroup(widget: DashboardWidget): WorkbenchMetricGroup | null {
  const payload = asStatGroupPayload(widget.payload);
  if (!payload || !payload.items.every(hasValidStatGroupOptionalFields)) {
    return null;
  }

  return {
    id: `metric-group:${widget.module_key}:${widget.id}`,
    ...projectWidgetGroupMetadata(widget),
    metrics: payload.items.map((item) => ({
      key: item.key,
      labelKey: item.label_key,
      labelFallback: item.label,
      value: item.value,
      unitKey: item.unit_key,
      unitFallback: item.unit,
      descriptionKey: item.description_key,
      descriptionFallback: item.description,
      tone: item.tone ?? 'normal',
      route: item.route_location,
    })),
  };
}

function projectContextLinkGroup(widget: DashboardWidget): WorkbenchContextLinkGroup | null {
  const payload = asLinkListPayload(widget.payload);
  if (!payload || !payload.items.every(hasValidLinkListOptionalFields)) {
    return null;
  }

  return {
    id: `context-link-group:${widget.module_key}:${widget.id}`,
    ...projectWidgetGroupMetadata(widget),
    links: payload.items.map((item) => ({
      key: item.key,
      labelKey: item.label_key,
      labelFallback: item.label,
      descriptionKey: item.description_key,
      descriptionFallback: item.description,
      route: item.route_location,
      iconKey: item.icon,
      badgeKey: item.badge_key,
      badgeFallback: item.badge,
      disabled: item.disabled ?? false,
    })),
  };
}

function projectWidgetNavigationAction(widget: DashboardWidget): WorkbenchAction | undefined {
  if (widget.action) {
    return {
      kind: 'navigate',
      labelKey: widget.action.label_key,
      labelFallback: widget.action.label,
      route: widget.action.route,
    };
  }
  if (widget.route_location) {
    return { kind: 'navigate', labelKey: DETAILS_ACTION_KEY, route: widget.route_location };
  }
  return undefined;
}

function hasValidStatGroupOptionalFields(item: DashboardStatGroupPayload['items'][number]) {
  return (
    isOptionalString(item.unit_key) &&
    isOptionalString(item.unit) &&
    isOptionalString(item.description_key) &&
    isOptionalString(item.description) &&
    isOptionalString(item.route_location)
  );
}

function hasValidLinkListOptionalFields(item: DashboardLinkListPayload['items'][number]) {
  return (
    isOptionalString(item.description_key) &&
    isOptionalString(item.description) &&
    isOptionalString(item.icon) &&
    isOptionalString(item.badge_key) &&
    isOptionalString(item.badge) &&
    (item.disabled === undefined || typeof item.disabled === 'boolean')
  );
}

function isOptionalString(value: unknown) {
  return value === undefined || typeof value === 'string';
}

function projectWidgetLoadFailure(widget: DashboardWidget): PresentationItem {
  return {
    id: `widget-load:${widget.module_key}:${widget.id}`,
    region: 'attention',
    status: PRESENTATION_STATUS.WARNING,
    evidenceState: EVIDENCE_STATE.SOURCE_FAILED,
    titleKey: widget.title_key || 'dashboard.workbench.sourceUnavailable.title',
    titleFallback: widget.title,
    descriptionKey: 'dashboard.workbench.sourceUnavailable.description',
    actionable: true,
    action: {
      kind: 'retry',
      labelKey: 'dashboard.actions.retry',
    },
    sourceWidgetId: widget.id,
  };
}

function projectAlertWidget(widget: DashboardWidget): PresentationItem[] {
  const payload = asAlertListPayload(widget.payload);
  if (!payload) {
    return [projectInvalidWidgetPayload(widget)];
  }

  return payload.items.map((item) => {
    const status = presentationStatusFromAlertLevel(item.level);
    return {
      id: `alert:${widget.module_key}:${widget.id}:${item.id}`,
      region: status === PRESENTATION_STATUS.INFO ? 'activity' : 'attention',
      ...projectConfirmedFact(widget, status, {
        titleKey: item.title_key,
        titleFallback: item.title,
        descriptionKey: item.description_key,
        descriptionFallback: item.description,
        occurredAt: item.occurred_at,
        route: item.route_location,
        actionLabelKey: item.action_label_key,
        actionLabelFallback: item.action_label,
      }),
    } satisfies PresentationItem;
  });
}

function projectHealthWidget(widget: DashboardWidget): PresentationItem[] {
  const payload = asHealthPayload(widget.payload);
  if (!payload) {
    return [projectInvalidWidgetPayload(widget)];
  }

  return payload.items.map((item) => {
    const status = presentationStatusFromHealthStatus(item.status);
    return {
      id: `health:${widget.module_key}:${widget.id}:${item.key}`,
      region: status === PRESENTATION_STATUS.WARNING || status === PRESENTATION_STATUS.UNKNOWN ? 'attention' : 'health',
      status,
      evidenceState:
        status === PRESENTATION_STATUS.UNKNOWN
          ? EVIDENCE_STATE.MISSING
          : status === PRESENTATION_STATUS.INFO
            ? EVIDENCE_STATE.NOT_APPLICABLE
            : EVIDENCE_STATE.CONFIRMED,
      titleKey: item.label_key,
      titleFallback: item.label,
      descriptionKey: item.description_key || '',
      descriptionFallback: item.description,
      actionable: Boolean(item.route_location),
      action: item.route_location
        ? { kind: 'navigate', labelKey: DETAILS_ACTION_KEY, route: item.route_location }
        : undefined,
      sourceWidgetId: widget.id,
    } satisfies PresentationItem;
  });
}

function projectTimelineWidget(widget: DashboardWidget): PresentationItem[] {
  const payload = asTimelinePayload(widget.payload);
  if (!payload) {
    return [projectInvalidWidgetPayload(widget)];
  }

  return payload.items.map((item) => {
    const status = timelineStatus(item.status);
    return {
      id: `timeline:${widget.module_key}:${widget.id}:${item.id}`,
      region: status === PRESENTATION_STATUS.ERROR || status === PRESENTATION_STATUS.WARNING ? 'attention' : 'activity',
      ...projectConfirmedFact(widget, status, {
        titleKey: item.title_key,
        titleFallback: item.title,
        descriptionKey: item.description_key,
        descriptionFallback: item.description,
        occurredAt: item.occurred_at,
        route: item.route_location,
      }),
    } satisfies PresentationItem;
  });
}

function timelineStatus(status: 'normal' | 'success' | 'warning' | 'error' | undefined) {
  if (status === 'error') {
    return PRESENTATION_STATUS.ERROR;
  }
  if (status === 'warning') {
    return PRESENTATION_STATUS.WARNING;
  }
  return PRESENTATION_STATUS.INFO;
}

type ConfirmedFact = {
  actionLabelFallback?: string;
  actionLabelKey?: string;
  descriptionFallback?: string;
  descriptionKey?: string;
  occurredAt?: string;
  route?: string;
  titleFallback?: string;
  titleKey: string;
};

function projectConfirmedFact(
  widget: DashboardWidget,
  status: PresentationItem['status'],
  fact: ConfirmedFact,
): Omit<PresentationItem, 'id' | 'region'> {
  return {
    status,
    evidenceState: EVIDENCE_STATE.CONFIRMED,
    titleKey: fact.titleKey,
    titleFallback: fact.titleFallback,
    descriptionKey: fact.descriptionKey || '',
    descriptionFallback: fact.descriptionFallback,
    occurredAt: fact.occurredAt,
    actionable: Boolean(fact.route),
    action: fact.route
      ? {
          kind: 'navigate',
          labelKey: fact.actionLabelKey || DETAILS_ACTION_KEY,
          labelFallback: fact.actionLabelFallback,
          route: fact.route,
        }
      : undefined,
    sourceWidgetId: widget.id,
  };
}

function projectInvalidWidgetPayload(widget: DashboardWidget): PresentationItem {
  return {
    ...projectWidgetLoadFailure(widget),
    id: `widget-payload:${widget.module_key}:${widget.id}`,
  };
}
