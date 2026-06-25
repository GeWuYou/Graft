import { flushPromises, mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, ref } from 'vue';
import type { RouteRecordRaw } from 'vue-router';

import { resetContainerStatsManager } from '@/modules/container/shared/stats-manager';
import { usePermissionStore } from '@/store/modules/permission';

import type { DashboardQuickActionConfig } from '../contract/quick-actions';
import type { DashboardSummaryResponse, DashboardWidget } from '../types/dashboard';
import DashboardHomePage from './index.vue';

function asRouteRecordRaw<T extends object>(route: T) {
  return route as unknown as RouteRecordRaw;
}

const dashboardApiMocks = vi.hoisted(() => ({
  getDashboardSummary: vi.fn(),
  getDashboardWidget: vi.fn(),
}));

const quickActionConfigApiMocks = vi.hoisted(() => ({
  getDashboardSystemConfigs: vi.fn(),
}));

const containerDashboardApiMocks = vi.hoisted(() => ({
  getContainerDashboardSummary: vi.fn(),
}));

const realtimeMocks = vi.hoisted(() => ({
  controllers: [] as Array<{
    close: ReturnType<typeof vi.fn>;
    emitMessage: (payload: unknown) => void;
    reconnect: ReturnType<typeof vi.fn>;
  }>,
  openRealtimeTopicSocket: vi.fn(
    (options?: { onMessage?: (payload: unknown) => void; parseMessage?: (payload: unknown) => unknown }) => {
      const controller = {
        close: vi.fn(),
        emitMessage: (payload: unknown) => {
          const parsed = options?.parseMessage ? options.parseMessage(payload) : payload;
          if (parsed) {
            options?.onMessage?.(parsed);
          }
        },
        reconnect: vi.fn(),
      };
      realtimeMocks.controllers.push(controller);
      return controller;
    },
  ),
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

vi.mock('@/shared/realtime', () => ({
  openRealtimeTopicSocket: realtimeMocks.openRealtimeTopicSocket,
}));

vi.mock('@/modules/container', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/modules/container')>();
  return {
    ...actual,
    containerModuleFacades: {
      ...actual.containerModuleFacades,
      getContainerDashboardSummary: containerDashboardApiMocks.getContainerDashboardSummary,
    },
  };
});

vi.mock('../components/DashboardRenderer.vue', () => ({
  default: defineComponent({
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
  }),
}));

vi.mock('../components/DashboardQuickActions.vue', () => ({
  default: defineComponent({
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
        const links = [
          ...(props.links as Array<{
            id: string;
            order: number;
            route_location: string;
            title?: string;
            group?: string;
            full_label?: string;
          }>),
        ].sort((left, right) => {
          if (left.order !== right.order) {
            return left.order - right.order;
          }
          return left.id.localeCompare(right.id);
        });
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
                    title: link.full_label || link.title,
                    onClick: () => routerMocks.push(link.route_location),
                  },
                  [h('strong', link.title), h('small', link.group || '')],
                ),
              )
            : [],
        );
      };
    },
  }),
}));

vi.mock('../components/DashboardContainerResources.vue', () => ({
  default: defineComponent({
    name: 'DashboardContainerResourcesStub',
    props: {
      summary: {
        type: Object,
        default: () => ({
          overview: {},
          hotspots: { cpu: [], memory: [] },
          anomalies: [],
        }),
      },
      loading: {
        type: Boolean,
        default: false,
      },
    },
    setup(props) {
      return () => {
        const summary = props.summary as {
          overview?: { runningContainers?: number; abnormalContainers?: number };
          hotspots?: { cpu?: Array<unknown>; memory?: Array<unknown> };
          anomalies?: Array<unknown>;
        };
        return h('section', {
          class: 'dashboard-container-resources-stub',
          'data-loading': String(props.loading),
          'data-running': String(summary.overview?.runningContainers ?? 0),
          'data-abnormal': String(summary.overview?.abnormalContainers ?? 0),
          'data-cpu-hotspots': String(summary.hotspots?.cpu?.length ?? 0),
          'data-memory-hotspots': String(summary.hotspots?.memory?.length ?? 0),
          'data-anomalies': String(summary.anomalies?.length ?? 0),
        });
      };
    },
  }),
}));

vi.mock('@/locales', () => ({
  currentLocale: ref('en-US'),
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

vi.mock('@/shared/components/page', () => ({
  PageHeader: defineComponent({
    name: 'PageHeaderStub',
    setup(_props, { slots }) {
      return () => h('section', { class: 'page-header-stub' }, [slots.default?.(), slots.actions?.()]);
    },
  }),
}));

vi.mock('@/utils/logger', () => ({
  createLogger: () => loggerMocks,
}));

vi.mock('@/shared/observability', () => ({
  MEDIUM_DATE_TIME_WITH_SECONDS_FORMAT_OPTIONS: {},
  formatLocaleDateTime: (value: string) => value,
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

function containerDashboardSummaryResponse() {
  return {
    overview: {
      runningContainers: 10,
      abnormalContainers: 3,
      cpuTotalPercent: 42.5,
      memoryTotalUsageBytes: 2147483648,
      memoryTotalLimitBytes: 4294967296,
      memoryTotalPercent: 58.3,
      collectedAt: '2026-06-24T00:02:00Z',
    },
    hotspots: {
      cpu: [
        {
          id: 'cpu-1',
          name: 'graft-server',
          state: 'running',
          health: null,
          image: '',
          shortId: 'cpu-1',
          restartCount: null,
          cpuPercent: 42.5,
          memoryPercent: 12.4,
          memoryUsageBytes: 536870912,
          memoryLimitBytes: 2147483648,
          collectedAt: '2026-06-24T00:01:00Z',
        },
      ],
      memory: [
        {
          id: 'mem-1',
          name: 'graft-worker',
          state: 'running',
          health: null,
          image: '',
          shortId: 'mem-1',
          restartCount: null,
          cpuPercent: 12.2,
          memoryPercent: 58.3,
          memoryUsageBytes: 1073741824,
          memoryLimitBytes: 2147483648,
          collectedAt: '2026-06-24T00:02:00Z',
        },
      ],
    },
    anomalies: [
      {
        id: 'bad-1',
        name: 'graft-scheduler',
        state: 'restarting',
        health: null,
        image: '',
        shortId: 'bad-1',
        restartCount: null,
        cpuPercent: 2.4,
        memoryPercent: 12.3,
        memoryUsageBytes: null,
        memoryLimitBytes: null,
        collectedAt: '2026-06-24T00:02:00Z',
      },
    ],
  };
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

function buildSidebarRoutes() {
  return [
    asRouteRecordRaw({
      path: '/audit',
      name: 'BootstrapGroupAudit',
      meta: {
        titleKey: 'audit.route.group.title',
        title: {
          'zh-CN': '安全审计',
          'en-US': 'Security Audit',
        },
      },
      children: [
        asRouteRecordRaw({
          path: 'events',
          name: 'AuditEventListIndex',
          meta: {
            icon: 'secured',
            orderNo: 10,
            tabTitle: {
              'zh-CN': '审计中心 - 事件',
              'en-US': 'Security Audit - Events',
            },
            breadcrumbTitle: {
              'zh-CN': '事件',
              'en-US': 'Events',
            },
            titleKey: 'audit.route.events.title',
          },
        }),
      ],
    }),
    asRouteRecordRaw({
      path: '/ops/containers',
      name: 'ContainerList',
      meta: {
        icon: 'layers',
        orderNo: 15,
        single: true,
        title: {
          'zh-CN': '运维管理',
          'en-US': 'Operations',
        },
        titleKey: 'container.route.list.title',
        tabTitle: {
          'zh-CN': '运维管理 - 容器管理',
          'en-US': 'Operations - Container Management',
        },
        breadcrumbTitle: {
          'zh-CN': '容器管理',
          'en-US': 'Container Management',
        },
      },
      children: [
        asRouteRecordRaw({
          path: 'index',
          name: 'ContainerListIndex',
          meta: {
            hidden: true,
            titleKey: 'container.route.list.title',
          },
        }),
      ],
    }),
    asRouteRecordRaw({
      path: '/access-control',
      name: 'BootstrapGroupAccessControl',
      meta: {
        titleKey: 'accessControl.route.group.title',
        title: {
          'zh-CN': '访问控制',
          'en-US': 'Access Control',
        },
      },
      children: [
        asRouteRecordRaw({
          path: 'roles',
          name: 'RoleListIndex',
          meta: {
            orderNo: 20,
            tabTitle: {
              'zh-CN': '访问控制 - 角色管理',
              'en-US': 'Access Control - Role Management',
            },
            breadcrumbTitle: {
              'zh-CN': '角色管理',
              'en-US': 'Role Management',
            },
            titleKey: 'rbac.role.list.title',
          },
        }),
        asRouteRecordRaw({
          path: 'permissions',
          name: 'PermissionListIndex',
          meta: {
            hiddenMenu: true,
            orderNo: 25,
            tabTitle: {
              'zh-CN': '访问控制 - 权限管理',
              'en-US': 'Access Control - Permission Management',
            },
            breadcrumbTitle: {
              'zh-CN': '权限管理',
              'en-US': 'Permission Management',
            },
            titleKey: 'rbac.permission.list.title',
          },
        }),
      ],
    }),
  ] as RouteRecordRaw[];
}

describe('DashboardHomePage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resetContainerStatsManager();
    realtimeMocks.controllers = [];
    setActivePinia(createPinia());
    quickActionConfigApiMocks.getDashboardSystemConfigs.mockResolvedValue({ items: [] });
    containerDashboardApiMocks.getContainerDashboardSummary.mockResolvedValue(containerDashboardSummaryResponse());
    usePermissionStore().routers = buildSidebarRoutes();
    usePermissionStore().setBootstrapSnapshot({
      permissions: ['ops.container.view'],
      menus: [],
      user: null,
    } as never);
  });

  it('loads and renders the fixed system summary plus sidebar-derived quick links and widgets', async () => {
    dashboardApiMocks.getDashboardSummary.mockResolvedValueOnce(summaryResponse());

    const wrapper = mountPage();
    await flushPromises();

    expect(dashboardApiMocks.getDashboardSummary).toHaveBeenCalledTimes(1);
    expect(wrapper.text()).toContain('Today overview');
    expect(wrapper.text()).toContain('4 running');
    expect(wrapper.text()).toContain('Abnormal services');
    expect(wrapper.text()).toContain('Failed tasks');
    expect(wrapper.text()).toContain('High-risk events');
    expect(wrapper.text()).toContain('Events');
    expect(wrapper.text()).toContain('Security Audit');
    expect(wrapper.text()).toContain('Container Management');
    expect(wrapper.text()).toContain('Operations');
    expect(wrapper.text()).toContain('Role Management');
    expect(wrapper.text()).toContain('Access Control');
    expect(wrapper.text()).not.toContain('Access Control - Permissions');
    expect(wrapper.text()).toContain('core.module-runtime-health');
    expect(wrapper.text()).toContain('monitor.system-health');
    expect(containerDashboardApiMocks.getContainerDashboardSummary).toHaveBeenCalledTimes(1);
    expect(wrapper.find('.dashboard-container-resources-stub').attributes('data-running')).toBe('10');
    expect(wrapper.find('.dashboard-container-resources-stub').attributes('data-anomalies')).toBe('1');
  });

  it('does not acquire the dashboard collection subscription twice on repeated refresh', async () => {
    dashboardApiMocks.getDashboardSummary.mockResolvedValue(summaryResponse());

    const wrapper = mountPage();
    await flushPromises();

    await wrapper.find('button').trigger('click');
    await flushPromises();

    expect(containerDashboardApiMocks.getContainerDashboardSummary).toHaveBeenCalledTimes(2);
    expect(realtimeMocks.openRealtimeTopicSocket).toHaveBeenCalledTimes(1);
  });

  it('skips container dashboard consumption when the permission is missing', async () => {
    dashboardApiMocks.getDashboardSummary.mockResolvedValueOnce(summaryResponse());
    usePermissionStore().setBootstrapSnapshot({
      permissions: [],
      menus: [],
      user: null,
    } as never);

    const wrapper = mountPage();
    await flushPromises();

    expect(containerDashboardApiMocks.getContainerDashboardSummary).not.toHaveBeenCalled();
    expect(wrapper.find('.dashboard-container-resources-stub').exists()).toBe(false);
  });

  it('opens the sidebar-derived route when a quick action is clicked', async () => {
    dashboardApiMocks.getDashboardSummary.mockResolvedValueOnce(summaryResponse());

    const wrapper = mountPage();
    await flushPromises();

    const quickActionButtons = wrapper.findAll('button.dashboard-quick-actions__item');
    expect(quickActionButtons).toHaveLength(3);

    await quickActionButtons[0].trigger('click');

    expect(routerMocks.push).toHaveBeenCalledWith('/audit/events');
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
    expect(wrapper.text()).toContain('Events');
    expect(wrapper.text()).toContain('Security Audit');
    expect(wrapper.text()).not.toContain('Container Management');
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

  it('renders manager-owned dashboard summary state and updates after realtime summary messages', async () => {
    dashboardApiMocks.getDashboardSummary.mockResolvedValueOnce(summaryResponse());

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.find('.dashboard-container-resources-stub').attributes('data-running')).toBe('10');
    expect(realtimeMocks.controllers).toHaveLength(1);

    realtimeMocks.controllers[0]!.emitMessage(
      JSON.stringify({
        data: {
          collected_at: '2026-06-24T00:03:00Z',
          overview: {
            running_containers: 12,
            abnormal_containers: 4,
            cpu_total_percent: 61.2,
            memory_total_usage_bytes: 3221225472,
            memory_total_limit_bytes: 4294967296,
            memory_total_percent: 75,
          },
          hotspots: {
            cpu_top: [],
            memory_top: [],
          },
          anomalies: [
            {
              id: 'bad-2',
              name: 'graft-cron',
              short_id: 'bad-2',
              image: 'graft/cron:latest',
              state: 'restarting',
              status: 'Restarting',
              reason_code: 'state.restarting',
              reason_label: 'Restarting',
              resource: {
                available: true,
                stats_available: true,
                cpu_percent: 4.2,
                memory_percent: 16.5,
                collected_at: '2026-06-24T00:03:00Z',
              },
            },
          ],
        },
      }),
    );
    await flushPromises();

    expect(wrapper.find('.dashboard-container-resources-stub').attributes('data-running')).toBe('12');
    expect(wrapper.find('.dashboard-container-resources-stub').attributes('data-abnormal')).toBe('4');
    expect(wrapper.find('.dashboard-container-resources-stub').attributes('data-anomalies')).toBe('1');
  });
});
