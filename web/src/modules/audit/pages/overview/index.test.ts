import { flushPromises, mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';
import { createI18n } from 'vue-i18n';

import { AUDIT_ROUTE_PATH } from '../../contract/paths';
import type { AuditOverviewResponse } from '../../types/audit';
import AuditOverviewPage from './index.vue';

const routerMocks = vi.hoisted(() => ({
  push: vi.fn(),
}));

const auditApiMocks = vi.hoisted(() => ({
  getAuditOverview: vi.fn(
    async (): Promise<AuditOverviewResponse> => ({
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
    }),
  ),
}));

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

vi.mock('../../api/audit', () => ({
  getAuditOverview: auditApiMocks.getAuditOverview,
}));

vi.mock('@/modules/shared/localized-api-error', () => ({
  resolveLocalizedErrorMessage: () => 'load failed',
}));

vi.mock('@/utils/logger', () => ({
  createLogger: () => ({
    error: vi.fn(),
  }),
}));

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: routerMocks.push,
  }),
}));

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

const tagStub = defineComponent({
  name: 'TTagStub',
  setup(_, { slots }) {
    return () => h('span', slots.default?.());
  },
});

const i18n = createI18n({
  legacy: false,
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
            legend: {
              total: 'Total',
              highRisk: 'High risk',
              security: 'Security events',
            },
            totalValue: 'Total {value}',
            highRiskValue: 'High risk: {value}',
            securityValue: 'Security events: {value}',
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
        common: {
          risk: { HIGH: 'High', CRITICAL: 'Critical' },
          source: { SECURITY_EVENT: 'Security Event' },
        },
      },
    },
  },
});

describe('AuditOverviewPage', () => {
  it('renders the streamlined workbench overview and opens a quick link with canonical filter keys', async () => {
    const wrapper = mount(AuditOverviewPage, {
      global: {
        plugins: [i18n],
        stubs: {
          'governance-dashboard-shell': passthroughStub,
          'governance-section': passthroughStub,
          'governance-summary-card': passthroughStub,
          'management-empty-state': passthroughStub,
          't-button': buttonStub,
          't-radio-group': radioGroupStub,
          't-radio-button': radioButtonStub,
          't-space': passthroughStub,
          't-tag': tagStub,
          't-timeline': passthroughStub,
          't-timeline-item': passthroughStub,
        },
      },
    });

    await flushPromises();

    expect(auditApiMocks.getAuditOverview).toHaveBeenCalledWith({ preset: 'last_24h' });
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

    const wrapper = mount(AuditOverviewPage, {
      global: {
        plugins: [i18n],
        stubs: {
          'governance-dashboard-shell': passthroughStub,
          'governance-section': passthroughStub,
          'governance-summary-card': passthroughStub,
          'management-empty-state': passthroughStub,
          't-button': buttonStub,
          't-radio-group': radioGroupStub,
          't-radio-button': radioButtonStub,
          't-space': passthroughStub,
          't-tag': tagStub,
          't-timeline': passthroughStub,
          't-timeline-item': passthroughStub,
        },
      },
    });

    await flushPromises();

    await wrapper.findAll('button[type="button"]')[1]!.trigger('click');

    expect(routerMocks.push).toHaveBeenCalledWith({
      path: AUDIT_ROUTE_PATH.LOGS,
      query: expect.objectContaining({
        success: 'false',
        created_from: expect.any(String),
        created_to: expect.any(String),
      }),
    });
  });

  it('opens the high-risk summary card with canonical summary query params', async () => {
    routerMocks.push.mockClear();

    const wrapper = mount(AuditOverviewPage, {
      global: {
        plugins: [i18n],
        stubs: {
          'governance-dashboard-shell': passthroughStub,
          'governance-section': passthroughStub,
          'governance-summary-card': passthroughStub,
          'management-empty-state': passthroughStub,
          't-button': buttonStub,
          't-radio-group': radioGroupStub,
          't-radio-button': radioButtonStub,
          't-space': passthroughStub,
          't-tag': tagStub,
          't-timeline': passthroughStub,
          't-timeline-item': passthroughStub,
        },
      },
    });

    await flushPromises();

    await wrapper.findAll('button[type="button"]')[2]!.trigger('click');

    expect(routerMocks.push).toHaveBeenCalledWith({
      path: AUDIT_ROUTE_PATH.LOGS,
      query: expect.objectContaining({
        risk_levels: 'HIGH,CRITICAL',
        created_from: expect.any(String),
        created_to: expect.any(String),
      }),
    });
  });

  it('opens risk groups with canonical visible audit filters', async () => {
    routerMocks.push.mockClear();
    auditApiMocks.getAuditOverview.mockResolvedValueOnce({
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

    const wrapper = mount(AuditOverviewPage, {
      global: {
        plugins: [i18n],
        stubs: {
          'governance-dashboard-shell': passthroughStub,
          'governance-section': passthroughStub,
          'governance-summary-card': passthroughStub,
          'management-empty-state': passthroughStub,
          't-button': buttonStub,
          't-radio-group': radioGroupStub,
          't-radio-button': radioButtonStub,
          't-space': passthroughStub,
          't-tag': tagStub,
          't-timeline': passthroughStub,
          't-timeline-item': passthroughStub,
        },
      },
    });

    await flushPromises();

    const riskGroupButton = wrapper.findAll('button').find((item) => item.text().includes('View related events'));

    expect(riskGroupButton).toBeTruthy();
    await riskGroupButton!.trigger('click');

    expect(routerMocks.push).toHaveBeenCalledWith({
      path: AUDIT_ROUTE_PATH.LOGS,
      query: expect.objectContaining({
        risk_levels: 'HIGH,CRITICAL',
        created_from: expect.any(String),
        created_to: expect.any(String),
      }),
    });
  });

  it('opens sensitive summary with the same keyword scope used by overview counters', async () => {
    routerMocks.push.mockClear();

    const wrapper = mount(AuditOverviewPage, {
      global: {
        plugins: [i18n],
        stubs: {
          'governance-dashboard-shell': passthroughStub,
          'governance-section': passthroughStub,
          'governance-summary-card': passthroughStub,
          'management-empty-state': passthroughStub,
          't-button': buttonStub,
          't-radio-group': radioGroupStub,
          't-radio-button': radioButtonStub,
          't-space': passthroughStub,
          't-tag': tagStub,
          't-timeline': passthroughStub,
          't-timeline-item': passthroughStub,
        },
      },
    });

    await flushPromises();

    await wrapper.findAll('button[type="button"]')[3]!.trigger('click');

    expect(routerMocks.push).toHaveBeenCalledWith({
      path: AUDIT_ROUTE_PATH.LOGS,
      query: expect.objectContaining({
        action_keywords: 'delete,reset,grant,assign,revoke,remove,replace',
        created_from: expect.any(String),
        created_to: expect.any(String),
      }),
    });
  });

  it('uses the updated overview stat labels', async () => {
    const wrapper = mount(AuditOverviewPage, {
      global: {
        plugins: [i18n],
        stubs: {
          'governance-dashboard-shell': passthroughStub,
          'governance-section': passthroughStub,
          'governance-summary-card': passthroughStub,
          'management-empty-state': passthroughStub,
          't-button': buttonStub,
          't-radio-group': radioGroupStub,
          't-radio-button': radioButtonStub,
          't-space': passthroughStub,
          't-tag': tagStub,
          't-timeline': passthroughStub,
          't-timeline-item': passthroughStub,
        },
      },
    });

    await flushPromises();

    const summaryCards = wrapper
      .findAllComponents({ name: 'PassthroughStub' })
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

    auditApiMocks.getAuditOverview.mockResolvedValueOnce(overviewWithTrend);

    const wrapper = mount(AuditOverviewPage, {
      global: {
        plugins: [i18n],
        stubs: {
          'governance-dashboard-shell': passthroughStub,
          'governance-section': passthroughStub,
          'governance-summary-card': passthroughStub,
          'management-empty-state': passthroughStub,
          't-button': buttonStub,
          't-radio-group': radioGroupStub,
          't-radio-button': radioButtonStub,
          't-space': passthroughStub,
          't-tag': tagStub,
          't-timeline': passthroughStub,
          't-timeline-item': passthroughStub,
          't-tooltip': passthroughStub,
        },
      },
    });

    await flushPromises();

    expect(wrapper.text()).not.toContain('Not enough risk events yet');
    expect(wrapper.text()).toContain('Total');
    expect(wrapper.text()).toContain('High risk');
    expect(wrapper.text()).toContain('Security events');
    expect(wrapper.findAll('.audit-overview__trend-point')).toHaveLength(3);
    expect(wrapper.text()).toContain('7');
    expect(wrapper.text()).toContain('0');
  });

  it('navigates security timeline items with the current request CTA-only interaction', async () => {
    const wrapper = mount(AuditOverviewPage, {
      global: {
        plugins: [i18n],
        stubs: {
          'governance-dashboard-shell': passthroughStub,
          'governance-section': passthroughStub,
          'governance-summary-card': passthroughStub,
          'management-empty-state': passthroughStub,
          't-button': buttonStub,
          't-radio-group': radioGroupStub,
          't-radio-button': radioButtonStub,
          't-space': passthroughStub,
          't-tag': tagStub,
          't-timeline': passthroughStub,
          't-timeline-item': passthroughStub,
        },
      },
    });

    await flushPromises();

    expect(wrapper.text()).not.toContain('Open Incident');
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
