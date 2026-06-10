// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import type { DashboardWidget } from '../../types/dashboard';
import AlertListWidget from './AlertListWidget.vue';

vi.mock('@/locales', () => ({
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
              route_location: '/audit/logs?preset=last_24h&scope=auth_failures',
              description: 'auth.token_expired',
              description_key: 'dashboard.widget.auditRiskEvents.authFailures.description',
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
    expect(wrapper.text()).toContain('权限拒绝');
    expect(wrapper.text()).toContain('过去 24 小时存在认证失败事件');
    expect(wrapper.text()).toContain('过去 24 小时存在权限拒绝事件');
    expect(wrapper.text()).not.toContain('Authentication Failures');
    expect(wrapper.text()).not.toContain('Permission Denials');
    expect(wrapper.text()).not.toContain('auth.token_expired');
    expect(wrapper.findAll('button')).toHaveLength(2);

    wrapper.findAllComponents(buttonStub)[1].vm.$emit('click');
    await wrapper.vm.$nextTick();

    expect(routerMocks.push).toHaveBeenCalledWith('/audit/logs?preset=last_24h&scope=permission_denials');
  });
});
