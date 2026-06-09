import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import type { DashboardWidget } from '../types/dashboard';
import DashboardRenderer from './DashboardRenderer.vue';

vi.mock('@/locales', () => ({
  t: (key: string, params?: Record<string, unknown>) => {
    const translations: Record<string, string> = {
      'dashboard.actions.details': '查看详情',
      'dashboard.actions.retry': 'Retry',
      'dashboard.category.count': `${params?.count ?? 0} widgets`,
      'dashboard.category.business': 'Business',
      'dashboard.category.operation': 'Operations',
      'dashboard.category.security': 'Security',
      'dashboard.category.system': 'System',
      'dashboard.health.summaryHealthy': 'Health checks passed',
      'dashboard.health.summaryHealthyWithCounts': `${params?.healthy ?? 0} modules running, ${
        params?.attention ?? 0
      } modules need attention`,
      'dashboard.health.healthy': 'Healthy',
      'dashboard.module.core': 'Core',
      'dashboard.module.audit': 'Audit',
      'dashboard.widget.disabledDescription': 'Disabled widget',
      'dashboard.widget.empty': 'No widgets',
      'dashboard.widget.errorFallback': 'Failed',
      'dashboard.widget.errorTitle': 'Widget failed',
      'dashboard.widget.state.critical': 'Critical',
      'dashboard.widget.state.warning': 'Attention',
      'dashboard.widget.status.disabled': 'Disabled',
      'dashboard.widget.status.error': 'Error',
      'dashboard.widget.status.normal': 'Normal',
      'dashboard.widget.status.warning': 'Warning',
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
    title: {
      type: String,
      default: '',
    },
    message: {
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
      h('div', [props.title, props.message, props.description, slots.title?.(), slots.default?.(), slots.actions?.()]);
  },
});

const buttonStub = defineComponent({
  name: 'TButtonStub',
  emits: ['click'],
  setup(_props, { emit, slots }) {
    return () => h('button', { onClick: (event: MouseEvent) => emit('click', event) }, slots.default?.());
  },
});

function baseWidget(partial: Partial<DashboardWidget>): DashboardWidget {
  return {
    id: 'core.module-runtime-health',
    module_key: 'core',
    category: 'system',
    order: 10,
    payload: {
      summary: {
        status: 'healthy',
      },
      items: [],
    },
    priority: 'normal',
    size: 'medium',
    state: 'normal',
    title: 'Module Health',
    type: 'health',
    visible: true,
    ...partial,
  };
}

function mountRenderer(widgets: DashboardWidget[]) {
  return mount(DashboardRenderer, {
    props: {
      widgets,
    },
    global: {
      stubs: {
        TAlert: passthroughStub,
        TButton: buttonStub,
        TCard: passthroughStub,
        TEmpty: passthroughStub,
        TList: passthroughStub,
        TListItem: passthroughStub,
        TSkeleton: passthroughStub,
        TTag: passthroughStub,
        TTimeline: passthroughStub,
        TTimelineItem: passthroughStub,
        't-skeleton': passthroughStub,
      },
    },
  });
}

describe('DashboardRenderer', () => {
  it('sorts widgets by order and renders by widget type only', () => {
    const wrapper = mountRenderer([
      baseWidget({
        id: 'core.recent-events',
        order: 20,
        payload: { items: [], empty: 'No events' },
        title: 'Recent Events',
        type: 'timeline',
      }),
      baseWidget({
        id: 'core.module-runtime-health',
        module_key: 'core',
        order: 5,
        title: 'Module Health',
        type: 'health',
      }),
    ]);

    const titles = wrapper.findAll('.dashboard-renderer__heading span').map((element) => element.text());
    expect(titles).toEqual(['Module Health', 'Recent Events']);
    expect(titles).not.toContain('');
    const text = wrapper.text();
    expect(text).toContain('No events');
  });

  it('groups visible widgets and moves critical priority groups first', () => {
    const wrapper = mountRenderer([
      baseWidget({
        id: 'monitor.system-health',
        order: 1,
        title: 'System Health',
        priority: 'info',
      }),
      baseWidget({
        id: 'audit.risk-events',
        module_key: 'audit',
        category: 'security',
        order: 100,
        payload: { items: [], empty: 'No events' },
        priority: 'critical',
        state: 'critical',
        title: 'Audit Risk Events',
        type: 'timeline',
      }),
      baseWidget({
        id: 'scheduler.empty',
        category: 'operation',
        priority: 'warning',
        state: 'hidden',
        title: 'Hidden Scheduler',
        visible: false,
      }),
    ]);

    const headings = wrapper.findAll('.dashboard-renderer__category-header h2').map((element) => element.text());
    expect(headings).toEqual(['Security', 'System']);
    expect(wrapper.text()).toContain('Critical');
    expect(wrapper.text()).not.toContain('Hidden Scheduler');
  });

  it('keeps an error widget visible and emits a focused refresh request', async () => {
    const wrapper = mountRenderer([
      baseWidget({
        error: {
          code: 'LOAD_FAILED',
          message: 'Widget unavailable',
        },
        status: 'error',
      }),
    ]);

    expect(wrapper.text()).toContain('Widget failed');
    expect(wrapper.text()).toContain('Widget unavailable');

    await wrapper.find('button').trigger('click');

    expect(wrapper.emitted('refresh-widget')?.[0]).toEqual(['core.module-runtime-health']);
  });

  it('renders the empty state when no widgets are visible', () => {
    const wrapper = mountRenderer([]);

    expect(wrapper.text()).toContain('No widgets');
  });

  it('renders framework-level action buttons with consistent route handling', async () => {
    const wrapper = mountRenderer([
      baseWidget({
        action: {
          label: 'View details',
          label_key: 'dashboard.actions.details',
          route: '/server/modules',
        },
      }),
    ]);

    await wrapper.find('button').trigger('click');

    expect(routerMocks.push).toHaveBeenCalledWith('/server/modules');
    expect(wrapper.text()).toContain('查看详情');
    expect(wrapper.text()).not.toContain('View details');
  });

  it('renders healthy summary text instead of an empty state when health payload has no items', () => {
    const wrapper = mountRenderer([
      baseWidget({
        payload: {
          summary: {
            status: 'healthy',
          },
          abnormal_services: 0,
          healthy_modules: 7,
          items: [],
        },
        title_key: 'dashboard.missingTitle',
      }),
    ]);

    expect(wrapper.text()).toContain('7 modules running, 0 modules need attention');
    expect(wrapper.text()).not.toContain('No widgets');
  });
});
