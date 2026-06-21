import { mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import type { DashboardQuickActionLink } from '../contract/quick-action-links';
import type { DashboardQuickActionConfig } from '../contract/quick-actions';
import DashboardQuickActions from './DashboardQuickActions.vue';

const localeCallKeys: string[] = [];

vi.mock('@/locales', () => ({
  t: (key: string, params?: Record<string, unknown>) => {
    localeCallKeys.push(key);

    const translations: Record<string, string> = {
      'dashboard.module.audit': 'Security Audit',
      'dashboard.module.core': 'Service Management',
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

const routerMocks = {
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
};

vi.mock('vue-router', () => ({
  useRouter: () => routerMocks,
}));

const passthroughStub = defineComponent({
  name: 'PassthroughStub',
  props: {
    description: {
      type: String,
      default: '',
    },
  },
  setup(props, { slots }) {
    return () => h('div', [props.description, slots.title?.(), slots.default?.(), slots.actions?.(), slots.icon?.()]);
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

function quickLink(index: number, partial: Partial<DashboardQuickActionLink> = {}): DashboardQuickActionLink {
  return {
    id: `link-${index}`,
    module_key: index % 2 === 0 ? 'core' : 'audit',
    group: index % 2 === 0 ? 'Service Management' : 'Security Audit',
    order: index,
    route_location: `/route-${index}`,
    title: `Link ${index}`,
    full_label: `${index % 2 === 0 ? 'Service Management' : 'Security Audit'} - Link ${index}`,
    ...partial,
  };
}

function mountQuickActions(links: DashboardQuickActionLink[], config?: DashboardQuickActionConfig) {
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
        TTooltip: passthroughStub,
      },
    },
  });
}

describe('DashboardQuickActions', () => {
  beforeEach(() => {
    localeCallKeys.length = 0;
    routerMocks.push.mockReset();
    localStorage.clear();
  });

  it('shows configured links by default and exposes a drawer affordance without expanding the home grid', async () => {
    const wrapper = mountQuickActions(Array.from({ length: 10 }, (_, index) => quickLink(index + 1)));

    expect(wrapper.findAll('.dashboard-quick-actions__item')).toHaveLength(4);
    expect(wrapper.text()).toContain('查看全部 10 个');
    expect(wrapper.text()).toContain('Security Audit');
    expect(wrapper.text()).toContain('Service Management');
    expect(wrapper.text()).not.toContain('Link 5');

    await wrapper.findAll('button').at(-1)?.trigger('click');

    expect(wrapper.findAll('.dashboard-quick-actions__item')).toHaveLength(14);
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
          TDrawer: drawerStub,
          TEmpty: passthroughStub,
          TIcon: passthroughStub,
          TTooltip: passthroughStub,
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

  it('renders split title and group labels while preserving the full label in card title', async () => {
    const wrapper = mountQuickActions(
      [
        quickLink(1, {
          route_location: '/server/overview',
          title: 'Overview',
          group: 'Service Management',
          full_label: 'Service Management - Overview',
        }),
        quickLink(2, {
          route_location: '/access-control/users',
          title: 'User Management',
          group: 'Access Control',
          full_label: 'Access Control - User Management',
        }),
        quickLink(3, {
          route_location: '/unknown/overview',
          title: 'Overview',
          group: 'Unknown',
          full_label: 'Overview',
        }),
      ],
      { enabled: true, maxItems: 2, strategy: 'hybrid' },
    );

    const titles = wrapper.findAll('.dashboard-quick-actions__item strong').map((item) => item.text());
    const groups = wrapper.findAll('.dashboard-quick-actions__item small').map((item) => item.text());
    const fullLabels = wrapper.findAll('.dashboard-quick-actions__item').map((item) => item.attributes('title'));

    expect(titles).toEqual(['Overview', 'User Management']);
    expect(groups).toEqual(['Service Management', 'Access Control']);
    expect(fullLabels).toEqual(['Service Management - Overview', 'Access Control - User Management']);

    await wrapper.findAll('button').at(-1)?.trigger('click');

    const drawerTitles = wrapper.findAll('.dashboard-quick-actions__item--drawer strong').map((item) => item.text());
    expect(drawerTitles).toEqual(['Overview', 'User Management', 'Overview']);
  });

  it('falls back to module label when group is missing', () => {
    const wrapper = mountQuickActions([
      quickLink(1, {
        module_key: 'audit',
        group: undefined,
        title: 'Audit Logs',
        full_label: 'Security Audit - Audit Logs',
      }),
    ]);

    expect(wrapper.find('.dashboard-quick-actions__item small').text()).toBe('Security Audit');
  });

  it('does not resolve module fallback keys when group text is already available', () => {
    mountQuickActions([
      quickLink(1, {
        module_key: 'container',
        group: '运维管理',
        title: '容器管理',
        full_label: '运维管理 - 容器管理',
      }),
    ]);

    expect(localeCallKeys).not.toContain('dashboard.module.container');
  });
});
