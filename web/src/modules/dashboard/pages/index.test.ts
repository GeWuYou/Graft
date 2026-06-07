import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import type { DashboardSummaryResponse, DashboardWidget } from '../types/dashboard';
import DashboardHomePage from './index.vue';

const dashboardApiMocks = vi.hoisted(() => ({
  getDashboardSummary: vi.fn(),
  getDashboardWidget: vi.fn(),
}));

const loggerMocks = vi.hoisted(() => ({
  error: vi.fn(),
}));

vi.mock('../api/dashboard', () => ({
  getDashboardSummary: dashboardApiMocks.getDashboardSummary,
  getDashboardWidget: dashboardApiMocks.getDashboardWidget,
}));

vi.mock('@/locales', () => ({
  t: (key: string, params?: Record<string, unknown>) => {
    const translations: Record<string, string> = {
      'dashboard.actions.refresh': 'Refresh',
      'dashboard.actions.retry': 'Retry',
      'dashboard.empty': 'No dashboard data',
      'dashboard.error.fallback': 'Dashboard failed',
      'dashboard.error.title': 'Dashboard load failed',
      'dashboard.loading': 'Loading dashboard',
      'dashboard.page.description': 'Dashboard description',
      'dashboard.page.eyebrow': 'Workspace',
      'dashboard.page.title': 'Home',
      'dashboard.systemSummary.currentUser.label': 'Current user',
      'dashboard.systemSummary.environment.description': 'Runtime environment',
      'dashboard.systemSummary.environment.label': 'Environment',
      'dashboard.systemSummary.locale.description': `Fallback locale ${params?.fallback ?? ''}`,
      'dashboard.systemSummary.locale.label': 'Locale',
      'dashboard.systemSummary.modules.description': `${params?.total ?? 0} total, ${params?.degraded ?? 0} degraded`,
      'dashboard.systemSummary.modules.label': 'Enabled modules',
      'dashboard.systemSummary.title': 'System summary',
      'dashboard.systemSummary.widgets.description': 'Visible widgets',
      'dashboard.systemSummary.widgets.label': 'Widgets',
      'dashboard.widget.errorFallback': 'Widget failed',
    };
    return translations[key] ?? key;
  },
}));

vi.mock('@/utils/logger', () => ({
  createLogger: () => loggerMocks,
}));

const rendererStub = defineComponent({
  name: 'DashboardRendererStub',
  props: {
    widgets: {
      type: Array,
      default: () => [],
    },
  },
  emits: ['refresh-widget'],
  setup(props, { emit }) {
    return () =>
      h('div', { class: 'renderer-stub' }, [
        (props.widgets as DashboardWidget[]).map((widget) => h('span', { class: 'widget-id' }, widget.id)),
        h('button', { class: 'refresh-widget', onClick: () => emit('refresh-widget', 'core.module-runtime-health') }),
      ]);
  },
});

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
    text: {
      type: String,
      default: '',
    },
  },
  setup(props, { slots }) {
    return () =>
      h('div', [props.title, props.message, props.description, props.text, slots.default?.(), slots.operation?.()]);
  },
});

const buttonStub = defineComponent({
  name: 'TButtonStub',
  emits: ['click'],
  setup(_props, { emit, slots }) {
    return () => h('button', { onClick: (event: MouseEvent) => emit('click', event) }, slots.default?.());
  },
});

function summaryResponse(): DashboardSummaryResponse {
  return {
    system_summary: {
      app_env: 'development',
      current_user: {
        display_name: 'Admin',
        username: 'admin',
      },
      locale: {
        default_locale: 'zh-CN',
        fallback_locale: 'zh-CN',
      },
      modules: {
        degraded_modules: 1,
        enabled_modules: 4,
        total_modules: 5,
      },
      visible_widgets: 1,
    },
    widgets: [
      {
        id: 'core.module-runtime-health',
        module_key: 'core',
        order: 1,
        payload: {
          summary: {
            status: 'healthy',
          },
          items: [],
        },
        size: 'medium',
        title: 'Module Health',
        type: 'health',
      },
    ],
  };
}

function mountPage() {
  return mount(DashboardHomePage, {
    global: {
      stubs: {
        DashboardRenderer: rendererStub,
        TAlert: passthroughStub,
        TButton: buttonStub,
        TEmpty: passthroughStub,
        TLoading: passthroughStub,
      },
    },
  });
}

describe('DashboardHomePage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('loads and renders the fixed system summary plus widgets', async () => {
    dashboardApiMocks.getDashboardSummary.mockResolvedValueOnce(summaryResponse());

    const wrapper = mountPage();
    await flushPromises();

    expect(dashboardApiMocks.getDashboardSummary).toHaveBeenCalledTimes(1);
    expect(wrapper.text()).toContain('Admin');
    expect(wrapper.text()).toContain('development');
    expect(wrapper.text()).toContain('zh-CN');
    expect(wrapper.text()).toContain('4');
    expect(wrapper.text()).toContain('core.module-runtime-health');
  });

  it('refreshes one widget through the focused widget endpoint', async () => {
    dashboardApiMocks.getDashboardSummary.mockResolvedValueOnce(summaryResponse());
    dashboardApiMocks.getDashboardWidget.mockResolvedValueOnce({
      ...summaryResponse().widgets[0],
      title: 'Updated Module Health',
    });

    const wrapper = mountPage();
    await flushPromises();
    await wrapper.find('.refresh-widget').trigger('click');
    await flushPromises();

    expect(dashboardApiMocks.getDashboardWidget).toHaveBeenCalledWith('core.module-runtime-health');
    expect(wrapper.text()).toContain('core.module-runtime-health');
  });

  it('shows a page error when summary loading fails', async () => {
    dashboardApiMocks.getDashboardSummary.mockRejectedValueOnce(new Error('Network failed'));

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain('Dashboard load failed');
    expect(wrapper.text()).toContain('Network failed');
  });
});
