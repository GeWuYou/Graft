import { mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import type { DashboardWidget } from '../../types/dashboard';
import AlertListWidget from './AlertListWidget.vue';

vi.mock('@/locales', () => ({
  currentLocale: 'en-US',
  t: (key: string, params?: Record<string, unknown>) => {
    const translations: Record<string, string> = {
      'dashboard.actions.open': '打开',
      'dashboard.alert.count': `${params?.count ?? 0}次`,
      'dashboard.alert.latestAt': `最近时间：${params?.time ?? ''}`,
      'dashboard.alert.level.error': '错误',
      'dashboard.alert.level.info': '信息',
      'dashboard.alert.level.warning': '警告',
      'audit.overview.riskGroups.authFailures': '认证失败',
      'audit.overview.riskGroups.permissionDenials': '权限拒绝',
      'dashboard.widget.auditRiskEvents.authFailures.action': '查看认证失败',
      'dashboard.widget.auditRiskEvents.authFailures.description': '过去 24 小时存在认证失败事件',
      'dashboard.widget.auditRiskEvents.permissionDenials.description': '过去 24 小时存在权限拒绝事件',
      'dashboard.widget.empty': '暂无组件数据',
      'dashboard.widget.invalidPayload': '组件数据格式不可用',
    };
    return translations[key] ?? key;
  },
}));

const routerMocks = vi.hoisted(() => ({
  push: vi.fn(),
}));

vi.mock('vue-router', () => ({
  useRouter: () => routerMocks,
}));

beforeEach(() => {
  routerMocks.push.mockClear();
});

const listStub = defineComponent({
  name: 'TListStub',
  setup(_props, { slots }) {
    return () => h('div', slots.default?.());
  },
});

const listItemStub = defineComponent({
  name: 'TListItemStub',
  setup(_props, { slots }) {
    return () => h('div', [slots.default?.(), slots.action?.()]);
  },
});

const buttonStub = defineComponent({
  name: 'TButtonStub',
  emits: ['click'],
  setup(_props, { emit, slots }) {
    return () => h('button', { onClick: (event: MouseEvent) => emit('click', event) }, slots.default?.());
  },
});

const passthroughStub = defineComponent({
  name: 'PassthroughStub',
  props: {
    description: {
      type: String,
      default: '',
    },
  },
  setup(props, { slots }) {
    return () => h('div', [props.description, slots.default?.()]);
  },
});

function alertWidget(payload: DashboardWidget['payload']): DashboardWidget {
  return {
    category: 'security',
    id: 'audit.risk-events',
    module_key: 'audit',
    order: 1,
    payload,
    priority: 'warning',
    size: 'medium',
    state: 'warning',
    title: 'Audit Risk Events',
    type: 'alert-list',
    visible: true,
  };
}

describe('AlertListWidget', () => {
  it('renders backend-provided counts and routes without merging duplicate titles', async () => {
    const wrapper = mount(AlertListWidget, {
      props: {
        widget: alertWidget({
          items: [
            {
              count: 4,
              id: 'audit.auth-failures',
              level: 'warning',
              route_location: '/audit/logs?preset=last_24h&business_category=auth_failures',
              action_label: 'View authentication failures',
              action_label_key: 'dashboard.widget.auditRiskEvents.authFailures.action',
              description: 'auth.token_expired',
              description_key: 'dashboard.widget.auditRiskEvents.authFailures.description',
              occurred_at: '2026-06-10T02:38:00Z',
              title: 'Authentication failures',
              title_key: 'audit.overview.riskGroups.authFailures',
            },
            {
              count: 2,
              id: 'audit.permission-denials',
              level: 'warning',
              route_location: '/audit/logs?preset=last_24h&scope=permission_denials',
              description: 'auth.token_expired',
              description_key: 'dashboard.widget.auditRiskEvents.permissionDenials.description',
              title: 'Permission denials',
              title_key: 'audit.overview.riskGroups.permissionDenials',
            },
          ],
        }),
      },
      global: {
        stubs: {
          TButton: buttonStub,
          TEmpty: passthroughStub,
          TList: listStub,
          TListItem: listItemStub,
          TTag: passthroughStub,
        },
      },
    });

    expect(wrapper.text()).toContain('4次');
    expect(wrapper.text()).toContain('2次');
    expect(wrapper.text()).toContain('认证失败');
    expect(wrapper.text()).toContain(
      new Intl.DateTimeFormat('en-US', {
        dateStyle: 'medium',
        timeStyle: 'short',
      }).format(new Date('2026-06-10T02:38:00Z')),
    );
    expect(wrapper.text()).toContain('权限拒绝');
    expect(wrapper.text()).toContain('过去 24 小时存在认证失败事件');
    expect(wrapper.text()).toContain('过去 24 小时存在权限拒绝事件');
    expect(wrapper.text()).toContain('查看认证失败');
    expect(wrapper.text()).toContain('打开');
    expect(wrapper.text()).not.toContain('Authentication Failures');
    expect(wrapper.text()).not.toContain('Permission Denials');
    expect(wrapper.text()).not.toContain('auth.token_expired');
    expect(wrapper.findAll('button')).toHaveLength(2);

    wrapper.findAllComponents(buttonStub)[1].vm.$emit('click');
    await wrapper.vm.$nextTick();

    expect(routerMocks.push).toHaveBeenCalledWith('/audit/logs?preset=last_24h&scope=permission_denials');
  });

  it('opens explicit audit-log filters from security dashboard entries', async () => {
    const wrapper = mount(AlertListWidget, {
      props: {
        widget: alertWidget({
          items: [
            {
              action_label: 'View events',
              action_label_key: 'dashboard.widget.auditRiskEvents.highRisk.action',
              count: 3,
              id: 'audit.high-risk',
              level: 'error',
              route_location: '/audit/logs?preset=last_24h&risk_levels=HIGH%2CCRITICAL',
              title: 'High-risk audit events',
              title_key: 'dashboard.widget.auditRiskEvents.highRisk.title',
            },
            {
              action_label: 'View failures',
              action_label_key: 'dashboard.widget.auditRiskEvents.failedOperations.action',
              count: 5,
              id: 'audit.failed-operations',
              level: 'warning',
              route_location: '/audit/logs?preset=last_24h&results=FAILED%2CDENIED%2CERROR',
              title: 'Failed operations',
              title_key: 'dashboard.widget.auditRiskEvents.failedOperations.title',
            },
          ],
        }),
      },
      global: {
        stubs: {
          TButton: buttonStub,
          TEmpty: passthroughStub,
          TList: listStub,
          TListItem: listItemStub,
          TTag: passthroughStub,
        },
      },
    });

    expect(wrapper.text()).toContain('View events');
    expect(wrapper.text()).toContain('View failures');

    wrapper.findAllComponents(buttonStub)[0].vm.$emit('click');
    await wrapper.vm.$nextTick();
    wrapper.findAllComponents(buttonStub)[1].vm.$emit('click');
    await wrapper.vm.$nextTick();

    expect(routerMocks.push).toHaveBeenNthCalledWith(1, '/audit/logs?preset=last_24h&risk_levels=HIGH%2CCRITICAL');
    expect(routerMocks.push).toHaveBeenNthCalledWith(2, '/audit/logs?preset=last_24h&results=FAILED%2CDENIED%2CERROR');
  });
});
