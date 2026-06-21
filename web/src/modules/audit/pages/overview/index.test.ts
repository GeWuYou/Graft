import { flushPromises, mount } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';
import { createI18n } from 'vue-i18n';

import { AUDIT_ROUTE_PATH } from '../../contract/paths';
import type { AuditOverviewResponse } from '../../types/audit';
import AuditOverviewPage from './index.vue';

const { getAuditOverviewMock } = vi.hoisted(() => ({
  getAuditOverviewMock: vi.fn(),
}));

const chartMocks = vi.hoisted(() => {
  const setOption = vi.fn();
  const resize = vi.fn();
  const dispose = vi.fn();
  const init = vi.fn(() => ({
    setOption,
    resize,
    dispose,
  }));

  return {
    init,
    setOption,
    resize,
    dispose,
  };
});

const resizeObserverMocks = vi.hoisted(() => {
  const observe = vi.fn();
  const disconnect = vi.fn();
  let callback: ResizeObserverCallback | null = null;

  class ResizeObserverMock {
    constructor(nextCallback: ResizeObserverCallback) {
      callback = nextCallback;
    }

    observe = observe;
    disconnect = disconnect;
  }

  return {
    ResizeObserverMock,
    observe,
    disconnect,
    trigger() {
      callback?.([], {} as ResizeObserver);
    },
  };
});

const settingStoreMock = vi.hoisted(() => ({
  displayMode: 'light',
  brandTheme: '#0052D9',
  chartColors: {
    textColor: 'rgba(0, 0, 0, 0.9)',
    placeholderColor: 'rgba(0, 0, 0, 0.35)',
    borderColor: '#dcdcdc',
    containerColor: '#ffffff',
  },
}));

const routerMocks = {
  push: vi.fn(),
};

function createTrendPoint(
  bucketStart: string,
  bucketEnd: string,
  total: number,
  failed: number,
  highRisk: number,
  securityEvents: number,
): NonNullable<AuditOverviewResponse['trend']>['points'][number] {
  return {
    bucket_start: bucketStart,
    bucket_end: bucketEnd,
    total,
    failed,
    high_risk: highRisk,
    security_events: securityEvents,
  };
}

function createAuditOverviewResponse(): AuditOverviewResponse {
  return {
    time_preset: 'last_24h',
    summary: {
      total_logs: 12,
      failed_operations: 3,
      high_risk_events: 5,
      sensitive_operations: 4,
    },
    failed_auth: [
      {
        id: 1,
        actor_display_name: 'ops-admin',
        action: 'POST /api/auth/login',
        success: false,
        request_id: 'req-1',
        message: '',
        metadata: { request_path: '/api/auth/login' },
        created_at: '2026-05-27T08:00:00Z',
      },
    ],
    permission_denied: [
      {
        id: 2,
        actor_display_name: 'viewer-01',
        action: 'rbac.role.delete',
        success: false,
        request_id: 'req-2',
        message: 'common.forbidden',
        metadata: { request_path: '/api/roles/1/delete' },
        created_at: '2026-05-27T08:05:00Z',
      },
    ],
    security_timeline: [
      {
        id: 11,
        incident_seed: { event_id: 42 },
        created_at: '2026-05-27T08:12:00Z',
        source: 'SECURITY_EVENT',
        risk_level: 'HIGH',
        action: 'auth.failed',
        result: 'FAILED',
        request_id: 'req-incident-42',
        resource_name: 'console',
      },
    ],
    risk_groups: [],
    trend: { bucket_unit: 'hour', bucket_size: 1, points: [] },
    sensitive_operations: [
      {
        id: 3,
        actor_display_name: 'security-lead',
        resource_name: 'alice',
        resource_type: 'user',
        resource_id: '42',
        action: 'user.password.reset',
        success: true,
        request_id: 'req-3',
        message: '',
        metadata: { request_path: '/api/users/42/reset-password' },
        created_at: '2026-05-27T08:10:00Z',
      },
    ],
  };
}

vi.mock('../../api/audit', () => ({
  getAuditOverview: getAuditOverviewMock,
}));

vi.mock('@/shared/localized-api-error', () => ({
  resolveLocalizedErrorMessage: () => 'load failed',
}));

vi.mock('@/utils/logger', () => ({
  createLogger: () => ({
    error: vi.fn(),
  }),
}));

vi.mock('@/store', () => ({
  useSettingStore: () => settingStoreMock,
}));

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: routerMocks.push,
  }),
}));

vi.mock('echarts/core', async () => {
  const actual = await vi.importActual<typeof import('echarts/core')>('echarts/core');
  return {
    ...actual,
    init: chartMocks.init,
    use: vi.fn(),
  };
});

const passthroughStub = defineComponent({
  name: 'PassthroughStub',
  props: ['title', 'description', 'items', 'value', 'valueAside'],
  setup(props, { slots }) {
    return () =>
      h('div', [
        props.title,
        props.description,
        props.value,
        props.valueAside,
        JSON.stringify(props.items),
        slots.summary?.(),
        slots.default?.(),
        slots.actions?.(),
      ]);
  },
});

const shellStub = defineComponent({
  name: 'GovernanceDashboardShellStub',
  props: ['eyebrow', 'title', 'description', 'titleKey', 'descriptionKey'],
  setup(props, { slots }) {
    const resolveText = (key?: unknown, fallback?: unknown) => {
      if (typeof key !== 'string' || key.length === 0) {
        return typeof fallback === 'string' ? fallback : '';
      }

      const translated = i18n.global.t(key);
      return translated && translated !== key ? translated : typeof fallback === 'string' ? fallback : '';
    };

    return () =>
      h('div', { 'data-page-type': 'overview-dashboard' }, [
        props.eyebrow,
        resolveText(props.titleKey, props.title),
        resolveText(props.descriptionKey, props.description),
        slots.actions?.(),
        slots.summary?.(),
        slots.default?.(),
      ]);
  },
});

const sectionStub = defineComponent({
  name: 'GovernanceSectionStub',
  props: ['title'],
  setup(props, { slots }) {
    return () => h('section', [props.title, slots.default?.()]);
  },
});

const summaryCardStub = defineComponent({
  name: 'GovernanceSummaryCardStub',
  props: ['title', 'value', 'valueAside'],
  setup(props) {
    return () => h('div', [props.title, props.value, props.valueAside]);
  },
});

const buttonStub = defineComponent({
  name: 'TButtonStub',
  emits: ['click'],
  setup(_, { emit, slots, attrs }) {
    return () => h('button', { ...attrs, onClick: () => emit('click') }, slots.default?.());
  },
});

const radioGroupStub = defineComponent({
  name: 'TRadioGroupStub',
  props: { modelValue: { type: String, default: '' } },
  setup(_, { slots }) {
    return () => h('div', slots.default?.());
  },
});

const radioButtonStub = defineComponent({
  name: 'TRadioButtonStub',
  setup(_, { slots }) {
    return () => h('button', slots.default?.());
  },
});

const spaceStub = defineComponent({
  name: 'TSpaceStub',
  setup(_, { slots }) {
    return () => h('div', slots.default?.());
  },
});

const tagStub = defineComponent({
  name: 'TTagStub',
  setup(_, { slots }) {
    return () => h('span', slots.default?.());
  },
});

const tooltipStub = defineComponent({
  name: 'TTooltipStub',
  setup(_, { slots }) {
    return () => h('div', [slots.content?.(), slots.default?.()]);
  },
});

const timelineStub = defineComponent({
  name: 'TTimelineStub',
  setup(_, { slots }) {
    return () => h('div', slots.default?.());
  },
});

const timelineItemStub = defineComponent({
  name: 'TTimelineItemStub',
  props: ['label', 'dotColor'],
  setup(props, { slots }) {
    return () => h('div', [props.label, props.dotColor, slots.default?.()]);
  },
});

const i18n = createI18n({
  legacy: false,
  missingWarn: false,
  fallbackWarn: false,
  locale: 'en-US',
  messages: {
    'en-US': {
      menu: {
        audit: {
          title: 'Security Audit',
          overview: {
            title: 'Security Audit',
          },
        },
      },
      audit: {
        overview: {
          title: 'Security Audit Overview',
          description:
            'Focus on security-audit events tied to authentication, authorization, and sensitive operations, excluding health checks, monitor polling, and page-load noise.',
          refresh: 'Refresh',
          retry: 'Retry',
          errorTitle: 'Audit overview is temporarily unavailable',
          loadFailed: 'Failed to load audit overview',
          timeRanges: { '24h': 'Last 24h', '7d': 'Last 7d', '30d': 'Last 30d' },
          itemResult: { failed: 'Failed', denied: 'Denied', sensitive: 'Review' },
          sections: {
            failedAuth: 'Recent Authentication Failures',
            permissionDenied: 'Recent Permission Denials',
            sensitiveOps: 'Recent Sensitive Audit Events',
            trend: 'Risk Trend',
            securityTimeline: 'Security Event Timeline',
            shortcuts: 'Quick Links',
            riskWatch: 'Recent Risk',
          },
          trend: {
            emptyTitle: 'Not enough risk events yet',
            emptyDescription: 'Trend analysis will appear after more audit events are collected.',
            totalMetric: 'Total events',
            highRiskMetric: 'High risk',
            securityMetric: 'Security events',
            totalValue: 'Total {value}',
            highRiskValue: 'High risk: {value}',
            securityValue: 'Security events: {value}',
          },
          timeline: {
            openEvent: 'Open Event',
          },
          empty: {
            failedAuth: {
              title: 'No authentication failures',
              description: 'No failed sign-ins or authentication audit events were found in the selected window.',
            },
            permissionDenied: {
              title: 'No permission denials',
              description: 'No authorization denials were found in the selected window.',
            },
            sensitiveOps: {
              title: 'No sensitive audit events',
              description:
                'No export, delete, reset, grant, or other privileged write events were found in the selected window.',
            },
            riskGroups: {
              title: 'No active risk groups',
              description: 'No grouped audit risks need attention in the selected window.',
            },
            securityTimeline: {
              title: 'No security timeline events',
              description: 'No security events were found in the selected window.',
            },
          },
          riskGroups: {
            criticalSecurity: 'Critical Security Failures',
            meta: '{count} events in the current window',
            action: 'View related events',
          },
          stats: {
            totalLogs: { title: 'Audit Log Count', unit: 'events', meta: 'all records' },
            failedWindow: { title: 'Security Failure Count', unit: 'events', meta: 'current window' },
            highRisk: { title: 'High-Risk Event Count', unit: 'items', meta: 'current window' },
            sensitiveOps: { title: 'Sensitive Operation Count', unit: 'actions', meta: 'current window' },
          },
          shortcuts: {
            failedAuth: {
              title: 'Open Failed Authentication',
              description: 'Review failed sign-ins, token failures, and other authentication audit events',
            },
            rbacChanges: {
              title: 'Open Permission Configuration Changes',
              description: 'Review role, permission, resource, and menu configuration changes',
            },
            sensitiveOps: {
              title: 'Open Sensitive Operations',
              description: 'Locate export, delete, and other privileged write audit events',
            },
          },
        },
        logList: {
          drawer: {
            actions: {
              viewRelatedRequest: 'View Related Request',
            },
          },
        },
        actionLabel: {
          auth: {
            failed: 'Authentication Failed',
            permission: {
              denied: 'Permission Denied',
            },
          },
        },
        common: {
          targetType: {
            permission: 'Permission',
          },
          risk: { HIGH: 'High', CRITICAL: 'Critical' },
          source: { SECURITY_EVENT: 'Security Event' },
        },
      },
    },
  },
});

function mountOverview() {
  return mount(AuditOverviewPage, {
    global: {
      plugins: [i18n],
      stubs: {
        'governance-dashboard-shell': shellStub,
        'governance-section': sectionStub,
        'governance-summary-card': summaryCardStub,
        'management-empty-state': passthroughStub,
        't-button': buttonStub,
        't-radio-group': radioGroupStub,
        't-radio-button': radioButtonStub,
        't-space': spaceStub,
        't-tag': tagStub,
        't-tooltip': tooltipStub,
        't-timeline': timelineStub,
        't-timeline-item': timelineItemStub,
      },
    },
  });
}

describe('AuditOverviewPage', () => {
  beforeEach(() => {
    getAuditOverviewMock.mockReset();
    getAuditOverviewMock.mockResolvedValue(createAuditOverviewResponse());
    routerMocks.push.mockReset();
    chartMocks.init.mockClear();
    chartMocks.setOption.mockClear();
    chartMocks.resize.mockClear();
    chartMocks.dispose.mockClear();
    resizeObserverMocks.observe.mockClear();
    resizeObserverMocks.disconnect.mockClear();
    vi.stubGlobal('ResizeObserver', resizeObserverMocks.ResizeObserverMock);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('renders the streamlined workbench overview and opens a quick link with canonical filter keys', async () => {
    const wrapper = mountOverview();

    await flushPromises();

    expect(getAuditOverviewMock).toHaveBeenCalledWith({ preset: 'last_24h' });
    expect(wrapper.attributes('data-page-type')).toBe('overview-dashboard');
    expect(wrapper.text()).toContain('Security Audit Overview');
    expect(wrapper.text()).toContain('excluding health checks, monitor polling, and page-load noise');
    expect(wrapper.text()).toContain('Recent Authentication Failures');
    expect(wrapper.text()).toContain('Recent Permission Denials');
    expect(wrapper.text()).toContain('Recent Sensitive Audit Events');
    expect(wrapper.text()).toContain('Quick Links');
    expect(wrapper.text()).toContain('Refresh');
    expect(wrapper.text()).toContain('Not enough risk events yet');
    expect(wrapper.text()).toContain('Trend analysis will appear after more audit events are collected.');

    await wrapper.get('button[type="button"]').trigger('click');
    const firstQuery = routerMocks.push.mock.calls[0]?.[0]?.query ?? {};
    expect(routerMocks.push).toHaveBeenCalledWith(
      expect.objectContaining({
        path: AUDIT_ROUTE_PATH.LOGS,
        query: expect.objectContaining({
          created_from: expect.any(String),
          created_to: expect.any(String),
        }),
      }),
    );
    expect(firstQuery).not.toHaveProperty('preset');
  });

  it('opens the failed summary card with explicit failed-operation filters', async () => {
    routerMocks.push.mockClear();

    const wrapper = mountOverview();

    await flushPromises();

    await wrapper.findAll('button[type="button"]')[1]!.trigger('click');

    expect(routerMocks.push).toHaveBeenCalledWith({
      path: AUDIT_ROUTE_PATH.LOGS,
      query: expect.objectContaining({
        scope: 'failed_operations',
        created_from: expect.any(String),
        created_to: expect.any(String),
      }),
    });
  });

  it('opens the high-risk summary card with canonical summary query params', async () => {
    routerMocks.push.mockClear();

    const wrapper = mountOverview();

    await flushPromises();

    await wrapper.findAll('button[type="button"]')[2]!.trigger('click');

    expect(routerMocks.push).toHaveBeenCalledWith({
      path: AUDIT_ROUTE_PATH.LOGS,
      query: expect.objectContaining({
        scope: 'high_risk_operations',
        created_from: expect.any(String),
        created_to: expect.any(String),
      }),
    });
  });

  it('opens risk groups with canonical visible audit filters', async () => {
    routerMocks.push.mockClear();
    getAuditOverviewMock.mockResolvedValueOnce({
      time_preset: 'last_24h',
      summary: {
        total_logs: 12,
        failed_operations: 3,
        high_risk_events: 5,
        sensitive_operations: 4,
      },
      failed_auth: [],
      permission_denied: [],
      security_timeline: [],
      risk_groups: [
        {
          key: 'high_risk_operations',
          label_key: 'audit.overview.riskGroups.criticalSecurity',
          count: 3,
          risk_level: 'CRITICAL',
        },
      ],
      trend: { bucket_unit: 'hour', bucket_size: 1, points: [] },
      sensitive_operations: [],
    });

    const wrapper = mountOverview();

    await flushPromises();

    const riskGroupButton = wrapper.findAll('button').find((item) => item.text().includes('View related events'));

    expect(riskGroupButton).toBeTruthy();
    await riskGroupButton!.trigger('click');

    expect(routerMocks.push).toHaveBeenCalledWith({
      path: AUDIT_ROUTE_PATH.LOGS,
      query: expect.objectContaining({
        scope: 'high_risk_operations',
        created_from: expect.any(String),
        created_to: expect.any(String),
      }),
    });
  });

  it('opens sensitive summary with the same keyword scope used by overview counters', async () => {
    routerMocks.push.mockClear();

    const wrapper = mountOverview();

    await flushPromises();

    await wrapper.findAll('button[type="button"]')[3]!.trigger('click');

    expect(routerMocks.push).toHaveBeenCalledWith({
      path: AUDIT_ROUTE_PATH.LOGS,
      query: expect.objectContaining({
        scope: 'sensitive_operations',
        created_from: expect.any(String),
        created_to: expect.any(String),
      }),
    });
  });

  it('renders explicit empty states for empty overview activity sections', async () => {
    getAuditOverviewMock.mockResolvedValueOnce({
      time_preset: 'last_24h',
      summary: {
        total_logs: 0,
        failed_operations: 0,
        high_risk_events: 0,
        sensitive_operations: 0,
      },
      failed_auth: [],
      permission_denied: [],
      security_timeline: [],
      risk_groups: [],
      trend: { bucket_unit: 'hour', bucket_size: 1, points: [] },
      sensitive_operations: [],
    });

    const wrapper = mountOverview();

    await flushPromises();

    expect(wrapper.text()).toContain('No authentication failures');
    expect(wrapper.text()).toContain('No permission denials');
    expect(wrapper.text()).toContain('No sensitive audit events');
    expect(wrapper.text()).toContain('No active risk groups');
    expect(wrapper.text()).toContain('No security timeline events');
    expect(wrapper.text()).toContain(
      'No export, delete, reset, grant, or other privileged write events were found in the selected window.',
    );
  });

  it('uses the updated overview stat labels', async () => {
    const wrapper = mountOverview();

    await flushPromises();

    const summaryCards = wrapper
      .findAllComponents({ name: 'GovernanceSummaryCardStub' })
      .map((item) => item.props('title'))
      .filter((value) => typeof value === 'string');

    expect(summaryCards).toContain('Audit Log Count');
    expect(summaryCards).toContain('Security Failure Count');
    expect(summaryCards).toContain('High-Risk Event Count');
    expect(summaryCards).toContain('Sensitive Operation Count');
  });

  it('renders the trend chart only when enough meaningful points are present', async () => {
    const overviewWithTrend: AuditOverviewResponse = {
      time_preset: 'last_24h',
      summary: {
        total_logs: 18,
        failed_operations: 4,
        high_risk_events: 6,
        sensitive_operations: 5,
      },
      failed_auth: [],
      permission_denied: [],
      security_timeline: [],
      risk_groups: [],
      trend: {
        bucket_unit: 'hour',
        bucket_size: 1,
        points: [
          createTrendPoint('2026-05-27T08:00:00Z', '2026-05-27T09:00:00Z', 4, 1, 1, 1),
          createTrendPoint('2026-05-27T09:00:00Z', '2026-05-27T10:00:00Z', 7, 2, 3, 2),
          createTrendPoint('2026-05-27T10:00:00Z', '2026-05-27T11:00:00Z', 5, 1, 2, 1),
        ],
      },
      sensitive_operations: [],
    };

    getAuditOverviewMock.mockResolvedValueOnce(overviewWithTrend);

    const wrapper = mountOverview();

    await flushPromises();

    expect(wrapper.text()).not.toContain('Not enough risk events yet');
    expect(wrapper.text()).toContain('Total events');
    expect(wrapper.text()).toContain('High risk');
    expect(wrapper.text()).toContain('Security events');
    expect(wrapper.find('[data-audit-trend-chart="true"]').exists()).toBe(true);
    expect(chartMocks.init).toHaveBeenCalledTimes(1);
    expect(chartMocks.setOption).toHaveBeenCalled();

    const option = chartMocks.setOption.mock.calls.at(-1)?.[0];
    expect(option?.series).toHaveLength(3);
    expect(option?.series?.map((item: { name: string }) => item.name)).toEqual([
      'Total events',
      'High risk',
      'Security events',
    ]);
    expect(option?.series?.[0]?.data).toEqual([4, 7, 5]);
    expect(option?.series?.[1]?.data).toEqual([1, 3, 2]);
    expect(option?.series?.[2]?.data).toEqual([1, 2, 1]);
    expect(option?.yAxis?.axisLabel?.formatter(12)).toBe('12');
    expect(
      option?.tooltip?.formatter([
        {
          axisValue: '2026-05-27T08:00:00Z-2026-05-27T09:00:00Z',
          axisValueLabel: '05/27',
          seriesName: 'Total events',
          color: '#0052D9',
          data: 4,
        },
        {
          axisValue: '2026-05-27T08:00:00Z-2026-05-27T09:00:00Z',
          axisValueLabel: '05/27',
          seriesName: 'High risk',
          color: '#ED7B2F',
          data: 1,
        },
        {
          axisValue: '2026-05-27T08:00:00Z-2026-05-27T09:00:00Z',
          axisValueLabel: '05/27',
          seriesName: 'Security events',
          color: '#E34D59',
          data: 1,
        },
      ]),
    ).toContain('Total events');
    expect(option?.xAxis?.axisLabel?.formatter('2026-05-27T08:00:00Z-2026-05-27T09:00:00Z')).not.toBe('');
  });

  it('registers resize observation after the chart container appears', async () => {
    getAuditOverviewMock.mockResolvedValueOnce({
      ...createAuditOverviewResponse(),
      trend: {
        bucket_unit: 'hour',
        bucket_size: 1,
        points: [
          createTrendPoint('2026-05-27T08:00:00Z', '2026-05-27T09:00:00Z', 4, 1, 1, 1),
          createTrendPoint('2026-05-27T09:00:00Z', '2026-05-27T10:00:00Z', 7, 2, 3, 2),
          createTrendPoint('2026-05-27T10:00:00Z', '2026-05-27T11:00:00Z', 5, 1, 2, 1),
        ],
      },
    });

    mountOverview();

    await flushPromises();

    expect(resizeObserverMocks.observe).toHaveBeenCalled();
  });

  it('renders the chart for sparse seven-day trend activity', async () => {
    getAuditOverviewMock.mockResolvedValueOnce({
      ...createAuditOverviewResponse(),
      time_preset: 'last_7d',
      trend: {
        bucket_unit: 'day',
        bucket_size: 1,
        points: [
          createTrendPoint('2026-05-21T08:00:00Z', '2026-05-22T08:00:00Z', 0, 0, 0, 0),
          createTrendPoint('2026-05-22T08:00:00Z', '2026-05-23T08:00:00Z', 4, 1, 4, 1),
          createTrendPoint('2026-05-23T08:00:00Z', '2026-05-24T08:00:00Z', 0, 0, 0, 0),
          createTrendPoint('2026-05-24T08:00:00Z', '2026-05-25T08:00:00Z', 0, 0, 0, 0),
          createTrendPoint('2026-05-25T08:00:00Z', '2026-05-26T08:00:00Z', 0, 0, 0, 0),
          createTrendPoint('2026-05-26T08:00:00Z', '2026-05-27T08:00:00Z', 0, 0, 0, 0),
          createTrendPoint('2026-05-27T08:00:00Z', '2026-05-28T08:00:00Z', 0, 0, 0, 0),
        ],
      },
    });

    const wrapper = mountOverview();

    await flushPromises();

    expect(wrapper.text()).not.toContain('Not enough risk events yet');
    expect(wrapper.find('[data-audit-trend-chart="true"]').exists()).toBe(true);
    expect(chartMocks.init).toHaveBeenCalledTimes(1);
    expect(chartMocks.setOption).toHaveBeenCalled();

    const option = chartMocks.setOption.mock.calls.at(-1)?.[0];
    expect(option?.series?.[0]?.data).toEqual([0, 4, 0, 0, 0, 0, 0]);
    expect(option?.xAxis?.axisLabel?.formatter('2026-05-22T08:00:00Z-2026-05-23T08:00:00Z')).not.toBe('');
  });

  it('keeps the chart hidden when every bucket is empty', async () => {
    getAuditOverviewMock.mockResolvedValueOnce({
      ...createAuditOverviewResponse(),
      trend: {
        bucket_unit: 'hour',
        bucket_size: 1,
        points: [
          createTrendPoint('2026-05-27T08:00:00Z', '2026-05-27T09:00:00Z', 0, 0, 0, 0),
          createTrendPoint('2026-05-27T09:00:00Z', '2026-05-27T10:00:00Z', 0, 0, 0, 0),
          createTrendPoint('2026-05-27T10:00:00Z', '2026-05-27T11:00:00Z', 0, 0, 0, 0),
        ],
      },
    });

    const wrapper = mountOverview();

    await flushPromises();

    expect(wrapper.text()).toContain('Not enough risk events yet');
    expect(chartMocks.init).not.toHaveBeenCalled();
  });

  it('uses quantity-based y-axis instead of percentage labels', async () => {
    getAuditOverviewMock.mockResolvedValueOnce({
      ...createAuditOverviewResponse(),
      trend: {
        bucket_unit: 'hour',
        bucket_size: 1,
        points: [
          createTrendPoint('2026-05-27T08:00:00Z', '2026-05-27T09:00:00Z', 4, 1, 3, 1),
          createTrendPoint('2026-05-27T09:00:00Z', '2026-05-27T10:00:00Z', 12, 2, 7, 2),
          createTrendPoint('2026-05-27T10:00:00Z', '2026-05-27T11:00:00Z', 9, 1, 5, 1),
        ],
      },
    });

    mountOverview();

    await flushPromises();

    const option = chartMocks.setOption.mock.calls.at(-1)?.[0];
    expect(option?.yAxis?.axisLabel?.formatter(9)).toBe('9');
    expect(JSON.stringify(option)).not.toContain('%');
  });

  it('navigates security timeline items to audit incident and access-log request facts', async () => {
    const wrapper = mountOverview();

    await flushPromises();

    const eventButton = wrapper.findAll('button').find((item) => item.text().includes('Open Event'));
    expect(eventButton).toBeTruthy();
    await eventButton!.trigger('click');

    expect(routerMocks.push).toHaveBeenLastCalledWith({
      path: AUDIT_ROUTE_PATH.INCIDENT_DETAIL.replace(':event_id', '42'),
    });

    const timelineButton = wrapper.findAll('button').find((item) => item.text().includes('View Related Request'));

    expect(timelineButton).toBeTruthy();
    await timelineButton!.trigger('click');

    expect(routerMocks.push).toHaveBeenLastCalledWith({
      path: '/logs/access',
      query: {
        request_id: 'req-incident-42',
      },
    });
  });
});
