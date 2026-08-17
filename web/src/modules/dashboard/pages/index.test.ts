import { flushPromises, mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, type PropType } from 'vue';
import type { RouteRecordRaw } from 'vue-router';

import { resetContainerStatsManager } from '@/modules/container/shared/stats-manager';
import { usePermissionStore } from '@/store/modules/permission';

import type { WorkbenchPresentation } from '../presentation/workbench';
import type { DashboardSummaryResponse } from '../types/dashboard';
import DashboardHomePage from './index.vue';

const dashboardApiMocks = vi.hoisted(() => ({ getDashboardSummary: vi.fn(), getDashboardWidget: vi.fn() }));
const configApiMocks = vi.hoisted(() => ({ getDashboardSystemConfigs: vi.fn() }));
const containerApiMocks = vi.hoisted(() => ({ getContainerDashboardSummary: vi.fn() }));
const routerMocks = vi.hoisted(() => ({ push: vi.fn() }));

vi.mock('../api/dashboard', () => dashboardApiMocks);
vi.mock('../api/quick-actions-config', () => configApiMocks);
vi.mock('vue-router', () => ({ useRouter: () => routerMocks }));
vi.mock('@/modules/container', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/modules/container')>();
  return {
    ...actual,
    containerModuleFacades: {
      ...actual.containerModuleFacades,
      getContainerDashboardSummary: containerApiMocks.getContainerDashboardSummary,
    },
  };
});
vi.mock('@/shared/realtime', () => ({
  createRealtimeSnapshotGate: ({ apply }: { apply: (value: unknown) => void }) => ({
    clear: vi.fn(),
    commit: apply,
    dispose: vi.fn(),
    flush: vi.fn(),
  }),
  openRealtimeTopicSocket: () => ({ close: vi.fn(), reconnect: vi.fn() }),
}));
vi.mock('@/utils/logger', () => ({
  createLogger: () => ({ error: vi.fn(), warn: vi.fn() }),
}));
vi.mock('../components/workbench/DashboardWorkbench.vue', () => ({
  default: defineComponent({
    name: 'DashboardWorkbenchStub',
    props: {
      errorMessage: { type: String, default: '' },
      presentation: { type: Object as PropType<WorkbenchPresentation>, required: true },
      quickActionsEnabled: Boolean,
      ready: Boolean,
      retryingId: { type: String, default: '' },
    },
    emits: ['navigate', 'refresh', 'retry-item'],
    setup(props, { emit }) {
      return () =>
        h('section', { 'data-error': props.errorMessage, 'data-ready': String(props.ready) }, [
          h('button', { class: 'refresh-page', onClick: () => emit('refresh') }, 'refresh'),
          ...props.presentation.attention.map((item) =>
            h(
              'button',
              {
                class: 'attention-item',
                'data-evidence': item.evidenceState,
                'data-status': item.status,
                onClick: () => emit('retry-item', item),
              },
              item.titleFallback || item.titleKey,
            ),
          ),
          ...props.presentation.health.map((item) =>
            h('span', { class: 'health-item', 'data-status': item.status }, item.titleFallback || item.titleKey),
          ),
          ...props.presentation.resources.map((item) =>
            h('span', { class: 'resource-item', 'data-evidence': item.evidenceState, 'data-status': item.status }),
          ),
          ...(props.quickActionsEnabled
            ? props.presentation.homeQuickActions.map((action) =>
                h(
                  'button',
                  { class: 'quick-action', onClick: () => emit('navigate', action.route) },
                  action.titleFallback || action.titleKey,
                ),
              )
            : []),
        ]);
    },
  }),
}));

function summaryResponse(overrides: Partial<DashboardSummaryResponse> = {}): DashboardSummaryResponse {
  return {
    system_summary: {
      abnormal_services: 0,
      app_env: 'development',
      current_user: { display_name: 'Admin', username: 'admin' },
      failed_tasks: 0,
      high_risk_events: 0,
      locale: { default_locale: 'zh-CN', fallback_locale: 'zh-CN' },
      modules: { degraded_modules: 0, enabled_modules: 4, total_modules: 4 },
      visible_widgets: 1,
    },
    widgets: [
      {
        category: 'system',
        id: 'core-health',
        module_key: 'core',
        order: 1,
        payload: {
          summary: { status: 'healthy' },
          items: [
            { key: 'postgresql', label_key: 'postgresql', label: 'PostgreSQL', status: 'healthy' },
            { key: 'redis', label_key: 'redis', label: 'Redis', status: 'healthy' },
          ],
        },
        priority: 'critical',
        size: 'medium',
        state: 'critical',
        status: 'normal',
        type: 'health',
        visible: true,
      },
    ],
    ...overrides,
  };
}

function containerSummary(collectedAt: string | null = '2026-08-17T03:20:00Z') {
  return {
    overview: {
      abnormalContainers: 0,
      collectedAt,
      cpuTotalPercent: 12.5,
      memoryTotalLimitBytes: null,
      memoryTotalPercent: null,
      memoryTotalUsageBytes: null,
      runningContainers: 2,
    },
    hotspots: { cpu: [], memory: [] },
    anomalies: [],
  };
}

function configItem(value: string) {
  return {
    config_schema: {},
    default_value: null,
    effective_value: value,
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

function routes(count = 8): RouteRecordRaw[] {
  return Array.from(
    { length: count },
    (_, index) =>
      ({
        path: `/feature-${index}`,
        name: `Feature${index}Index`,
        meta: {
          icon: 'platform',
          orderNo: index,
          single: true,
          title: { 'en-US': `Feature ${index}`, 'zh-CN': `功能 ${index}` },
          titleKey: `feature${index}.title`,
        },
      }) as unknown as RouteRecordRaw,
  );
}

function mountPage() {
  return mount(DashboardHomePage);
}

describe('DashboardHomePage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resetContainerStatsManager();
    setActivePinia(createPinia());
    usePermissionStore().routers = routes();
    usePermissionStore().setBootstrapSnapshot({ permissions: ['container.view'], menus: [], user: null } as never);
    dashboardApiMocks.getDashboardSummary.mockResolvedValue(summaryResponse());
    configApiMocks.getDashboardSystemConfigs.mockResolvedValue({ items: [] });
    containerApiMocks.getContainerDashboardSummary.mockResolvedValue(containerSummary());
  });

  it('projects real summary health and resource data into the accepted workbench', async () => {
    const wrapper = mountPage();
    await flushPromises();

    const workbench = wrapper.getComponent({ name: 'DashboardWorkbenchStub' });
    const presentation = workbench.props('presentation') as WorkbenchPresentation;
    expect(workbench.props('ready')).toBe(true);
    expect(presentation.operational.enabledModules).toBe(4);
    expect(presentation.health.map((item) => item.status)).toEqual(['healthy', 'healthy']);
    expect(presentation.resources[0]).toMatchObject({ status: 'info', evidenceState: 'confirmed' });
    expect(wrapper.findAll('.quick-action')).toHaveLength(4);
  });

  it('honors configurable quick-action counts without fixing the homepage to four items', async () => {
    configApiMocks.getDashboardSystemConfigs.mockResolvedValueOnce({
      items: [configItem('{"enabled":true,"maxItems":6,"strategy":"hybrid"}')],
    });
    const wrapper = mountPage();
    await flushPromises();

    const presentation = wrapper
      .getComponent({ name: 'DashboardWorkbenchStub' })
      .props('presentation') as WorkbenchPresentation;
    expect(presentation.quickActions).toHaveLength(8);
    expect(presentation.homeQuickActions).toHaveLength(6);
    expect(wrapper.findAll('.quick-action')).toHaveLength(6);
  });

  it('hides quick actions when the existing system config disables them', async () => {
    configApiMocks.getDashboardSystemConfigs.mockResolvedValueOnce({
      items: [configItem('{"enabled":false,"maxItems":4,"strategy":"hybrid"}')],
    });
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.findAll('.quick-action')).toHaveLength(0);
  });

  it('shows only a page-level error when the whole summary request fails', async () => {
    dashboardApiMocks.getDashboardSummary.mockRejectedValueOnce(new Error('Network failed'));
    const wrapper = mountPage();
    await flushPromises();

    const workbench = wrapper.getComponent({ name: 'DashboardWorkbenchStub' });
    expect(workbench.props('ready')).toBe(false);
    expect(workbench.props('errorMessage')).toBe('Network failed');
  });

  it('keeps a widget loader failure local, warning, and retryable', async () => {
    const failedSummary = summaryResponse();
    failedSummary.widgets[0] = {
      ...failedSummary.widgets[0]!,
      id: 'access-log-source',
      status: 'error',
      state: 'critical',
      priority: 'critical',
      title: 'Access Logs',
    };
    dashboardApiMocks.getDashboardSummary.mockResolvedValueOnce(failedSummary);
    dashboardApiMocks.getDashboardWidget.mockResolvedValueOnce(summaryResponse().widgets[0]);
    const wrapper = mountPage();
    await flushPromises();

    const attention = wrapper.get('.attention-item');
    expect(attention.attributes('data-status')).toBe('warning');
    expect(attention.attributes('data-evidence')).toBe('source-failed');
    expect(wrapper.getComponent({ name: 'DashboardWorkbenchStub' }).props('errorMessage')).toBe('');

    await attention.trigger('click');
    await flushPromises();
    expect(dashboardApiMocks.getDashboardWidget).toHaveBeenCalledWith('access-log-source');
  });

  it('distinguishes failed resource loading from missing permission', async () => {
    containerApiMocks.getContainerDashboardSummary.mockRejectedValueOnce(new Error('container unavailable'));
    const failedWrapper = mountPage();
    await flushPromises();
    expect(failedWrapper.get('.resource-item').attributes('data-evidence')).toBe('source-failed');
    failedWrapper.unmount();

    resetContainerStatsManager();
    setActivePinia(createPinia());
    usePermissionStore().routers = routes();
    usePermissionStore().setBootstrapSnapshot({ permissions: [], menus: [], user: null } as never);
    dashboardApiMocks.getDashboardSummary.mockResolvedValueOnce(summaryResponse());
    const hiddenWrapper = mountPage();
    await flushPromises();
    expect(hiddenWrapper.find('.resource-item').exists()).toBe(false);
  });

  it('records and opens the selected authorized quick action', async () => {
    const wrapper = mountPage();
    await flushPromises();
    await wrapper.get('.quick-action').trigger('click');
    expect(routerMocks.push).toHaveBeenCalledWith('/feature-0');
  });
});
