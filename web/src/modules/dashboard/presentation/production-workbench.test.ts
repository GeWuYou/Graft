import { describe, expect, it } from 'vitest';

import type { ContainerDashboardSummary } from '@/modules/container/contract/dashboard-summary';

import type { DashboardQuickActionLink } from '../contract/quick-action-links';
import type { DashboardSummaryResponse, DashboardWidget } from '../types/dashboard';
import {
  projectDashboardSummaryToWorkbench,
  projectModuleCoverage,
  projectQuickActions,
  projectResources,
  projectResourceSummary,
  projectWidget,
} from './production-workbench';

function widget(overrides: Partial<DashboardWidget>): DashboardWidget {
  return {
    category: 'system',
    id: 'widget',
    module_key: 'test',
    order: 1,
    payload: {},
    priority: 'normal',
    size: 'medium',
    state: 'normal',
    type: 'health',
    visible: true,
    ...overrides,
  };
}

function resourceSummary(collectedAt: string | null): ContainerDashboardSummary {
  return {
    overview: {
      abnormalContainers: 0,
      collectedAt,
      cpuTotalPercent: 12.5,
      memoryTotalLimitBytes: null,
      memoryTotalPercent: null,
      memoryTotalUsageBytes: null,
      runningContainers: 2,
    },
    hotspots: { cpu: [], memory: [] },
    anomalies: [],
  };
}

function summaryResponse(widgets: DashboardWidget[]): DashboardSummaryResponse {
  return {
    system_summary: {
      abnormal_services: 0,
      app_env: 'development',
      current_user: { display_name: 'Admin', username: 'admin' },
      failed_tasks: 0,
      high_risk_events: 0,
      locale: { default_locale: 'zh-CN', fallback_locale: 'zh-CN' },
      modules: { degraded_modules: 2, enabled_modules: 10, total_modules: 12 },
      visible_widgets: widgets.length,
    },
    widgets,
  };
}

function resourceHotspot(index: number) {
  return {
    collectedAt: '2026-01-01T00:00:00Z',
    cpuPercent: index + 0.5,
    health: 'healthy',
    id: `container-${index}`,
    image: `image-${index}`,
    memoryLimitBytes: 1024,
    memoryPercent: index + 1,
    memoryUsageBytes: 512,
    name: `Container ${index}`,
    restartCount: index,
    shortId: `short-${index}`,
    state: 'running',
  };
}

describe('production workbench projection', () => {
  it('treats loader failure as retryable source warning even with critical legacy metadata', () => {
    const [item] = projectWidget(
      widget({ id: 'source', status: 'error', state: 'critical', priority: 'critical', title: 'Access Logs' }),
    ).items;

    expect(item).toMatchObject({
      status: 'warning',
      evidenceState: 'source-failed',
      region: 'attention',
      sourceWidgetId: 'source',
      action: { kind: 'retry' },
    });
  });

  it('maps each typed health state without reading widget state or priority', () => {
    const items = projectWidget(
      widget({
        state: 'critical',
        priority: 'critical',
        payload: {
          summary: { status: 'degraded' },
          items: [
            { key: 'db', label_key: 'db', label: 'DB', status: 'healthy' },
            { key: 'cache', label_key: 'cache', label: 'Cache', status: 'degraded' },
            { key: 'network', label_key: 'network', label: 'Network', status: 'unknown' },
            { key: 'optional', label_key: 'optional', label: 'Optional', status: 'disabled' },
          ],
        },
      }),
    ).items;

    expect(items.map(({ status, evidenceState, region }) => ({ status, evidenceState, region }))).toEqual([
      { status: 'healthy', evidenceState: 'confirmed', region: 'health' },
      { status: 'warning', evidenceState: 'confirmed', region: 'attention' },
      { status: 'unknown', evidenceState: 'missing', region: 'attention' },
      { status: 'info', evidenceState: 'not-applicable', region: 'health' },
    ]);
  });

  it('uses authoritative alert level despite misleading ids and titles', () => {
    const [warning] = projectWidget(
      widget({
        id: 'looks-critical',
        type: 'alert-list',
        payload: {
          items: [
            {
              id: 'error.5xx',
              level: 'warning',
              title_key: 'healthy.title',
              title: 'Critical server error',
            },
          ],
        },
      }),
    ).items;
    const [error] = projectWidget(
      widget({
        id: 'looks-healthy',
        type: 'alert-list',
        payload: {
          items: [{ id: '4xx', level: 'error', title_key: 'ok', title: 'Healthy' }],
        },
      }),
    ).items;

    expect(warning?.status).toBe('warning');
    expect(error?.status).toBe('error');
  });

  it('projects real key-first audit and access-log alerts as confirmed facts', () => {
    const [audit] = projectWidget(
      widget({
        id: 'audit.risk-events',
        module_key: 'audit',
        type: 'alert-list',
        payload: {
          items: [
            {
              count: 2,
              id: 'audit.high-risk',
              level: 'error',
              title_key: 'dashboard.widget.auditRiskEvents.highRisk.title',
            },
          ],
        },
      }),
    ).items;
    const [accessLog] = projectWidget(
      widget({
        id: 'core.httpx.access-log.request-attention',
        module_key: 'core.httpx',
        type: 'alert-list',
        payload: {
          items: [
            {
              count: 1,
              id: 'error.5xx',
              level: 'error',
              title_key: 'dashboard.widget.accessLogRequestAttention.error',
            },
          ],
        },
      }),
    ).items;

    expect(audit).toMatchObject({
      id: 'alert:audit:audit.risk-events:audit.high-risk',
      evidenceState: 'confirmed',
      sourceWidgetId: 'audit.risk-events',
      titleKey: 'dashboard.widget.auditRiskEvents.highRisk.title',
    });
    expect(accessLog).toMatchObject({
      id: 'alert:core.httpx:core.httpx.access-log.request-attention:error.5xx',
      evidenceState: 'confirmed',
      sourceWidgetId: 'core.httpx.access-log.request-attention',
      titleKey: 'dashboard.widget.accessLogRequestAttention.error',
    });
    expect(audit?.id).not.toContain('widget-payload:');
    expect(accessLog?.id).not.toContain('widget-payload:');
  });

  it('routes only typed timeline failures to Attention and keeps successful activity neutral', () => {
    const items = projectWidget(
      widget({
        type: 'timeline',
        payload: {
          items: [
            {
              id: 'failed',
              title_key: 'failed',
              title: 'Failed',
              occurred_at: '2026-01-01T00:00:00Z',
              status: 'error',
            },
            { id: 'slow', title_key: 'slow', title: 'Slow', occurred_at: '2026-01-01T00:00:01Z', status: 'warning' },
            { id: 'done', title_key: 'done', title: 'Done', occurred_at: '2026-01-01T00:00:02Z', status: 'success' },
            { id: 'run', title_key: 'run', title: 'Run', occurred_at: '2026-01-01T00:00:03Z', status: 'normal' },
          ],
        },
      }),
    ).items;

    expect(items.map(({ status, region }) => ({ status, region }))).toEqual([
      { status: 'error', region: 'attention' },
      { status: 'warning', region: 'attention' },
      { status: 'info', region: 'activity' },
      { status: 'info', region: 'activity' },
    ]);
  });

  it('projects stat groups with widget and metric navigation without flattening them into attention items', () => {
    const projection = projectWidget(
      widget({
        id: 'scheduler.task-attention',
        module_key: 'scheduler',
        type: 'stat-group',
        title_key: 'scheduler.title',
        title: 'Scheduler',
        description_key: 'scheduler.description',
        description: 'Scheduler status',
        action: { label_key: 'scheduler.action', label: 'View tasks', route: '/scheduler/tasks' },
        payload: {
          items: [
            {
              key: 'failed',
              label_key: 'scheduler.failed',
              label: 'Failed',
              value: '2',
              unit_key: 'scheduler.unit.tasks',
              unit: 'tasks',
              description_key: 'scheduler.failed.description',
              description: 'Last run failed',
              tone: 'error',
              route_location: '/scheduler/tasks?status=failed',
            },
          ],
        },
      }),
    );

    expect(projection.items).toEqual([]);
    expect(projection.metricGroups[0]).toMatchObject({
      id: 'metric-group:scheduler:scheduler.task-attention',
      titleKey: 'scheduler.title',
      descriptionKey: 'scheduler.description',
      sourceWidgetId: 'scheduler.task-attention',
      action: { kind: 'navigate', labelKey: 'scheduler.action', route: '/scheduler/tasks' },
      metrics: [
        {
          key: 'failed',
          labelKey: 'scheduler.failed',
          value: '2',
          unitKey: 'scheduler.unit.tasks',
          descriptionKey: 'scheduler.failed.description',
          tone: 'error',
          route: '/scheduler/tasks?status=failed',
        },
      ],
    });
  });

  it('turns malformed stat-group payloads into retryable source warnings', () => {
    const missingValue = projectWidget(
      widget({
        id: 'invalid-stat-required',
        type: 'stat-group',
        payload: { items: [{ key: 'failed', label_key: 'failed', label: 'Failed' }] },
      }),
    );
    const invalidOptional = projectWidget(
      widget({
        id: 'invalid-stat-optional',
        type: 'stat-group',
        payload: {
          items: [{ key: 'failed', label_key: 'failed', label: 'Failed', route_location: 42, value: '1' }],
        },
      }),
    );

    expect(missingValue.metricGroups).toEqual([]);
    expect(invalidOptional.metricGroups).toEqual([]);
    expect(missingValue.items[0]).toMatchObject({
      id: 'widget-payload:test:invalid-stat-required',
      status: 'warning',
      evidenceState: 'source-failed',
      action: { kind: 'retry' },
    });
    expect(invalidOptional.items[0]?.id).toBe('widget-payload:test:invalid-stat-optional');
  });

  it('keeps complete contextual link groups separate from ranked quick entries', () => {
    const links = Array.from({ length: 8 }, (_, index) => ({
      key: `context-${index}`,
      label_key: `context.${index}`,
      label: `Context ${index}`,
      description_key: `context.${index}.description`,
      description: `Context description ${index}`,
      route_location: `/context/${index}`,
      icon: `icon-${index}`,
      badge_key: `badge.${index}`,
      badge: `${index}`,
      disabled: index === 7,
    }));
    const linkWidget = widget({
      id: 'operations',
      module_key: 'operations',
      type: 'link-list',
      status: 'normal',
      title_key: 'operations.title',
      route_location: '/operations',
      payload: { items: links },
    });
    const quickLink = {
      id: 'quick-only',
      module_key: 'quick',
      order: 1,
      route_location: '/quick',
      title: 'Quick only',
    } satisfies DashboardQuickActionLink;
    const presentation = projectDashboardSummaryToWorkbench({
      generatedAt: '2026-01-01T00:00:00Z',
      quickActionConfig: { enabled: true, maxItems: 6, strategy: 'hybrid' },
      rankedQuickLinks: [quickLink],
      resources: { state: 'hidden' },
      summary: summaryResponse([linkWidget]),
    });

    expect(presentation.contextLinkGroups[0]).toMatchObject({
      id: 'context-link-group:operations:operations',
      action: { kind: 'navigate', route: '/operations' },
    });
    expect(presentation.contextLinkGroups[0]?.links).toHaveLength(8);
    expect(presentation.contextLinkGroups[0]?.links[0]).toMatchObject({
      key: 'context-0',
      labelKey: 'context.0',
      descriptionKey: 'context.0.description',
      route: '/context/0',
      iconKey: 'icon-0',
      badgeKey: 'badge.0',
      disabled: false,
    });
    expect(presentation.contextLinkGroups[0]?.links[7]?.disabled).toBe(true);
    expect(presentation.quickActions.map(({ id }) => id)).toEqual(['quick-only']);
    expect(presentation.homeQuickActions.map(({ id }) => id)).toEqual(['quick-only']);
  });

  it('turns malformed link-list payloads into retryable source warnings', () => {
    const projection = projectWidget(
      widget({
        id: 'invalid-links',
        type: 'link-list',
        payload: {
          items: [{ key: 'logs', label_key: 'logs', label: 'Logs', route_location: '/logs', disabled: 'false' }],
        },
      }),
    );

    expect(projection.contextLinkGroups).toEqual([]);
    expect(projection.items[0]).toMatchObject({
      id: 'widget-payload:test:invalid-links',
      status: 'warning',
      evidenceState: 'source-failed',
    });
  });

  it('projects module and contribution-source coverage without counting disabled or warning sources as failures', () => {
    const widgets = [
      widget({ id: 'normal', status: 'normal' }),
      widget({ id: 'failed', status: 'error' }),
      widget({ id: 'disabled', status: 'disabled' }),
      widget({ id: 'warning', status: 'warning' }),
      widget({ id: 'unclassified', status: undefined }),
    ];

    expect(projectModuleCoverage(summaryResponse(widgets))).toEqual({
      registeredModules: 12,
      enabledModules: 10,
      degradedModules: 2,
      normalContributionSources: 1,
      failedContributionSources: 1,
    });
  });

  it('keeps resource request failure, missing sample, and confirmed zero sample distinct', () => {
    expect(projectResources({ state: 'failed' })[0]).toMatchObject({
      status: 'warning',
      evidenceState: 'source-failed',
    });
    expect(projectResources({ state: 'loaded', summary: resourceSummary(null) })[0]).toMatchObject({
      status: 'info',
      evidenceState: 'missing',
    });
    const confirmed = resourceSummary('2026-01-01T00:00:00Z');
    confirmed.overview.runningContainers = 0;
    expect(projectResources({ state: 'loaded', summary: confirmed })[0]).toMatchObject({
      status: 'info',
      evidenceState: 'confirmed',
    });
    expect(projectResourceSummary({ state: 'failed' })).toMatchObject({ state: 'failed', topCpu: [], topMemory: [] });
    const noSample = projectResourceSummary({ state: 'loaded', summary: resourceSummary(null) });
    expect(noSample).toMatchObject({ state: 'no-sample' });
    expect(noSample).not.toHaveProperty('overview');
  });

  it('keeps bounded resource details in API order without inventing severity', () => {
    const summary = resourceSummary('2026-01-01T00:00:00Z');
    summary.hotspots.cpu = Array.from({ length: 5 }, (_, index) => resourceHotspot(index));
    summary.hotspots.memory = Array.from({ length: 5 }, (_, index) => resourceHotspot(index + 10));
    summary.anomalies = Array.from({ length: 7 }, (_, index) => ({
      ...resourceHotspot(index + 20),
      reasonCode: 'state.restarting',
      reasonLabel: 'Restarting',
      status: 'restarting',
    }));

    const projected = projectResourceSummary({ route: '/containers', state: 'loaded', summary });

    expect(projected).toMatchObject({
      state: 'loaded',
      route: '/containers',
      overview: { collectedAt: '2026-01-01T00:00:00Z', runningContainers: 2 },
    });
    expect(projected.topCpu.map(({ id }) => id)).toEqual(['container-0', 'container-1', 'container-2']);
    expect(projected.topMemory.map(({ id }) => id)).toEqual(['container-10', 'container-11', 'container-12']);
    expect(projected.anomalies.map(({ id }) => id)).toEqual([
      'container-20',
      'container-21',
      'container-22',
      'container-23',
      'container-24',
    ]);
  });

  it('uses configured home count without limiting the complete action collection', () => {
    const links = Array.from({ length: 8 }, (_, index) => ({
      id: `link-${index}`,
      module_key: 'test',
      order: index,
      route_location: `/test/${index}`,
      title: `Link ${index}`,
    })) satisfies DashboardQuickActionLink[];

    expect(projectQuickActions(links, { enabled: false, maxItems: 4, strategy: 'hybrid' })).toHaveLength(0);
    const actions = projectQuickActions(links, { enabled: true, maxItems: 6, strategy: 'hybrid' });
    expect(actions).toHaveLength(8);
    expect(actions.filter((action) => action.showOnHome)).toHaveLength(6);
  });
});
