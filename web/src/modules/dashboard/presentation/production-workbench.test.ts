import { describe, expect, it } from 'vitest';

import type { ContainerDashboardSummary } from '@/modules/container/contract/dashboard-summary';

import type { DashboardQuickActionLink } from '../contract/quick-action-links';
import type { DashboardWidget } from '../types/dashboard';
import { projectQuickActions, projectResources, projectWidget } from './production-workbench';

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

describe('production workbench projection', () => {
  it('treats loader failure as retryable source warning even with critical legacy metadata', () => {
    const [item] = projectWidget(
      widget({ id: 'source', status: 'error', state: 'critical', priority: 'critical', title: 'Access Logs' }),
    );

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
    );

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
    );
    const [error] = projectWidget(
      widget({
        id: 'looks-healthy',
        type: 'alert-list',
        payload: {
          items: [{ id: '4xx', level: 'error', title_key: 'ok', title: 'Healthy' }],
        },
      }),
    );

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
    );
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
    );

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
    );

    expect(items.map(({ status, region }) => ({ status, region }))).toEqual([
      { status: 'error', region: 'attention' },
      { status: 'warning', region: 'attention' },
      { status: 'info', region: 'activity' },
      { status: 'info', region: 'activity' },
    ]);
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
