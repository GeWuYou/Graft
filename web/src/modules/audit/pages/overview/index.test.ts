import { flushPromises, mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';
import { createI18n } from 'vue-i18n';

import AuditOverviewPage from './index.vue';

const routerMocks = vi.hoisted(() => ({
  push: vi.fn(),
}));

const auditApiMocks = vi.hoisted(() => ({
  getAuditOverview: vi.fn(async () => ({
    window: '24h',
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
  })),
}));

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

vi.mock('../../components/AuditStats.vue', () => ({
  default: defineComponent({
    name: 'AuditStatsStub',
    props: {
      items: {
        type: Array,
        default: () => [],
      },
    },
    setup(props) {
      return () => h('div', { 'data-testid': 'audit-stats' }, JSON.stringify(props.items));
    },
  }),
}));

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: routerMocks.push,
  }),
}));

const passthroughStub = defineComponent({
  name: 'PassthroughStub',
  props: ['title', 'description', 'items'],
  setup(props, { slots }) {
    return () =>
      h('div', [props.title, props.description, JSON.stringify(props.items), slots.default?.(), slots.actions?.()]);
  },
});

const cardStub = defineComponent({
  name: 'TCardStub',
  setup(_, { slots }) {
    return () => h('section', [slots.header?.(), slots.default?.()]);
  },
});

const buttonStub = defineComponent({
  name: 'TButtonStub',
  emits: ['click'],
  setup(_, { emit, slots }) {
    return () => h('button', { onClick: () => emit('click') }, slots.default?.());
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
          title: 'Security Audit',
          description: 'Review recent anomalies, risk items, and common audit entry points.',
          refresh: 'Refresh',
          retry: 'Retry',
          errorTitle: 'Audit overview is temporarily unavailable',
          loadFailed: 'Failed to load audit overview',
          timeRanges: { '24h': 'Last 24h', '7d': 'Last 7d', '30d': 'Last 30d' },
          itemResult: { failed: 'Failed', denied: 'Denied', sensitive: 'Review' },
          sections: {
            failedAuth: 'Recent Failed Authentication',
            permissionDenied: 'Recent Permission Denied',
            sensitiveOps: 'Recent Sensitive Operations',
            shortcuts: 'Quick Links',
            riskWatch: 'Recent Risk',
          },
          stats: {
            totalLogs: { title: 'Audit Logs', unit: 'events', meta: 'window' },
            failedToday: { title: 'Failed Operations Today', unit: 'events', meta: 'watch' },
            highRisk: { title: 'High-Risk Events', unit: 'items', meta: 'failed' },
            sensitiveOps: { title: 'Sensitive Operations', unit: 'actions', meta: 'write' },
          },
          shortcuts: {
            failedAuth: { title: 'Open Failed Authentication', description: 'Apply the failed authentication preset' },
            rbacChanges: { title: 'Open RBAC Changes', description: 'Review role and permission records' },
            sensitiveOps: { title: 'Open Sensitive Operations', description: 'Locate privileged write actions' },
          },
        },
      },
    },
  },
});

describe('AuditOverviewPage', () => {
  it('renders the streamlined workbench overview and opens a quick link', async () => {
    const wrapper = mount(AuditOverviewPage, {
      global: {
        plugins: [i18n],
        stubs: {
          'management-page-content': passthroughStub,
          'management-page-header': passthroughStub,
          'management-empty-state': passthroughStub,
          't-card': cardStub,
          't-button': buttonStub,
          't-radio-group': radioGroupStub,
          't-radio-button': radioButtonStub,
          't-space': passthroughStub,
          't-tag': tagStub,
        },
      },
    });

    await flushPromises();

    expect(auditApiMocks.getAuditOverview).toHaveBeenCalledWith({ window: '24h' });
    expect(wrapper.attributes('data-page-type')).toBe('overview-dashboard');
    expect(wrapper.text()).toContain('Recent Failed Authentication');
    expect(wrapper.text()).toContain('Recent Permission Denied');
    expect(wrapper.text()).toContain('Recent Sensitive Operations');
    expect(wrapper.text()).toContain('Quick Links');
    expect(wrapper.text()).toContain('12');

    await wrapper.get('button[type="button"]').trigger('click');
    expect(routerMocks.push).toHaveBeenCalled();
  });
});
