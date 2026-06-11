// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import type { DashboardQuickActionConfig } from '../contract/quick-actions';
import type { DashboardSummaryResponse, DashboardWidget } from '../types/dashboard';
import DashboardHomePage from './index.vue';

const dashboardApiMocks = vi.hoisted(() => ({
  getDashboardSummary: vi.fn(),
  getDashboardWidget: vi.fn(),
}));

const quickActionConfigApiMocks = vi.hoisted(() => ({
  getDashboardSystemConfigs: vi.fn(),
}));

const loggerMocks = vi.hoisted(() => ({
  error: vi.fn(),
}));

const routerMocks = vi.hoisted(() => ({
  push: vi.fn(),
}));

vi.mock('../api/dashboard', () => ({
  getDashboardSummary: dashboardApiMocks.getDashboardSummary,
  getDashboardWidget: dashboardApiMocks.getDashboardWidget,
}));

vi.mock('../api/quick-actions-config', () => ({
  getDashboardSystemConfigs: quickActionConfigApiMocks.getDashboardSystemConfigs,
}));

vi.mock('@/locales', () => ({
  currentLocale: 'en-US',
  i18n: {
    global: {
      getLocaleMessage: () => ({}),
    },
  },
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
      'dashboard.page.lastUpdated': `Last updated: ${params?.time ?? ''}`,
      'dashboard.page.title': 'Home',
      'dashboard.quickActions.description': 'Permission entries',
      'dashboard.quickActions.empty': 'No quick actions',
      'dashboard.quickActions.title': 'Quick Actions',
      'dashboard.systemSummary.abnormalServices.description': 'Services that need attention',
      'dashboard.systemSummary.abnormalServices.label': 'Abnormal services',
      'dashboard.systemSummary.abnormalServices.value': `${params?.count ?? 0}`,
      'dashboard.systemSummary.currentUser.label': 'Current user',
      'dashboard.systemSummary.environment.description': 'Runtime environment',
      'dashboard.systemSummary.environment.label': 'Environment',
      'dashboard.systemSummary.eyebrow': 'Today overview',
      'dashboard.systemSummary.failedTasks.description': 'Latest failed tasks',
      'dashboard.systemSummary.failedTasks.label': 'Failed tasks',
      'dashboard.systemSummary.failedTasks.value': `${params?.count ?? 0}`,
      'dashboard.systemSummary.highRiskEvents.description': 'High-risk events',
      'dashboard.systemSummary.highRiskEvents.label': 'High-risk events',
      'dashboard.systemSummary.highRiskEvents.value': `${params?.count ?? 0}`,
      'dashboard.systemSummary.locale.description': `Fallback locale ${params?.fallback ?? ''}`,
      'dashboard.systemSummary.locale.label': 'Locale',
      'dashboard.systemSummary.modules.description': `${params?.total ?? 0} total, ${params?.degraded ?? 0} degraded`,
      'dashboard.systemSummary.modules.label': 'Module runtime',
      'dashboard.systemSummary.modules.value': `${params?.count ?? 0} running`,
      'dashboard.systemSummary.title': 'Today overview',
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

vi.mock('vue-router', () => ({
  useRouter: () => routerMocks,
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

const quickActionsStub = defineComponent({
  name: 'DashboardQuickActions',
  props: {
    config: {
      type: Object,
      default: () => ({ enabled: true, maxItems: 8, strategy: 'hybrid' }),
    },
    links: {
      type: Array,
      default: () => [],
    },
  },
  setup(props) {
    return () => {
      const config = props.config as DashboardQuickActionConfig;
      const links = summaryQuickLinks(props.links);
      return h(
        'section',
        {
          class: 'quick-actions-stub',
          'data-enabled': String(config.enabled),
          'data-max-items': String(config.maxItems),
          'data-strategy': config.strategy,
        },
        config.enabled
          ? links.slice(0, config.maxItems).map((link) =>
              h(
                'button',
                {
                  class: 'dashboard-quick-actions__item',
                  onClick: () => routerMocks.push(link.route_location),
                },
                [h('strong', link.title), link.description ? h('small', link.description) : null],
              ),
            )
          : [],
      );
    };
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
      h('div', [
        props.title,
        props.message,
        props.description,
        props.text,
        slots.title?.(),
        slots.default?.(),
        slots.operation?.(),
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

function summaryResponse(): DashboardSummaryResponse {
  return {
    system_summary: {
      abnormal_services: 0,
      app_env: 'development',
      current_user: {
        display_name: 'Admin',
        username: 'admin',
      },
      failed_tasks: 0,
      high_risk_events: 0,
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
    quick_links: [
      {
        id: 'rbac.roles',
        module_key: 'rbac',
        order: 20,
        route_location: '/rbac/roles',
        title: 'Roles',
      },
      {
        description: 'Review events',
        id: 'audit.logs',
        module_key: 'audit',
        order: 10,
        route_location: '/audit/events?level=warning',
        title: 'Audit Logs',
      },
    ],
    widgets: [
      {
        category: 'system',
        id: 'core.module-runtime-health',
        module_key: 'core',
        order: 1,
        payload: {
          summary: {
            status: 'healthy',
          },
          items: [],
        },
        priority: 'info',
        size: 'medium',
        state: 'normal',
        title: 'Module Health',
        type: 'health',
        visible: true,
      },
      {
        category: 'system',
        id: 'monitor.system-health',
        module_key: 'monitor',
        order: 2,
        payload: {
          summary: {
            status: 'healthy',
          },
          items: [],
        },
        priority: 'normal',
        size: 'medium',
        state: 'normal',
        title: 'System Health',
        type: 'health',
        visible: true,
      },
    ],
  };
}

function summaryQuickLinks(value: unknown) {
  return [...(value as DashboardSummaryResponse['quick_links'])].sort((left, right) => {
    if (left.order !== right.order) {
      return left.order - right.order;
    }
    return left.id.localeCompare(right.id);
  });
}

function quickActionsConfigItem(effectiveValue: string) {
  return {
    config_schema: {},
    default_value: null,
    effective_value: effectiveValue,
    group: 'dashboard.quick_actions',
    has_override: false,
    key: 'dashboard.quick_actions',
    masked: false,
    module: 'core',
    restart_required: false,
    sensitive: false,
    status: 'default',
    type: 'string',
  } as const;
}

function mountPage() {
  return mount(DashboardHomePage, {
    global: {
      stubs: {
        DashboardQuickActions: quickActionsStub,
        DashboardRenderer: rendererStub,
        TAlert: passthroughStub,
        TBadge: passthroughStub,
        TBreadcrumb: passthroughStub,
        TBreadcrumbItem: passthroughStub,
        TButton: buttonStub,
        TCard: passthroughStub,
        TDrawer: drawerStub,
        TEmpty: passthroughStub,
        TIcon: passthroughStub,
        TLoading: passthroughStub,
        TSkeleton: passthroughStub,
        't-button': buttonStub,
        't-card': passthroughStub,
        't-badge': passthroughStub,
        't-breadcrumb': passthroughStub,
        't-breadcrumb-item': passthroughStub,
        't-drawer': drawerStub,
        't-empty': passthroughStub,
        't-icon': passthroughStub,
        't-loading': passthroughStub,
        't-skeleton': passthroughStub,
      },
    },
  });
}

describe('DashboardHomePage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    quickActionConfigApiMocks.getDashboardSystemConfigs.mockResolvedValue({ items: [] });
  });

  it('loads and renders the fixed system summary plus API-provided quick links and widgets', async () => {
    dashboardApiMocks.getDashboardSummary.mockResolvedValueOnce(summaryResponse());

    const wrapper = mountPage();
    await flushPromises();

    expect(dashboardApiMocks.getDashboardSummary).toHaveBeenCalledTimes(1);
    expect(wrapper.text()).toContain('Today overview');
    expect(wrapper.text()).toContain('4 running');
    expect(wrapper.text()).toContain('Abnormal services');
    expect(wrapper.text()).toContain('Failed tasks');
    expect(wrapper.text()).toContain('High-risk events');
    expect(wrapper.text()).toContain('Audit Logs');
    expect(wrapper.text()).toContain('Review events');
    expect(wrapper.text()).toContain('Roles');
    expect(wrapper.text()).toContain('core.module-runtime-health');
    expect(wrapper.text()).toContain('monitor.system-health');
  });

  it('opens the API-provided route when a quick action is clicked', async () => {
    dashboardApiMocks.getDashboardSummary.mockResolvedValueOnce(summaryResponse());

    const wrapper = mountPage();
    await flushPromises();

    const quickActionButtons = wrapper.findAll('button.dashboard-quick-actions__item');
    expect(quickActionButtons).toHaveLength(2);

    await quickActionButtons[0].trigger('click');

    expect(routerMocks.push).toHaveBeenCalledWith('/audit/events?level=warning');
  });

  it('passes disabled quick-action config from system config to the dashboard section', async () => {
    dashboardApiMocks.getDashboardSummary.mockResolvedValueOnce(summaryResponse());
    quickActionConfigApiMocks.getDashboardSystemConfigs.mockResolvedValueOnce({
      items: [quickActionsConfigItem('{"enabled":false,"maxItems":4,"strategy":"hybrid"}')],
    });

    const wrapper = mountPage();
    await flushPromises();

    const quickActions = wrapper.find('.quick-actions-stub');
    expect(quickActions.attributes('data-enabled')).toBe('false');
    expect(wrapper.findAll('button.dashboard-quick-actions__item')).toHaveLength(0);
  });

  it('passes max item quick-action config from system config to the dashboard section', async () => {
    dashboardApiMocks.getDashboardSummary.mockResolvedValueOnce(summaryResponse());
    quickActionConfigApiMocks.getDashboardSystemConfigs.mockResolvedValueOnce({
      items: [quickActionsConfigItem('{"enabled":true,"maxItems":1,"strategy":"hybrid"}')],
    });

    const wrapper = mountPage();
    await flushPromises();

    const quickActions = wrapper.find('.quick-actions-stub');
    expect(quickActions.attributes('data-max-items')).toBe('1');
    expect(wrapper.findAll('button.dashboard-quick-actions__item')).toHaveLength(1);
    expect(wrapper.text()).toContain('Audit Logs');
    expect(wrapper.text()).not.toContain('Roles');
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
    const renderer = wrapper.findComponent(rendererStub);
    const renderedWidgets = renderer.props('widgets') as DashboardWidget[];
    expect(renderedWidgets.find((widget) => widget.id === 'core.module-runtime-health')?.title).toBe(
      'Updated Module Health',
    );
  });

  it('shows a page error when summary loading fails', async () => {
    dashboardApiMocks.getDashboardSummary.mockRejectedValueOnce(new Error('Network failed'));

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain('Dashboard load failed');
    expect(wrapper.text()).toContain('Network failed');
  });
});
