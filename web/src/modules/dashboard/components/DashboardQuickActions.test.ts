// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import type { DashboardQuickActionConfig } from '../contract/quick-actions';
import type { DashboardQuickLink } from '../types/dashboard';
import DashboardQuickActions from './DashboardQuickActions.vue';

vi.mock('@/locales', () => ({
  t: (key: string, params?: Record<string, unknown>) => {
    const translations: Record<string, string> = {
      'dashboard.module.audit': '审计',
      'dashboard.module.core': '核心',
      'dashboard.quickActions.description': '当前权限可用入口',
      'dashboard.quickActions.drawerTitle': '全部快捷入口',
      'dashboard.quickActions.empty': '暂无可用快捷入口',
      'dashboard.quickActions.title': '快捷操作',
      'dashboard.quickActions.viewAll': `查看全部 ${params?.count ?? 0} 个`,
    };
    return translations[key] ?? key;
  },
}));

vi.mock('@/locales/useLocale', () => ({
  useLocale: () => ({
    locale: { value: 'zh-CN' },
  }),
}));

vi.mock('@/modules', () => ({
  getBootstrapRouteRegistration: (menuPath: string) => {
    const registrations = new Map([
      [
        '/server/overview',
        {
          meta: {
            tabTitle: {
              'zh-CN': '服务管理 - 概览',
              'en-US': 'Service Management - Overview',
            },
          },
        },
      ],
    ]);

    return registrations.get(menuPath);
  },
}));

const routerMocks = vi.hoisted(() => ({
  getRoutes: vi.fn(() => [
    {
      path: '/access-control',
      children: [
        {
          path: 'users',
          meta: {
            tabTitle: {
              'zh-CN': '访问控制 - 用户管理',
              'en-US': 'Access Control - User Management',
            },
          },
        },
      ],
    },
    {
      path: '/logs',
      children: [
        {
          path: 'access',
          meta: {
            tabTitle: {
              'zh-CN': '日志中心 - 访问日志',
              'en-US': 'Log Center - Access Logs',
            },
          },
        },
        {
          path: 'app',
          meta: {
            tabTitle: {
              'zh-CN': '日志中心 - 应用日志',
              'en-US': 'Log Center - App Logs',
            },
          },
        },
      ],
    },
  ]),
  push: vi.fn(),
}));

vi.mock('vue-router', () => ({
  useRouter: () => routerMocks,
}));

const passthroughStub = defineComponent({
  name: 'PassthroughStub',
  props: {
    color: {
      type: String,
      default: '',
    },
    content: {
      type: String,
      default: '',
    },
    description: {
      type: String,
      default: '',
    },
  },
  setup(props, { slots }) {
    return () =>
      h('div', { 'data-color': props.color }, [
        props.content,
        props.description,
        slots.title?.(),
        slots.default?.(),
        slots.actions?.(),
        slots.icon?.(),
      ]);
  },
});
const buttonStub = defineComponent({
  name: 'TButtonStub',
  emits: ['click'],
  setup(_props, { attrs, emit, slots }) {
    return () => h('button', { ...attrs, onClick: (event: MouseEvent) => emit('click', event) }, slots.default?.());
  },
});

const drawerStub = defineComponent({
  name: 'TDrawerStub',
  props: {
    header: {
      type: String,
      default: '',
    },
    visible: {
      type: Boolean,
      default: false,
    },
  },
  setup(props, { slots }) {
    return () => (props.visible ? h('div', [props.header, slots.default?.()]) : null);
  },
});

function quickLink(index: number, partial: Partial<DashboardQuickLink> = {}): DashboardQuickLink {
  return {
    id: `link-${index}`,
    module_key: index % 2 === 0 ? 'core' : 'audit',
    order: index,
    route_location: `/route-${index}`,
    title: `Link ${index}`,
    ...partial,
  };
}

function mountQuickActions(links: DashboardQuickLink[], config?: DashboardQuickActionConfig) {
  return mount(DashboardQuickActions, {
    props: {
      config,
      links,
    },
    global: {
      stubs: {
        TButton: buttonStub,
        TCard: passthroughStub,
        TBadge: passthroughStub,
        TDrawer: drawerStub,
        TEmpty: passthroughStub,
        TIcon: passthroughStub,
        TTag: passthroughStub,
      },
    },
  });
}

describe('DashboardQuickActions', () => {
  beforeEach(() => {
    routerMocks.push.mockReset();
    localStorage.clear();
  });

  it('shows configured links by default and exposes a drawer affordance without expanding the home grid', async () => {
    const wrapper = mountQuickActions(Array.from({ length: 10 }, (_, index) => quickLink(index + 1)));

    expect(wrapper.findAll('.dashboard-quick-actions__item')).toHaveLength(8);
    expect(wrapper.text()).toContain('查看全部 10 个');
    expect(wrapper.text()).toContain('审计');
    expect(wrapper.text()).toContain('核心');
    expect(wrapper.text()).not.toContain('Link 10');

    await wrapper.findAll('button').at(-1)?.trigger('click');

    expect(wrapper.findAll('.dashboard-quick-actions__item')).toHaveLength(18);
    expect(wrapper.text()).toContain('全部快捷入口');
    expect(wrapper.text()).toContain('Link 10');
  });

  it('ranks links by stored usage under most-used strategy', () => {
    localStorage.setItem(
      'dashboard:quick-actions:route-usage',
      JSON.stringify({
        '/route-3': { accessCount: 20, lastAccessAt: '2026-06-09T10:00:00.000Z' },
        '/route-1': { accessCount: 3, lastAccessAt: '2026-06-09T12:00:00.000Z' },
      }),
    );

    const wrapper = mount(DashboardQuickActions, {
      props: {
        config: { enabled: true, maxItems: 2, strategy: 'most_used' },
        links: [quickLink(1), quickLink(2), quickLink(3)],
      },
      global: {
        stubs: {
          TButton: buttonStub,
          TCard: passthroughStub,
          TBadge: passthroughStub,
          TDrawer: drawerStub,
          TEmpty: passthroughStub,
          TIcon: passthroughStub,
          TTag: passthroughStub,
        },
      },
    });

    const titles = wrapper.findAll('.dashboard-quick-actions__item strong').map((item) => item.text());
    expect(titles.slice(0, 2)).toEqual(['Link 3', 'Link 1']);
  });

  it('opens the selected backend-provided route', async () => {
    const wrapper = mountQuickActions([quickLink(1, { route_location: '/audit/events' })]);

    await wrapper.find('.dashboard-quick-actions__item').trigger('click');

    expect(routerMocks.push).toHaveBeenCalledWith('/audit/events');
  });

  it('uses route tab titles so overview quick links include their first-level menu context', async () => {
    const wrapper = mountQuickActions(
      [
        quickLink(1, { route_location: '/server/overview', title: '概览' }),
        quickLink(2, { route_location: '/access-control/users', title: '用户管理' }),
        quickLink(3, { route_location: '/unknown/overview', title: '概览' }),
      ],
      { enabled: true, maxItems: 2, strategy: 'hybrid' },
    );

    const titles = wrapper.findAll('.dashboard-quick-actions__item strong').map((item) => item.text());
    expect(titles).toEqual(['服务管理 - 概览', '访问控制 - 用户管理']);

    await wrapper.findAll('button').at(-1)?.trigger('click');

    const drawerTitles = wrapper.findAll('.dashboard-quick-actions__item--drawer strong').map((item) => item.text());
    expect(drawerTitles).toEqual(['服务管理 - 概览', '访问控制 - 用户管理', '概览']);
  });

  it('uses runtime menu-derived titles for log quick links without local title mappings', () => {
    const wrapper = mountQuickActions([
      quickLink(1, { route_location: '/logs/access', title: '访问日志' }),
      quickLink(2, { route_location: '/logs/app', title: '应用日志' }),
    ]);

    const titles = wrapper.findAll('.dashboard-quick-actions__item strong').map((item) => item.text());
    expect(titles).toEqual(['日志中心 - 访问日志', '日志中心 - 应用日志']);
  });

  it('colors only canonical module keys and dotted descendants', () => {
    const wrapper = mountQuickActions([
      quickLink(1, { module_key: 'audit' }),
      quickLink(2, { module_key: 'audit.events' }),
      quickLink(3, { module_key: 'not-audit' }),
    ]);

    const colors = wrapper.findAll('.dashboard-quick-actions__badge').map((badge) => badge.attributes('data-color'));

    expect(colors).toEqual(['var(--td-error-color-6)', 'var(--td-error-color-6)', 'var(--td-text-color-secondary)']);
  });
});
