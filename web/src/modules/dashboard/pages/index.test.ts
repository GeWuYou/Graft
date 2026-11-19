import { flushPromises, mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, type PropType } from 'vue';
import type { RouteRecordRaw } from 'vue-router';

import type { BootstrapResponse } from '@/modules/auth/contract/types';
import { CONTAINER_ROUTE_PATH } from '@/modules/container/contract/paths';
import { resetContainerStatsManager } from '@/modules/container/shared/stats-manager';
import { clearQueryCache, queryClient } from '@/shared/query';
import { usePermissionStore } from '@/store/modules/permission';

import { DASHBOARD_QUICK_ACTION_STORAGE_KEY } from '../contract/quick-actions';
import type { WorkbenchPresentation } from '../presentation/workbench';
import type { DashboardSummaryResponse } from '../types/dashboard';
import DashboardHomePage from './index.vue';

const dashboardApiMocks = vi.hoisted(() => ({ getDashboardSummary: vi.fn(), getDashboardWidget: vi.fn() }));
const configApiMocks = vi.hoisted(() => ({ getSystemConfigs: vi.fn() }));
const containerApiMocks = vi.hoisted(() => ({ getContainerDashboardSummary: vi.fn() }));
const routerMocks = vi.hoisted(() => ({ push: vi.fn() }));

vi.mock('../api/dashboard', () => dashboardApiMocks);
vi.mock('@/modules/system-config/api/system-config', () => configApiMocks);
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
            h('button', {
              class: 'resource-item',
              'data-evidence': item.evidenceState,
              'data-status': item.status,
              onClick: () => item.action?.route && emit('navigate', item.action.route, 'contextual-action'),
            }),
          ),
          ...(props.quickActionsEnabled
            ? props.presentation.homeQuickActions.map((action) =>
                h(
                  'button',
                  { class: 'quick-action', onClick: () => emit('navigate', action.route, 'quick-entry') },
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
  return Array.from({ length: count }, (_, index): RouteRecordRaw => ({
    path: `/feature-${index}`,
    name: `Feature${index}Index`,
    component: defineComponent({ render: () => null }),
    meta: {
      icon: 'platform',
      orderNo: index,
      single: true,
      title: `Feature ${index}`,
      titleKey: `feature${index}.title`,
    },
  }));
}

function bootstrapSnapshot(permissions: string[]): BootstrapResponse {
  return {
    user: { id: 1, username: 'admin', display_name: 'Admin' },
    must_change_password: false,
    roles: ['admin'],
    permissions,
    menus: [],
    locale: {
      current_locale: 'zh-CN',
      default_locale: 'zh-CN',
      fallback_locale: 'zh-CN',
      supported_locales: ['zh-CN', 'en-US'],
    },
  };
}

function mountPage() {
  return mount(DashboardHomePage);
}

describe('DashboardHomePage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    clearQueryCache();
    window.localStorage.clear();
    resetContainerStatsManager();
    setActivePinia(createPinia());
    usePermissionStore().routers = routes();
    usePermissionStore().setBootstrapSnapshot(bootstrapSnapshot(['container.view', 'system-config.read']));
    dashboardApiMocks.getDashboardSummary.mockResolvedValue(summaryResponse());
    configApiMocks.getSystemConfigs.mockResolvedValue({ items: [] });
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

  it('reuses fresh dashboard snapshots across remounts and refetches them on demand', async () => {
    const first = mountPage();
    await flushPromises();
    first.unmount();

    const second = mountPage();
    await flushPromises();
    expect(second.getComponent({ name: 'DashboardWorkbenchStub' }).props('ready')).toBe(true);
    expect(dashboardApiMocks.getDashboardSummary).toHaveBeenCalledTimes(1);
    expect(configApiMocks.getSystemConfigs).toHaveBeenCalledTimes(1);
    expect(containerApiMocks.getContainerDashboardSummary).toHaveBeenCalledTimes(1);

    await second.get('.refresh-page').trigger('click');
    await flushPromises();
    expect(dashboardApiMocks.getDashboardSummary).toHaveBeenCalledTimes(2);
    expect(configApiMocks.getSystemConfigs).toHaveBeenCalledTimes(2);
    expect(containerApiMocks.getContainerDashboardSummary).toHaveBeenCalledTimes(2);
    second.unmount();
  });

  it('keeps default quick actions without querying system config when permission is absent', async () => {
    usePermissionStore().setBootstrapSnapshot(bootstrapSnapshot(['container.view']));
    const wrapper = mountPage();
    await flushPromises();
    await wrapper.get('.refresh-page').trigger('click');
    await flushPromises();

    expect(configApiMocks.getSystemConfigs).not.toHaveBeenCalled();
    expect(wrapper.findAll('.quick-action')).toHaveLength(4);
    wrapper.unmount();
  });

  it('honors configurable quick-action counts without fixing the homepage to four items', async () => {
    configApiMocks.getSystemConfigs.mockResolvedValueOnce({
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
    configApiMocks.getSystemConfigs.mockResolvedValueOnce({
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

  it('keeps the overview refresh failure through failed and successful widget retries', async () => {
    const snapshot = summaryResponse();
    const recoveredWidget = snapshot.widgets[0]!;
    const failedSnapshot = summaryResponse({ widgets: [{ ...recoveredWidget, status: 'error' }] });
    dashboardApiMocks.getDashboardSummary
      .mockResolvedValueOnce(failedSnapshot)
      .mockRejectedValueOnce(new Error('Overview refresh failed'))
      .mockResolvedValueOnce(failedSnapshot);
    dashboardApiMocks.getDashboardWidget
      .mockRejectedValueOnce(new Error('Widget refresh failed'))
      .mockResolvedValueOnce(recoveredWidget);
    const wrapper = mountPage();
    const workbench = wrapper.getComponent({ name: 'DashboardWorkbenchStub' });
    await flushPromises();
    expect(workbench.props('ready')).toBe(true);
    expect(wrapper.get('.attention-item').attributes('data-evidence')).toBe('source-failed');

    await wrapper.get('.refresh-page').trigger('click');
    await flushPromises();
    expect(workbench.props('errorMessage')).toBe('Overview refresh failed');

    await wrapper.get('.attention-item').trigger('click');
    await flushPromises();
    expect(workbench.props('errorMessage')).toBe('Overview refresh failed');
    expect(wrapper.get('.attention-item').attributes('data-evidence')).toBe('source-failed');

    await wrapper.get('.attention-item').trigger('click');
    await flushPromises();
    expect(workbench.props('errorMessage')).toBe('Overview refresh failed');
    expect(wrapper.find('.attention-item').exists()).toBe(false);

    await wrapper.get('.refresh-page').trigger('click');
    await flushPromises();
    expect(workbench.props('errorMessage')).toBe('');
    expect(wrapper.get('.attention-item').attributes('data-evidence')).toBe('source-failed');
    wrapper.unmount();
  });

  it('retains a widget retry across remounts until a new overview succeeds', async () => {
    const recoveredWidget = summaryResponse().widgets[0]!;
    const failedSnapshot = summaryResponse({ widgets: [{ ...recoveredWidget, status: 'error' }] });
    dashboardApiMocks.getDashboardSummary.mockResolvedValue(failedSnapshot);
    dashboardApiMocks.getDashboardWidget.mockResolvedValueOnce(recoveredWidget);
    const first = mountPage();
    await flushPromises();
    const overviewUpdatedAt = queryClient.getQueryState(['dashboard', 'summary'])?.dataUpdatedAt;

    await first.get('.attention-item').trigger('click');
    await flushPromises();
    expect(first.find('.attention-item').exists()).toBe(false);
    expect(queryClient.getQueryState(['dashboard', 'summary'])?.dataUpdatedAt).toBe(overviewUpdatedAt);
    first.unmount();

    const second = mountPage();
    await flushPromises();
    expect(dashboardApiMocks.getDashboardSummary).toHaveBeenCalledTimes(1);
    expect(second.find('.attention-item').exists()).toBe(false);

    await second.get('.refresh-page').trigger('click');
    await flushPromises();
    expect(dashboardApiMocks.getDashboardSummary).toHaveBeenCalledTimes(2);
    expect(second.get('.attention-item').attributes('data-evidence')).toBe('source-failed');
    second.unmount();
  });

  it('discards a widget retry that settles after a new overview succeeds', async () => {
    const recoveredWidget = summaryResponse().widgets[0]!;
    const failedSnapshot = summaryResponse({ widgets: [{ ...recoveredWidget, status: 'error' }] });
    const retry = Promise.withResolvers<typeof recoveredWidget>();
    dashboardApiMocks.getDashboardSummary.mockResolvedValue(failedSnapshot);
    dashboardApiMocks.getDashboardWidget.mockReturnValueOnce(retry.promise);
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('.attention-item').trigger('click');
    await flushPromises();
    expect(dashboardApiMocks.getDashboardWidget).toHaveBeenCalledTimes(1);

    await wrapper.get('.refresh-page').trigger('click');
    await flushPromises();
    expect(dashboardApiMocks.getDashboardSummary).toHaveBeenCalledTimes(2);
    retry.resolve(recoveredWidget);
    await flushPromises();

    expect(wrapper.get('.attention-item').attributes('data-evidence')).toBe('source-failed');
    wrapper.unmount();
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
    usePermissionStore().setBootstrapSnapshot(bootstrapSnapshot([]));
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
    expect(window.localStorage.getItem(DASHBOARD_QUICK_ACTION_STORAGE_KEY.ROUTE_USAGE)).toContain('/feature-0');
  });

  it('opens contextual resource actions without changing quick-entry usage ranking', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('.resource-item').trigger('click');

    expect(routerMocks.push).toHaveBeenCalledWith(CONTAINER_ROUTE_PATH.LIST);
    expect(window.localStorage.getItem(DASHBOARD_QUICK_ACTION_STORAGE_KEY.ROUTE_USAGE)).toBeNull();
  });
});
