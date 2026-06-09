import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import type { DashboardQuickLink } from '../types/dashboard';
import DashboardQuickActions from './DashboardQuickActions.vue';

vi.mock('@/locales', () => ({
  t: (key: string, params?: Record<string, unknown>) => {
    const translations: Record<string, string> = {
      'dashboard.module.audit': '审计',
      'dashboard.module.core': '核心',
      'dashboard.quickActions.description': '当前权限可用入口',
      'dashboard.quickActions.empty': '暂无可用快捷入口',
      'dashboard.quickActions.title': '快捷操作',
      'dashboard.quickActions.viewAll': `查看全部 ${params?.count ?? 0} 个`,
      'dashboard.quickActions.viewLess': '收起',
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

function mountQuickActions(links: DashboardQuickLink[]) {
  return mount(DashboardQuickActions, {
    props: {
      links,
    },
    global: {
      stubs: {
        TButton: buttonStub,
        TCard: passthroughStub,
        TEmpty: passthroughStub,
        TIcon: passthroughStub,
        TTag: passthroughStub,
      },
    },
  });
}

describe('DashboardQuickActions', () => {
  it('shows the first eight links by default and exposes a localized view-all affordance', async () => {
    const wrapper = mountQuickActions(Array.from({ length: 10 }, (_, index) => quickLink(index + 1)));

    expect(wrapper.findAll('.dashboard-quick-actions__item')).toHaveLength(8);
    expect(wrapper.text()).toContain('查看全部 2 个');
    expect(wrapper.text()).toContain('审计');
    expect(wrapper.text()).toContain('核心');
    expect(wrapper.text()).not.toContain('Link 10');

    await wrapper.findAll('button').at(-1)?.trigger('click');

    expect(wrapper.findAll('.dashboard-quick-actions__item')).toHaveLength(10);
    expect(wrapper.text()).toContain('收起');
    expect(wrapper.text()).toContain('Link 10');
  });

  it('opens the selected backend-provided route', async () => {
    const wrapper = mountQuickActions([quickLink(1, { route_location: '/audit/events' })]);

    await wrapper.find('.dashboard-quick-actions__item').trigger('click');

    expect(routerMocks.push).toHaveBeenCalledWith('/audit/events');
  });
});
