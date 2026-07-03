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
  getProjectServices: vi.fn(),
  postProjectConfigurationDiff: vi.fn(),
  postProjectConfigurationValidate: vi.fn(),
  postProjectDeploy: vi.fn(),
  postProjectDown: vi.fn(),
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
  'project.detail.actions.down': 'Down',
  'project.detail.actions.restart': 'Restart',
  'project.detail.actions.unregister': 'Unregister',
  'project.detail.actions.up': 'Up',
  'project.detail.description': 'Project detail description',
  'project.detail.eyebrow': 'Compose Project',
  'project.detail.refresh': 'Refresh Snapshot',
  'project.detail.runtime.activityAuthority': 'Activity Authority',
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
  'project.detail.tabs.activity': 'Activity',
  'project.detail.tabs.configuration': 'Configuration',
  'project.detail.tabs.containers': 'Containers',
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
  getProjectServices: projectApiMocks.getProjectServices,
  postProjectConfigurationDiff: projectApiMocks.postProjectConfigurationDiff,
  postProjectConfigurationValidate: projectApiMocks.postProjectConfigurationValidate,
  postProjectDeploy: projectApiMocks.postProjectDeploy,
  postProjectDown: projectApiMocks.postProjectDown,
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
  copyText: vi.fn(),
}));

vi.mock('@/utils/logger', () => ({
  createLogger: () => ({
    error: vi.fn(),
  }),
}));

vi.mock('../../shared/display', () => ({
  formatProjectTime: (_locale: string, value?: string | null) => value || '-',
  projectCanonicalNameSourceLabel: () => 'explicit',
  projectDriftStatusLabel: () => 'in-sync',
  projectDriftStatusTheme: () => 'success',
  projectHostScopeLabel: () => 'local',
  projectLifecycleActionVisibility: (status?: string | null) => ({
    up: status === 'stopped' || status === 'unknown' || status === 'transitioning' || !status,
    down:
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
      drift_status: 'in-sync',
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
      drift_status: 'in-sync',
      env_files: [],
      last_refresh_status: 'success',
      ownership_mode: 'external',
    });
    projectApiMocks.getProjectConfigurationPreview.mockResolvedValue(null);
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

  it('loads activity fan-out on initial detail refresh', async () => {
    shallowMount(ProjectDetailPage);
    await flushPromises();

    expect(projectApiMocks.getProject).toHaveBeenCalledWith(7);
    expect(projectApiMocks.getProjectConfiguration).toHaveBeenCalledWith(7);
    expect(projectApiMocks.getProjectConfigurationPreview).toHaveBeenCalledWith(7);
    expect(projectApiMocks.getProjectServices).toHaveBeenCalledTimes(1);
    expect(projectApiMocks.getProjectServices).toHaveBeenCalledWith(7);
    expect(containerApiMocks.getContainerEvents).toHaveBeenCalledWith('container-1');
    expect(containerApiMocks.getContainerLogs).toHaveBeenCalledWith('container-1', {
      since: '1h',
      stderr: true,
      stdout: true,
      tail: 40,
      timestamps: true,
    });
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
    expect(wrapper.find('[data-testid="project-detail-action-down"]').exists()).toBe(true);
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
      drift_status: 'in-sync',
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
    expect(wrapper.find('[data-testid="project-detail-action-down"]').exists()).toBe(false);
    expect(wrapper.find('[data-testid="project-detail-action-restart"]').exists()).toBe(true);
    expect(wrapper.text()).toContain('Last Project Refresh');

    wrapper.unmount();

    projectApiMocks.getProject.mockResolvedValueOnce({
      activity_authority: 'frontend-fanout',
      canonical_project_name: 'compose-demo',
      canonical_project_name_source: 'explicit',
      compose_files: [],
      container_counts: { running: 0, stopped: 0, transitioning: 0, issue: 0, total: 0 },
      display_name: 'Compose Demo',
      drift_status: 'in-sync',
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
    expect(wrapper.find('[data-testid="project-detail-action-down"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="project-detail-action-restart"]').exists()).toBe(true);

    wrapper.unmount();
  });
});
