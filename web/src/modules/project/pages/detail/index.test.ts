import { flushPromises, mount, shallowMount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, ref } from 'vue';

import ProjectDetailPage from './index.vue';

const containerApiMocks = vi.hoisted(() => ({
  getContainerEvents: vi.fn(),
  getContainerLogs: vi.fn(),
}));

const projectApiMocks = vi.hoisted(() => ({
  getProject: vi.fn(),
  getProjectConfiguration: vi.fn(),
  getProjectConfigurationFile: vi.fn(),
  getProjectConfigurationPreview: vi.fn(),
  getProjectOverview: vi.fn(),
  getProjectServices: vi.fn(),
  postProjectConfigurationDiff: vi.fn(),
  postProjectConfigurationValidate: vi.fn(),
  postProjectDeploy: vi.fn(),
  postProjectDestroy: vi.fn(),
  postProjectRedeploy: vi.fn(),
  postProjectStop: vi.fn(),
  postProjectRestart: vi.fn(),
  postProjectUnregister: vi.fn(),
  postProjectUp: vi.fn(),
}));

const tabsRouterStoreMock = vi.hoisted(() => ({
  tabRouterList: [
    {
      fullPath: '/ops/projects/7',
      path: '/ops/projects/7',
      tabKey: '/ops/projects/7',
      title: { 'en-US': 'Project Detail', 'zh-CN': '项目详情' },
    },
  ],
}));

const routerMocks = vi.hoisted(() => ({
  push: vi.fn(),
  replace: vi.fn(),
  resolve: vi.fn((target: { params?: { id?: string } }) => ({
    fullPath: `/ops/containers/${target.params?.id ?? ''}`,
    path: `/ops/containers/${target.params?.id ?? ''}`,
  })),
}));

const detailMessages = {
  'project.detail.actions.copyPath': 'Copy Path',
  'project.detail.actions.destroy': 'Destroy',
  'project.detail.actions.stop': 'Stop',
  'project.detail.actions.restart': 'Restart',
  'project.detail.actions.unregister': 'Unregister',
  'project.detail.actions.up': 'Up',
  'project.detail.description': 'Project detail description',
  'project.detail.eyebrow': 'Compose Project',
  'project.detail.overview.allHealthy': 'No unhealthy containers',
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
  'project.detail.overview.lastUpdated': 'Last Updated',
  'project.detail.overview.memoryUsage': 'Memory',
  'project.detail.overview.metricsCoverage': 'Metrics Coverage',
  'project.detail.overview.networkRealtime': 'Realtime',
  'project.detail.overview.networkRx': 'Downstream',
  'project.detail.overview.networkTitle': 'Network I/O',
  'project.detail.overview.networkTotalRx': 'Total Rx',
  'project.detail.overview.networkTotalTx': 'Total Tx',
  'project.detail.overview.networkTx': 'Upstream',
  'project.detail.overview.notCollected': 'Not Collected',
  'project.detail.overview.realtimeLabel': 'Current Value',
  'project.detail.overview.resourceTitle': 'Resource Usage',
  'project.detail.overview.restartClear': 'No restarts',
  'project.detail.overview.restartCount': 'Restart Count',
  'project.detail.overview.restartWarning': 'Restart Warning',
  'project.detail.overview.runtimeMembers': 'Runtime Members',
  'project.detail.overview.runtimeStatus': 'Runtime Status',
  'project.detail.overview.serviceCount': 'Service Count',
  'project.detail.overview.serviceHealthHealthy': 'Healthy',
  'project.detail.overview.serviceHealthUnknown': 'Unknown',
  'project.detail.overview.serviceHealthAttention': 'Attention',
  'project.detail.overview.serviceStatusDegraded': 'Degraded',
  'project.detail.overview.serviceStatusRunning': 'Running',
  'project.detail.overview.serviceStatusStopped': 'Stopped',
  'project.detail.overview.startingHint': 'Starting',
  'project.detail.overview.topologyCaption': 'Topology',
  'project.detail.overview.topologyTitle': 'Running Members',
  'project.detail.overview.unhealthyHint': 'Unhealthy Hint',
  'project.detail.logs.allLevels': 'All Levels',
  'project.detail.logs.authorityBackendPlanned': 'Backend planned',
  'project.detail.logs.authorityFrontendFanout': 'Frontend fan-out',
  'project.detail.logs.autoScroll': 'Auto Scroll',
  'project.detail.logs.autoScrollTooltip': 'Follow latest log',
  'project.detail.logs.basicInfo': 'Basic Info',
  'project.detail.logs.clear': 'Clear',
  'project.detail.logs.collapseDetail': 'Collapse',
  'project.detail.logs.copy': 'Copy',
  'project.detail.logs.copyError': 'Copy failed',
  'project.detail.logs.copyJson': 'Copy JSON',
  'project.detail.logs.copyLine': 'Copy Line',
  'project.detail.logs.copyMessage': 'Copy Message',
  'project.detail.logs.copySuccess': 'Copied',
  'project.detail.logs.detailTitle': 'Log Detail',
  'project.detail.logs.download': 'Download',
  'project.detail.logs.empty': 'No Logs',
  'project.detail.logs.emptyDescription': 'No logs yet',
  'project.detail.logs.exitFullscreen': 'Exit Fullscreen',
  'project.detail.logs.fullscreen': 'Fullscreen',
  'project.detail.logs.importantFields': 'Important Fields',
  'project.detail.logs.jumpBottom': 'Jump Bottom',
  'project.detail.logs.level': 'Level',
  'project.detail.logs.levelFilter': 'Level',
  'project.detail.logs.loadFailed': 'Load failed',
  'project.detail.logs.logCount': 'Log Count',
  'project.detail.logs.matchCount': 'Matches',
  'project.detail.logs.memberCount': 'Sources',
  'project.detail.logs.message': 'Message',
  'project.detail.logs.metadata': 'Metadata',
  'project.detail.logs.operation': 'Operation',
  'project.detail.logs.pause': 'Pause',
  'project.detail.logs.raw': 'Raw',
  'project.detail.logs.refreshAction': 'Refresh Logs',
  'project.detail.logs.resize': 'Resize',
  'project.detail.logs.resume': 'Resume',
  'project.detail.logs.searchPlaceholder': 'Search logs',
  'project.detail.logs.sinceLabel': 'Since',
  'project.detail.logs.source': 'Source',
  'project.detail.logs.stderr': 'STDERR',
  'project.detail.logs.stdout': 'STDOUT',
  'project.detail.logs.stream': 'Stream',
  'project.detail.logs.summary': 'Summary',
  'project.detail.logs.tailLabel': 'Tail',
  'project.detail.logs.time': 'Time',
  'project.detail.logs.title': 'Project Logs',
  'project.detail.logs.truncated': 'Truncated',
  'project.detail.logs.viewDetail': 'View Detail',
  'project.detail.logs.wrap': 'Wrap',
  'project.detail.runtime.authorityTitle': 'Runtime Boundary',
  'project.detail.runtime.canonicalName': 'Canonical Project Name',
  'project.detail.runtime.description': 'Runtime description',
  'project.detail.runtime.driftStatus': 'Drift Status',
  'project.detail.runtime.hostScope': 'Host Scope',
  'project.detail.runtime.lastRefreshAt': 'Last Project Refresh',
  'project.detail.runtime.membersTitle': 'Runtime Members',
  'project.detail.runtime.nameSource': 'Name Source',
  'project.detail.runtime.refreshStatus': 'Refresh Status',
  'project.detail.runtime.runningMembers': 'Running Members',
  'project.detail.runtime.runtimeStatus': 'Project Status',
  'project.detail.runtime.serviceCount': 'Service Count',
  'project.detail.runtime.statusTitle': 'Runtime Status',
  'project.detail.runtime.title': 'Runtime',
  'project.detail.runtime.totalMembers': 'Total Members',
  'project.detail.runtime.workingDirectory': 'Working Directory',
  'project.detail.summary.canonicalName': 'Canonical Project Name',
  'project.detail.summary.composeFiles': 'Compose Files',
  'project.detail.summary.configHash': 'Config Hash',
  'project.detail.summary.configurationTitle': 'Configuration',
  'project.detail.summary.discoveryTitle': 'Discovery',
  'project.detail.summary.envFiles': 'Env Files',
  'project.detail.summary.hostScope': 'Host Scope',
  'project.detail.summary.lastRefreshAt': 'Last Project Refresh',
  'project.detail.summary.nameSource': 'Name Source',
  'project.detail.summary.runningMembers': 'Running Members',
  'project.detail.summary.runtimeMembers': 'Runtime Members',
  'project.detail.summary.services': 'Services',
  'project.detail.summary.status': 'Status',
  'project.detail.summary.summaryTitle': 'Summary',
  'project.detail.summary.workingDirectory': 'Working Directory',
  'project.detail.tabs.configuration': 'Configuration',
  'project.detail.tabs.containers': 'Containers',
  'project.detail.tabs.logs': 'Logs',
  'project.detail.tabs.networks': 'Networks',
  'project.detail.tabs.overview': 'Overview',
  'project.detail.tabs.runtime': 'Runtime',
  'project.detail.tabs.services': 'Services',
  'project.detail.tabs.volumes': 'Volumes',
  'project.list.refresh': 'Refresh',
  'project.list.retry': 'Retry',
} as const;

function slotStub(name: string) {
  return defineComponent({
    name,
    props: {
      label: { type: String, default: '' },
      title: { type: String, default: '' },
      value: { type: [String, Number, Array], default: undefined },
    },
    emits: ['change', 'update:value', 'click'],
    setup(props, { slots }) {
      return () =>
        h('div', { 'data-stub': name }, [
          props.title ? h('div', { 'data-title': name }, props.title) : null,
          props.label ? h('div', { 'data-label': name }, props.label) : null,
          slots.actions?.(),
          slots.meta?.(),
          slots.operation?.(),
          slots.default?.(),
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
  setup(props, { emit, slots, attrs }) {
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

const TTableStub = defineComponent({
  name: 'TTableStub',
  setup(_props, { slots }) {
    return () => h('div', { 'data-stub': 'TTable' }, [slots.empty?.()]);
  },
});

function mountRuntimePage() {
  return mount(ProjectDetailPage, {
    global: {
      renderStubDefaultSlot: true,
      stubs: {
        'management-page-header': slotStub('ManagementPageHeader'),
        'project-file-editor': slotStub('ProjectFileEditor'),
        'project-resources-section': slotStub('ProjectResourcesSection'),
        'refresh-icon': true,
        't-alert': slotStub('TAlert'),
        't-button': TButtonStub,
        't-card': slotStub('TCard'),
        't-collapse': slotStub('TCollapse'),
        't-collapse-panel': slotStub('TCollapsePanel'),
        't-descriptions': slotStub('TDescriptions'),
        't-descriptions-item': slotStub('TDescriptionsItem'),
        't-empty': slotStub('TEmpty'),
        't-input': slotStub('TInput'),
        't-progress': slotStub('TProgress'),
        't-loading': slotStub('TLoading'),
        't-space': slotStub('TSpace'),
        't-tab-panel': slotStub('TTabPanel'),
        't-tabs': slotStub('TTabs'),
        't-table': TTableStub,
        't-tag': slotStub('TTag'),
      },
    },
  });
}

vi.mock('@/modules/container/api/container', () => ({
  getContainerEvents: containerApiMocks.getContainerEvents,
  getContainerLogs: containerApiMocks.getContainerLogs,
}));

vi.mock('../../api/project', () => ({
  getProject: projectApiMocks.getProject,
  getProjectConfiguration: projectApiMocks.getProjectConfiguration,
  getProjectConfigurationFile: projectApiMocks.getProjectConfigurationFile,
  getProjectConfigurationPreview: projectApiMocks.getProjectConfigurationPreview,
  getProjectOverview: projectApiMocks.getProjectOverview,
  getProjectServices: projectApiMocks.getProjectServices,
  postProjectConfigurationDiff: projectApiMocks.postProjectConfigurationDiff,
  postProjectConfigurationValidate: projectApiMocks.postProjectConfigurationValidate,
  postProjectDeploy: projectApiMocks.postProjectDeploy,
  postProjectDestroy: projectApiMocks.postProjectDestroy,
  postProjectRedeploy: projectApiMocks.postProjectRedeploy,
  postProjectStop: projectApiMocks.postProjectStop,
  postProjectRestart: projectApiMocks.postProjectRestart,
  postProjectUnregister: projectApiMocks.postProjectUnregister,
  postProjectUp: projectApiMocks.postProjectUp,
}));

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>();
  const locale = ref('en-US');
  return {
    ...actual,
    useI18n: () => ({
      locale,
      t: (key: string) => detailMessages[key as keyof typeof detailMessages] ?? key,
    }),
  };
});

vi.mock('vue-router', () => ({
  useRoute: () => ({
    fullPath: '/ops/projects/7',
    params: { id: '7' },
    path: '/ops/projects/7',
    query: {},
  }),
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
      return () => h('div', { 'data-stub': 'LogViewer' }, [slots.default?.()]);
    },
  }),
  copyText: vi.fn(),
  formatBytes: (value?: number | null) => (typeof value === 'number' ? `${value} B` : '-'),
  normalizeStructuredLogEntry: (value: { line?: string; occurred_at?: string; stream?: string }) =>
    value?.line
      ? {
          line: value.line,
          occurredAt: value.occurred_at ?? '',
          stream: value.stream === 'stderr' ? 'stderr' : 'stdout',
        }
      : null,
  formatPercent: (value?: number | null) => (typeof value === 'number' ? `${value.toFixed(1)}%` : '-'),
  toProgressPercent: (value?: number | null) => (typeof value === 'number' ? Math.max(0, Math.min(100, value)) : 0),
}));

vi.mock('@/utils/logger', () => ({
  createLogger: () => ({
    warn: vi.fn(),
    error: vi.fn(),
  }),
}));

vi.mock('../../shared/display', () => ({
  formatProjectTime: (_locale: string, value?: string | null) => value || '-',
  projectDriftStatusLabel: () => 'in-sync',
  projectDriftStatusTheme: () => 'success',
  projectLifecycleActionVisibility: (status?: string | null) => ({
    up: status === 'stopped' || status === 'unknown' || status === 'transitioning' || !status,
    redeploy: true,
    stop:
      status === 'running' || status === 'degraded' || status === 'unknown' || status === 'transitioning' || !status,
    restart: true,
    unregister: true,
  }),
  projectOwnershipModeLabel: () => 'external',
  projectRefreshStatusLabel: () => 'success',
  projectRefreshStatusTheme: () => 'success',
  projectRuntimeStatusLabel: () => 'running',
  projectRuntimeStatusTheme: () => 'success',
}));

describe('Project detail page', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    tabsRouterStoreMock.tabRouterList = [
      {
        fullPath: '/ops/projects/7',
        path: '/ops/projects/7',
        tabKey: '/ops/projects/7',
        title: { 'en-US': 'Project Detail', 'zh-CN': '项目详情' },
      },
    ];
    projectApiMocks.getProject.mockResolvedValue({
      activity_authority: 'frontend-fanout',
      canonical_project_name: 'compose-demo',
      canonical_project_name_source: 'explicit',
      compose_files: [],
      container_counts: { running: 1, stopped: 0, transitioning: 0, issue: 0, total: 1 },
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
      runtime_status: 'running',
      service_count: 1,
      working_directory: '/srv/compose-demo',
    });
    projectApiMocks.getProjectConfiguration.mockResolvedValue({
      compose_files: [],
      diagnostics_summary: [],
      drift_status: 'clean',
      env_files: [],
      last_refresh_status: 'success',
      ownership_mode: 'external',
    });
    projectApiMocks.getProjectConfigurationPreview.mockResolvedValue(null);
    projectApiMocks.getProjectOverview.mockResolvedValue({
      canonical_project_name: 'compose-demo',
      collected_at: '2026-07-03T10:00:00Z',
      health: {
        healthy_service_count: 1,
        healthy_container_count: 1,
        networks_count: 1,
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
        stats_available_container_count: 1,
        tx_bytes: 32,
      },
      services: [
        {
          container_count: 1,
          cpu_percent: 12.5,
          health: 'healthy',
          healthy_container_count: 1,
          image: 'demo:latest',
          issue_count: 0,
          memory_limit_bytes: 512,
          memory_usage_bytes: 128,
          restart_count: 0,
          running_count: 1,
          service_name: 'app',
          starting_container_count: 0,
          stats_available: true,
          stats_available_container_count: 1,
          status: 'running',
          stopped_count: 0,
          transitioning_count: 0,
          unhealthy_container_count: 0,
        },
      ],
    });
    projectApiMocks.getProjectServices.mockResolvedValue({
      items: [
        {
          build_context: null,
          container_members: [{ container_id: 'container-1', container_name: 'compose-demo-1', state: 'running' }],
          declared_networks: [],
          declared_ports: [],
          declared_volumes: [],
          image: 'demo:latest',
          running_count: 1,
          service_name: 'app',
          stopped_count: 0,
        },
      ],
    });
    containerApiMocks.getContainerEvents.mockResolvedValue({
      items: [],
    });
    containerApiMocks.getContainerLogs.mockResolvedValue({
      entries: [],
    });
  });

  it('loads detail and overview data on initial refresh without preloading project logs', async () => {
    shallowMount(ProjectDetailPage);
    await flushPromises();

    expect(projectApiMocks.getProject).toHaveBeenCalledWith(7);
    expect(projectApiMocks.getProjectConfiguration).toHaveBeenCalledWith(7);
    expect(projectApiMocks.getProjectConfigurationPreview).toHaveBeenCalledWith(7);
    expect(projectApiMocks.getProjectOverview).toHaveBeenCalledWith(7);
    expect(projectApiMocks.getProjectServices).toHaveBeenCalledTimes(1);
    expect(projectApiMocks.getProjectServices).toHaveBeenCalledWith(7);
    expect(containerApiMocks.getContainerEvents).not.toHaveBeenCalled();
    expect(containerApiMocks.getContainerLogs).not.toHaveBeenCalled();
  });

  it('keeps the detail record when loading services fails during refresh', async () => {
    projectApiMocks.getProjectServices.mockRejectedValueOnce(new Error('service list failed'));

    const wrapper = mountRuntimePage();
    await flushPromises();

    expect(projectApiMocks.getProject).toHaveBeenCalledWith(7);
    expect(wrapper.text()).toContain('Compose Demo');

    wrapper.unmount();
  });

  it('uses the same lifecycle visibility rules as the list page', async () => {
    const wrapper = mountRuntimePage();
    await flushPromises();

    expect(wrapper.find('[data-testid="project-detail-action-up"]').exists()).toBe(false);
    expect(wrapper.find('[data-testid="project-detail-action-stop"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="project-detail-action-restart"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="project-detail-action-unregister"]').exists()).toBe(true);

    wrapper.unmount();
  });

  it('shows stopped and unknown lifecycle visibility variants and project refresh labels', async () => {
    projectApiMocks.getProject.mockResolvedValueOnce({
      activity_authority: 'frontend-fanout',
      canonical_project_name: 'compose-demo',
      canonical_project_name_source: 'explicit',
      compose_files: [],
      container_counts: { running: 0, stopped: 1, transitioning: 0, issue: 0, total: 1 },
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
      runtime_status: 'stopped',
      service_count: 1,
      working_directory: '/srv/compose-demo',
    });

    let wrapper = mountRuntimePage();
    await flushPromises();

    expect(wrapper.find('[data-testid="project-detail-action-up"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="project-detail-action-stop"]').exists()).toBe(false);
    expect(wrapper.find('[data-testid="project-detail-action-restart"]').exists()).toBe(true);
    expect(wrapper.text()).toContain('Last Collected');

    wrapper.unmount();

    projectApiMocks.getProject.mockResolvedValueOnce({
      activity_authority: 'frontend-fanout',
      canonical_project_name: 'compose-demo',
      canonical_project_name_source: 'explicit',
      compose_files: [],
      container_counts: { running: 0, stopped: 0, transitioning: 0, issue: 0, total: 0 },
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
      runtime_status: 'unknown',
      service_count: 0,
      working_directory: '/srv/compose-demo',
    });

    wrapper = mountRuntimePage();
    await flushPromises();

    expect(wrapper.find('[data-testid="project-detail-action-up"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="project-detail-action-stop"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="project-detail-action-restart"]').exists()).toBe(true);

    wrapper.unmount();
  });
});
