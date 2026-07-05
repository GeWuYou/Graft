import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, ref } from 'vue';

import ProjectDetailPage from './index.vue';

const containerApiMocks = vi.hoisted(() => ({
  batchContainerActions: vi.fn(),
  getContainerEvents: vi.fn(),
  getContainerLogs: vi.fn(),
}));

const dialogMocks = vi.hoisted(() => ({
  alert: vi.fn(),
  confirm: vi.fn(),
}));

const messageMocks = vi.hoisted(() => ({
  error: vi.fn(),
  success: vi.fn(),
  warning: vi.fn(),
}));

const notifyMocks = vi.hoisted(() => ({
  warning: vi.fn(),
}));

const projectApiMocks = vi.hoisted(() => ({
  getProject: vi.fn(),
  getProjectConfiguration: vi.fn(),
  getProjectConfigurationFile: vi.fn(),
  getProjectConfigurationPreview: vi.fn(),
  getProjectLogs: vi.fn(),
  getProjectOverview: vi.fn(),
  getProjectServices: vi.fn(),
  postProjectConfigurationDiff: vi.fn(),
  postProjectConfigurationValidate: vi.fn(),
  postProjectDeploy: vi.fn(),
  postProjectDestroy: vi.fn(),
  postProjectRedeploy: vi.fn(),
  postProjectRestart: vi.fn(),
  postProjectStop: vi.fn(),
  postProjectUnregister: vi.fn(),
  postProjectUp: vi.fn(),
  putProjectLifecycleConfiguration: vi.fn(),
}));

const routeState = vi.hoisted(() => ({
  value: {
    fullPath: '/ops/projects/7',
    params: { id: '7' },
    path: '/ops/projects/7',
    query: {} as Record<string, unknown>,
  },
}));

const routerMocks = vi.hoisted(() => ({
  push: vi.fn(),
  replace: vi.fn(),
  resolve: vi.fn((target: { params?: { id?: string }; query?: Record<string, unknown> }) => ({
    fullPath: `/ops/containers/${target.params?.id ?? ''}`,
    name: 'ops:container-detail',
    params: target.params ?? {},
    path: `/ops/containers/${target.params?.id ?? ''}`,
    query: target.query ?? {},
  })),
}));

const tabsRouterStoreMock = vi.hoisted(() => ({
  appendTabRouterList: vi.fn(),
  setActiveTabKey: vi.fn(),
  tabRouterList: [
    {
      fullPath: '/ops/projects/7',
      path: '/ops/projects/7',
      tabKey: '/ops/projects/7',
      title: { 'en-US': 'Project Detail', 'zh-CN': '项目详情' },
    },
  ],
}));

const detailMessages = {
  'components.commonTable.detail': 'Detail',
  'project.detail.actions.destroy': 'Destroy',
  'project.detail.actions.redeploy': 'Redeploy',
  'project.detail.actions.restart': 'Restart',
  'project.detail.actions.stop': 'Stop',
  'project.detail.actions.unregister': 'Unregister',
  'project.detail.actions.up': 'Up',
  'project.detail.configuration.composeFiles': 'Compose Files',
  'project.detail.configuration.driftStatus': 'Drift Status',
  'project.detail.configuration.envFiles': 'Env Files',
  'project.detail.configuration.externalAuthorityHint': 'External authority',
  'project.detail.configuration.managedAuthorityHint': 'Managed authority',
  'project.detail.configuration.ownershipMode': 'Ownership Mode',
  'project.detail.configuration.previewEmpty': 'No Preview',
  'project.detail.configuration.previewTitle': 'Preview',
  'project.detail.configuration.refreshStatus': 'Refresh Status',
  'project.detail.configuration.title': 'Configuration',
  'project.detail.description': 'Project detail description',
  'project.detail.eyebrow': 'Compose Project',
  'project.detail.logs.authorityProjectOwned': 'Project-owned logs',
  'project.detail.logs.loadFailed': 'Load failed',
  'project.detail.overview.actionLabel': 'Action',
  'project.detail.overview.containerSnapshotTitle': 'Container Snapshot',
  'project.detail.overview.cpuUsage': 'CPU',
  'project.detail.overview.diagnosticConfigDrift': 'Config drift',
  'project.detail.overview.diagnosticConfigSynced': 'Config synced',
  'project.detail.overview.diagnosticHealthy': 'Healthy',
  'project.detail.overview.diagnosticRefreshHealthy': 'Refresh healthy',
  'project.detail.overview.diagnosticRefreshWarning': 'Refresh warning',
  'project.detail.overview.diagnosticRestartClear': 'No restart loops',
  'project.detail.overview.diagnosticRestartWarning': 'Restart warning',
  'project.detail.overview.diagnosticUnhealthy': 'Unhealthy',
  'project.detail.overview.diagnosticsTitle': 'Diagnostics',
  'project.detail.overview.lastCollectedAt': 'Last Collected',
  'project.detail.overview.memoryUsage': 'Memory',
  'project.detail.overview.metricsCoverage': 'Metrics Coverage',
  'project.detail.overview.networkRealtime': 'Realtime',
  'project.detail.overview.networkRx': 'Downstream',
  'project.detail.overview.networkTitle': 'Network I/O',
  'project.detail.overview.networkTotalRx': 'Total Rx {value}',
  'project.detail.overview.networkTotalTx': 'Total Tx {value}',
  'project.detail.overview.networkTx': 'Upstream',
  'project.detail.overview.notCollected': 'Not Collected',
  'project.detail.overview.realtimeLabel': 'Current Value',
  'project.detail.overview.resourceTitle': 'Resource Usage',
  'project.detail.overview.serviceHealthAttention': 'Attention',
  'project.detail.overview.serviceHealthHealthy': 'Healthy',
  'project.detail.overview.serviceHealthUnknown': 'Unknown',
  'project.detail.overview.serviceStatusDegraded': 'Degraded',
  'project.detail.overview.serviceStatusRunning': 'Running',
  'project.detail.overview.serviceStatusStopped': 'Stopped',
  'project.detail.overview.viewService': 'View Service',
  'project.detail.services.actions.restart': 'Restart Service',
  'project.detail.services.actions.start': 'Start Service',
  'project.detail.services.actions.stop': 'Stop Service',
  'project.detail.services.batch.failed': 'Service action failed.',
  'project.detail.services.batch.failureDetailTitle': 'Failure Details',
  'project.detail.services.batch.noFailureDetail': 'No failure detail.',
  'project.detail.services.batch.partialTitle': 'Partial Service Action',
  'project.detail.services.batch.refreshWarning': 'Refresh warning',
  'project.detail.services.batch.success': 'Service action success {count}',
  'project.detail.services.columns.health': 'Health',
  'project.detail.services.columns.image': 'Image',
  'project.detail.services.columns.operation': 'Operation',
  'project.detail.services.columns.ports': 'Ports',
  'project.detail.services.columns.service': 'Service',
  'project.detail.services.columns.status': 'Status',
  'project.detail.services.description': 'Service description',
  'project.detail.services.emptyDescription': 'No services yet',
  'project.detail.services.emptyTitle': 'No Services',
  'project.detail.services.loadFailed': 'Failed to load services.',
  'project.detail.services.refresh': 'Refresh Services',
  'project.detail.services.summary': '{count} Services',
  'project.detail.services.title': 'Services',
  'project.detail.tabs.configuration': 'Configuration',
  'project.detail.tabs.lifecycle': 'Lifecycle',
  'project.detail.tabs.logs': 'Logs',
  'project.detail.tabs.overview': 'Overview',
  'project.detail.tabs.services': 'Services',
  'project.detail.titleFallback': 'Project Detail',
  'project.list.actions.actionFailed': 'Action Failed',
  'project.list.actions.actionSuccess': 'Action Success',
  'project.list.actions.cancel': 'Cancel',
  'project.list.actions.confirm': 'Confirm',
  'project.list.retry': 'Retry',
} as const;

function slotStub(name: string) {
  return defineComponent({
    name,
    props: {
      label: { default: '' },
      title: { default: '' },
      value: { default: undefined },
    },
    emits: ['change', 'click', 'update:value'],
    setup(props, { slots }) {
      return () =>
        h('div', { 'data-stub': name }, [
          props.label ? h('span', props.label as string) : null,
          props.title ? h('span', props.title as string) : null,
          slots.actions?.(),
          slots.meta?.(),
          slots.operation?.(),
          slots.default?.(),
          slots.toolbar?.(),
          slots.panel?.(),
        ]);
    },
  });
}

const TButtonStub = defineComponent({
  name: 'TButtonStub',
  props: {
    disabled: { type: Boolean, default: false },
    loading: { type: Boolean, default: false },
  },
  emits: ['click'],
  setup(props, { attrs, emit, slots }) {
    return () =>
      h(
        'button',
        {
          ...attrs,
          disabled: props.disabled,
          'data-loading': props.loading ? 'true' : 'false',
          onClick: (event: MouseEvent) => emit('click', event),
        },
        slots.default?.(),
      );
  },
});

const ManagementPagedTableStub = defineComponent({
  name: 'ManagementPagedTable',
  props: {
    columns: { type: Array, default: () => [] },
    paginationVisible: { type: Boolean, default: true },
    rows: { type: Array, default: () => [] },
    summary: { type: String, default: '' },
  },
  setup(props, { slots }) {
    return () =>
      h('div', { 'data-stub': 'ManagementPagedTable' }, [
        h('div', { 'data-columns': JSON.stringify(props.columns) }),
        h('div', { 'data-pagination-visible': String(props.paginationVisible) }),
        h('div', { 'data-summary': props.summary }),
        slots.toolbar?.(),
        ...(props.rows as Array<Record<string, unknown>>).map((row) =>
          h('div', { key: String(row.service_name), 'data-row': String(row.service_name) }, [
            slots.name?.({ row }),
            slots.status?.({ row }),
            slots.health?.({ row }),
            slots.ports?.({ row }),
            slots.operation?.({ row }),
          ]),
        ),
      ]);
  },
});

const TableActionMenuStub = defineComponent({
  name: 'TableActionMenu',
  props: {
    actions: { type: Array, default: () => [] },
  },
  emits: ['action'],
  setup(props, { emit }) {
    return () =>
      h(
        'div',
        { 'data-stub': 'TableActionMenu' },
        (props.actions as Array<{ disabled?: boolean; label: string; value: string }>).map((action) =>
          h(
            'button',
            {
              key: action.value,
              'data-action': action.value,
              disabled: action.disabled,
              onClick: () => emit('action', action.value),
            },
            action.label,
          ),
        ),
      );
  },
});

function buildProjectDetail(runtimeStatus: string = 'running') {
  return {
    activity_authority: 'frontend-fanout',
    canonical_project_name: 'compose-demo',
    canonical_project_name_source: 'explicit',
    compose_files: [],
    container_counts: { issue: 0, running: 1, stopped: 0, total: 1, transitioning: 0 },
    display_name: 'Compose Demo',
    drift_status: 'clean',
    env_files: [],
    host_scope: 'local',
    id: 7,
    last_observed_config_hash: null,
    last_refresh_at: '2026-07-03T10:00:00Z',
    last_refresh_config_hash: null,
    last_refresh_status: 'success',
    ownership_mode: 'external',
    runtime_status: runtimeStatus,
    service_count: 2,
    working_directory: '/srv/compose-demo',
  };
}

function buildProjectOverview() {
  return {
    canonical_project_name: 'compose-demo',
    collected_at: '2026-07-03T10:00:00Z',
    health: {
      healthy_container_count: 2,
      healthy_service_count: 2,
      networks_count: 0,
      restart_count: 0,
      starting_container_count: 0,
      unhealthy_container_count: 0,
      volumes_count: 0,
    },
    project_id: 7,
    resources: {
      cpu_percent: 12.5,
      memory_limit_bytes: 512,
      memory_usage_bytes: 128,
      rx_bytes: 64,
      stats_available: true,
      stats_available_container_count: 2,
      tx_bytes: 32,
    },
    services: [
      {
        container_count: 2,
        cpu_percent: 12.5,
        health: 'healthy',
        healthy_container_count: 2,
        image: 'demo:latest',
        issue_count: 0,
        memory_limit_bytes: 512,
        memory_usage_bytes: 128,
        restart_count: 0,
        running_count: 1,
        service_name: 'app',
        starting_container_count: 0,
        stats_available: true,
        stats_available_container_count: 2,
        status: 'running',
        stopped_count: 1,
        transitioning_count: 0,
        unhealthy_container_count: 0,
      },
      {
        container_count: 1,
        cpu_percent: 0,
        health: 'unknown',
        healthy_container_count: 0,
        image: 'worker:latest',
        issue_count: 0,
        memory_limit_bytes: 0,
        memory_usage_bytes: 0,
        restart_count: 0,
        running_count: 0,
        service_name: 'worker',
        starting_container_count: 0,
        stats_available: false,
        stats_available_container_count: 0,
        status: 'stopped',
        stopped_count: 1,
        transitioning_count: 0,
        unhealthy_container_count: 0,
      },
    ],
  };
}

function buildProjectServices() {
  return {
    items: [
      {
        build_context: null,
        container_members: [
          { container_id: 'container-1', container_name: 'compose-demo-1', state: 'exited' },
          { container_id: 'container-2', container_name: 'compose-demo-2', state: 'running' },
        ],
        declared_networks: [],
        declared_ports: ['127.0.0.1:8080:8080', '8443:8443'],
        declared_volumes: [],
        image: 'demo:latest',
        running_count: 1,
        service_name: 'app',
        stopped_count: 1,
      },
      {
        build_context: null,
        container_members: [{ container_id: 'worker-1', container_name: 'worker-1', state: 'created' }],
        declared_networks: [],
        declared_ports: [],
        declared_volumes: [],
        image: 'worker:latest',
        running_count: 0,
        service_name: 'worker',
        stopped_count: 1,
      },
    ],
  };
}

function mountPage() {
  return mount(ProjectDetailPage, {
    global: {
      renderStubDefaultSlot: true,
      stubs: {
        ManagementPageHeader: slotStub('ManagementPageHeader'),
        ManagementPagedTable: ManagementPagedTableStub,
        ProjectFileEditor: slotStub('ProjectFileEditor'),
        TableActionMenu: TableActionMenuStub,
        'management-page-header': slotStub('management-page-header'),
        'management-paged-table': ManagementPagedTableStub,
        'project-file-editor': slotStub('project-file-editor'),
        'table-action-menu': TableActionMenuStub,
        't-alert': slotStub('TAlert'),
        't-button': TButtonStub,
        't-card': slotStub('TCard'),
        't-collapse': slotStub('TCollapse'),
        't-collapse-panel': slotStub('TCollapsePanel'),
        't-descriptions': slotStub('TDescriptions'),
        't-descriptions-item': slotStub('TDescriptionsItem'),
        't-empty': slotStub('TEmpty'),
        't-input': slotStub('TInput'),
        't-loading': slotStub('TLoading'),
        't-progress': slotStub('TProgress'),
        't-space': slotStub('TSpace'),
        't-tab-panel': slotStub('TTabPanel'),
        't-tabs': slotStub('TTabs'),
        't-tag': slotStub('TTag'),
      },
    },
  });
}

vi.mock('@/modules/container/api/container', () => ({
  batchContainerActions: containerApiMocks.batchContainerActions,
  getContainerEvents: containerApiMocks.getContainerEvents,
  getContainerLogs: containerApiMocks.getContainerLogs,
}));

vi.mock('../../api/project', () => ({
  getProject: projectApiMocks.getProject,
  getProjectConfiguration: projectApiMocks.getProjectConfiguration,
  getProjectConfigurationFile: projectApiMocks.getProjectConfigurationFile,
  getProjectConfigurationPreview: projectApiMocks.getProjectConfigurationPreview,
  getProjectLogs: projectApiMocks.getProjectLogs,
  getProjectOverview: projectApiMocks.getProjectOverview,
  getProjectServices: projectApiMocks.getProjectServices,
  postProjectConfigurationDiff: projectApiMocks.postProjectConfigurationDiff,
  postProjectConfigurationValidate: projectApiMocks.postProjectConfigurationValidate,
  postProjectDeploy: projectApiMocks.postProjectDeploy,
  postProjectDestroy: projectApiMocks.postProjectDestroy,
  postProjectRedeploy: projectApiMocks.postProjectRedeploy,
  postProjectRestart: projectApiMocks.postProjectRestart,
  postProjectStop: projectApiMocks.postProjectStop,
  postProjectUnregister: projectApiMocks.postProjectUnregister,
  postProjectUp: projectApiMocks.postProjectUp,
  putProjectLifecycleConfiguration: projectApiMocks.putProjectLifecycleConfiguration,
}));

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>();
  const locale = ref('en-US');
  return {
    ...actual,
    useI18n: () => ({
      locale,
      t: (key: string, params?: Record<string, unknown>) => {
        const template = detailMessages[key as keyof typeof detailMessages] ?? key;
        return typeof template === 'string'
          ? template.replace(/\{(\w+)\}/g, (_, token) => String(params?.[token] ?? `{${token}}`))
          : key;
      },
    }),
  };
});

vi.mock('vue-router', () => ({
  useRoute: () => routeState.value,
  useRouter: () => routerMocks,
}));

vi.mock('@/store/modules/tabs-router', () => ({
  useTabsRouterStore: () => tabsRouterStoreMock,
}));

vi.mock('@/shared/localized-api-error', () => ({
  resolveLocalizedErrorMessage: (_t: unknown, _error: unknown, fallback: string) => fallback,
}));

vi.mock('@/shared/observability', () => ({
  LogViewer: defineComponent({
    name: 'LogViewerStub',
    setup(_props, { slots }) {
      return () => h('div', { 'data-stub': 'LogViewer' }, slots.default?.());
    },
  }),
  copyText: vi.fn(),
  formatBytes: (value?: number | null) => (typeof value === 'number' ? `${value} B` : '-'),
  formatPercent: (value?: number | null) => (typeof value === 'number' ? `${value.toFixed(1)}%` : '-'),
  normalizeStructuredLogEntry: (value: { line?: string; occurred_at?: string; stream?: string }) =>
    value?.line
      ? {
          line: value.line,
          occurredAt: value.occurred_at ?? '',
          stream: value.stream === 'stderr' ? 'stderr' : 'stdout',
        }
      : null,
  toProgressPercent: (value?: number | null) => (typeof value === 'number' ? Math.max(0, Math.min(100, value)) : 0),
}));

vi.mock('@/shared/realtime', () => ({
  openRealtimeTopicSocket: vi.fn(() => ({ close: vi.fn() })),
}));

vi.mock('@/utils/logger', () => ({
  createLogger: () => ({
    error: vi.fn(),
    warn: vi.fn(),
  }),
}));

vi.mock('tdesign-vue-next/es/dialog', () => ({
  DialogPlugin: {
    alert: dialogMocks.alert,
    confirm: dialogMocks.confirm,
  },
}));

vi.mock('../../shared/display', () => ({
  formatProjectTime: (_locale: string, value?: string | null) => value || '-',
  projectDriftStatusLabel: () => 'in-sync',
  projectDriftStatusTheme: () => 'success',
  projectLifecycleActionVisibility: (status?: string | null) => ({
    redeploy: true,
    restart: true,
    stop:
      status === 'running' || status === 'degraded' || status === 'unknown' || status === 'transitioning' || !status,
    unregister: true,
    up: status === 'stopped' || status === 'unknown' || status === 'transitioning' || !status,
  }),
  projectOwnershipModeLabel: () => 'external',
  projectRefreshStatusLabel: () => 'success',
  projectRefreshStatusTheme: () => 'success',
  projectRuntimeStatusLabel: (value?: string | null) => value || 'running',
  projectRuntimeStatusTheme: () => 'success',
}));

vi.mock('tdesign-vue-next/es/message', () => ({
  MessagePlugin: messageMocks,
}));

vi.mock('tdesign-vue-next/es/notification', () => ({
  NotifyPlugin: notifyMocks,
}));

describe('Project detail service tab', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    routeState.value.query = {};
    tabsRouterStoreMock.tabRouterList = [
      {
        fullPath: '/ops/projects/7',
        path: '/ops/projects/7',
        tabKey: '/ops/projects/7',
        title: { 'en-US': 'Project Detail', 'zh-CN': '项目详情' },
      },
    ];
    tabsRouterStoreMock.setActiveTabKey.mockReset();
    projectApiMocks.getProject.mockResolvedValue(buildProjectDetail());
    projectApiMocks.getProjectConfiguration.mockResolvedValue({
      compose_files: [],
      diagnostics_summary: [],
      drift_status: 'clean',
      env_files: [],
      last_refresh_status: 'success',
      ownership_mode: 'external',
    });
    projectApiMocks.getProjectConfigurationPreview.mockResolvedValue(null);
    projectApiMocks.getProjectLogs.mockResolvedValue({ entries: [] });
    projectApiMocks.getProjectOverview.mockResolvedValue(buildProjectOverview());
    projectApiMocks.getProjectServices.mockResolvedValue(buildProjectServices());
    containerApiMocks.batchContainerActions.mockResolvedValue({
      failed_count: 0,
      items: [],
      success_count: 2,
    });
    containerApiMocks.getContainerEvents.mockResolvedValue({ items: [] });
    containerApiMocks.getContainerLogs.mockResolvedValue({ entries: [] });
  });

  it('normalizes legacy container tabs to services on mount', async () => {
    routeState.value.query = { tab: 'containers' };

    mountPage();
    await flushPromises();

    expect(routerMocks.replace).toHaveBeenCalledWith({
      query: {
        tab: 'services',
      },
    });
  });

  it('renders compact service columns without pagination', async () => {
    const wrapper = mountPage();
    await flushPromises();

    const table = wrapper.findComponent(ManagementPagedTableStub);
    const columns = table.props('columns') as Array<{ title: string }>;

    expect(columns.map((column) => column.title)).toEqual([
      'Service',
      'Status',
      'Image',
      'Health',
      'Ports',
      'Operation',
    ]);
    expect(table.props('paginationVisible')).toBe(false);
    expect(wrapper.text()).not.toContain('Containers');
    expect(wrapper.text()).not.toContain('Networks');
    expect(wrapper.text()).not.toContain('Volumes');
  });

  it('prefers the first running member when opening service detail', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.find('[data-row="app"] [data-action="detail"]').trigger('click');

    expect(routerMocks.push).toHaveBeenCalledWith(
      expect.objectContaining({
        params: { id: 'container-2' },
        query: { name: 'compose-demo-2' },
      }),
    );
  });

  it('shows mutually exclusive service actions by running state', async () => {
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.find('[data-row="app"] [data-action="stop"]').exists()).toBe(true);
    expect(wrapper.find('[data-row="app"] [data-action="restart"]').exists()).toBe(true);
    expect(wrapper.find('[data-row="app"] [data-action="start"]').exists()).toBe(false);

    expect(wrapper.find('[data-row="worker"] [data-action="start"]').exists()).toBe(true);
    expect(wrapper.find('[data-row="worker"] [data-action="stop"]').exists()).toBe(false);
    expect(wrapper.find('[data-row="worker"] [data-action="restart"]').exists()).toBe(false);
  });

  it('runs service actions against all container members and refreshes detail state', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.find('[data-row="app"] [data-action="stop"]').trigger('click');
    await flushPromises();

    expect(containerApiMocks.batchContainerActions).toHaveBeenCalledWith({
      action: 'stop',
      force: false,
      ids: ['container-1', 'container-2'],
    });
    expect(projectApiMocks.getProject).toHaveBeenCalledTimes(2);
    expect(projectApiMocks.getProjectOverview).toHaveBeenCalledTimes(2);
    expect(projectApiMocks.getProjectServices).toHaveBeenCalledTimes(2);
    expect(messageMocks.success).toHaveBeenCalledWith('Service action success 2');
  });
});
