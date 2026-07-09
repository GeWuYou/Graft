import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, ref } from 'vue';

import ProjectDetailPage from './index.vue';

const containerApiMocks = vi.hoisted(() => ({
  batchContainerActions: vi.fn(),
  getContainers: vi.fn(),
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

const realtimeMocks = vi.hoisted(() => ({
  sockets: [] as Array<{
    controller: { close: ReturnType<typeof vi.fn> };
    options: Record<string, any>;
  }>,
}));

const projectApiMocks = vi.hoisted(() => ({
  getProject: vi.fn(),
  getProjectConfiguration: vi.fn(),
  getProjectConfigurationFile: vi.fn(),
  getProjectConfigurationPreview: vi.fn(),
  getProjectLogs: vi.fn(),
  getProjectOverview: vi.fn(),
  getProjectServices: vi.fn(),
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
  resolve: vi.fn((target: { name?: string; params?: { id?: string }; query?: Record<string, unknown> }) =>
    target.name === 'ProjectConfigurationWorkspaceIndex'
      ? {
          fullPath: `/ops/projects/${target.params?.id ?? ''}/configuration`,
          name: target.name,
          params: target.params ?? {},
          path: `/ops/projects/${target.params?.id ?? ''}/configuration`,
          query: target.query ?? {},
        }
      : {
          fullPath: `/ops/containers/${target.params?.id ?? ''}`,
          name: 'ops:container-detail',
          params: target.params ?? {},
          path: `/ops/containers/${target.params?.id ?? ''}`,
          query: target.query ?? {},
        },
  ),
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
  'project.detail.actions.openConfigurationWorkspace': 'Configuration Workspace',
  'project.detail.actions.redeploy': 'Redeploy',
  'project.detail.actions.restart': 'Restart',
  'project.detail.actions.stop': 'Stop',
  'project.detail.actions.unregister': 'Unregister',
  'project.detail.actions.up': 'Up',
  'project.detail.lifecycle.copyCommand': 'Copy Command',
  'project.detail.lifecycle.copyCommandError': 'Command preview could not be copied.',
  'project.detail.lifecycle.copyCommandSuccess': 'Command preview copied.',
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
  'project.detail.services.actions.restart': 'Restart',
  'project.detail.services.actions.start': 'Start',
  'project.detail.services.actions.stop': 'Stop',
  'project.detail.services.batch.cancelSelection': 'Clear Selection',
  'project.detail.services.batch.confirmRestart': 'Restart the {count} selected services?',
  'project.detail.services.batch.confirmRestartTitle': 'Confirm Batch Service Restart',
  'project.detail.services.batch.confirmStart': 'Start the {count} selected services?',
  'project.detail.services.batch.confirmStartTitle': 'Confirm Batch Service Start',
  'project.detail.services.batch.confirmStop': 'Stop the {count} selected services?',
  'project.detail.services.batch.confirmStopTitle': 'Confirm Batch Service Stop',
  'project.detail.services.batch.failed': 'Service action failed.',
  'project.detail.services.batch.failureDetailTitle': 'Failure Details',
  'project.detail.services.batch.noSelection': 'Select actionable services first.',
  'project.detail.services.batch.noFailureDetail': 'No failure detail.',
  'project.detail.services.batch.partialTitle': 'Partial Service Action',
  'project.detail.services.batch.refreshWarning': 'Refresh warning',
  'project.detail.services.batch.restart': 'Batch Restart',
  'project.detail.services.batch.selected': '{count} Services Selected',
  'project.detail.services.batch.start': 'Batch Start',
  'project.detail.services.batch.stop': 'Batch Stop',
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
  'project.detail.tabs.lifecycle': 'Lifecycle',
  'project.detail.tabs.logs': 'Logs',
  'project.detail.tabs.overview': 'Overview',
  'project.detail.tabs.services': 'Services',
  'project.detail.lifecycle.statusDescription': 'Review authority and command generation inputs.',
  'project.detail.lifecycle.generatedCommandsDescription': 'Review each lifecycle command preview.',
  'project.detail.lifecycle.generatedCommandsDescriptions.up':
    'Launch the current compose project with the saved strategy.',
  'project.detail.lifecycle.generatedCommandsDescriptions.stop':
    'Stop running services without removing project resources.',
  'project.detail.lifecycle.generatedCommandsDescriptions.restart': 'Restart the current compose services in place.',
  'project.detail.lifecycle.generatedCommandsDescriptions.redeploy':
    'Apply the saved redeploy flow, including any optional preparation steps.',
  'project.detail.lifecycle.groups.base.title': 'Base Parameters',
  'project.detail.lifecycle.groups.base.description': 'Base settings',
  'project.detail.lifecycle.groups.redeploy.title': 'Redeploy Strategy',
  'project.detail.lifecycle.groups.redeploy.description': 'Redeploy settings',
  'project.detail.lifecycle.optionDescriptions.downBeforeRedeploy': 'Run docker compose down before redeploy.',
  'project.detail.lifecycle.optionDescriptions.pullBeforeRedeploy': 'Pull images before redeploy.',
  'project.detail.lifecycle.optionDescriptions.buildBeforeUp': 'Add --build before bring-up.',
  'project.detail.lifecycle.optionDescriptions.forceRecreate': 'Add --force-recreate during bring-up.',
  'project.detail.lifecycle.optionDescriptions.removeOrphans': 'Remove orphan containers.',
  'project.detail.lifecycle.optionDescriptions.waitAfterUp': 'Add --wait when bringing services up.',
  'project.detail.lifecycle.optionDescriptions.waitTimeoutSeconds': 'Limit wait time to 1-3600 seconds.',
  'project.detail.lifecycle.optionDescriptions.renewAnonVolumes': 'Do not reuse anonymous volumes.',
  'project.detail.lifecycle.optionDescriptions.pruneImagesAfterRedeploy': 'Prune images after redeploy.',
  'project.detail.lifecycle.renewAnonVolumesWarning': 'Anonymous volumes may be recreated and data may be lost.',
  'project.detail.lifecycle.waitTimeoutValidation': 'Wait timeout must be between 1 and 3600 seconds.',
  'project.detail.titleFallback': 'Project Detail',
  'project.route.configurationWorkspace.title': 'Configuration Workspace',
  'project.list.actions.actionFailed': 'Action Failed',
  'project.list.actions.actionSuccess': 'Action Success',
  'project.list.actions.cancel': 'Cancel',
  'project.list.actions.confirm': 'Confirm',
  'project.list.columns.selection': 'Selection',
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

const TSwitchStub = defineComponent({
  name: 'TSwitchStub',
  props: {
    modelValue: { type: Boolean, default: false },
  },
  emits: ['update:modelValue'],
  setup(props, { attrs, emit }) {
    return () =>
      h(
        'button',
        {
          ...attrs,
          'aria-pressed': props.modelValue ? 'true' : 'false',
          'data-stub': 'TSwitch',
          type: 'button',
          onClick: () => emit('update:modelValue', !props.modelValue),
        },
        props.modelValue ? 'on' : 'off',
      );
  },
});

const TInputNumberStub = defineComponent({
  name: 'TInputNumber',
  props: {
    modelValue: { type: Number, default: 0 },
    min: { type: Number, default: undefined },
    max: { type: Number, default: undefined },
  },
  emits: ['update:modelValue'],
  setup(props, { attrs, emit }) {
    return () =>
      h('input', {
        ...attrs,
        'data-stub': 'TInputNumber',
        max: props.max,
        min: props.min,
        type: 'number',
        value: props.modelValue,
        onInput: (event: Event) => emit('update:modelValue', Number((event.target as HTMLInputElement).value)),
      });
  },
});

const ManagementPagedTableStub = defineComponent({
  name: 'ManagementPagedTable',
  props: {
    columns: { type: Array, default: () => [] },
    paginationVisible: { type: Boolean, default: true },
    rows: { type: Array, default: () => [] },
    selectedRowKeys: { type: Array, default: () => [] },
    summary: { type: String, default: '' },
  },
  emits: ['select-change'],
  setup(props, { emit, slots }) {
    return () =>
      h('div', { 'data-stub': 'ManagementPagedTable' }, [
        h('div', { 'data-columns': JSON.stringify(props.columns) }),
        h('div', { 'data-pagination-visible': String(props.paginationVisible) }),
        h('div', { 'data-summary': props.summary }),
        slots.toolbar?.(),
        slots.batch?.(),
        ...(props.rows as Array<Record<string, unknown>>).map((row) =>
          h('div', { key: String(row.service_name), 'data-row': String(row.service_name) }, [
            h(
              'button',
              {
                'data-select-row': String(row.service_name),
                onClick: () => emit('select-change', [row.service_name]),
              },
              'select',
            ),
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
    ownership_mode: 'external',
    runtime_status: runtimeStatus,
    service_count: 2,
    working_directory: '/srv/compose-demo',
    lifecycle_configuration: {
      strategy_kind: 'standard',
      profiles: [],
      down_before_redeploy: true,
      pull_before_redeploy: false,
      build_before_up: false,
      force_recreate: false,
      remove_orphans: true,
      wait_after_up: false,
      wait_timeout_seconds: 120,
      renew_anon_volumes: false,
      prune_images_after_redeploy: false,
      generated_commands: {
        up: {
          action: 'up',
          display_command: 'docker compose -f compose.yaml -p compose-demo up -d --remove-orphans',
          steps: [
            {
              argv: ['docker', 'compose', '-f', 'compose.yaml', '-p', 'compose-demo', 'up', '-d', '--remove-orphans'],
              display_command: 'docker compose -f compose.yaml -p compose-demo up -d --remove-orphans',
              kind: 'up',
            },
          ],
        },
        stop: {
          action: 'stop',
          display_command: 'docker compose -f compose.yaml -p compose-demo stop',
          steps: [
            {
              argv: ['docker', 'compose', '-f', 'compose.yaml', '-p', 'compose-demo', 'stop'],
              display_command: 'docker compose -f compose.yaml -p compose-demo stop',
              kind: 'stop',
            },
          ],
        },
        restart: {
          action: 'restart',
          display_command: 'docker compose -f compose.yaml -p compose-demo restart',
          steps: [
            {
              argv: ['docker', 'compose', '-f', 'compose.yaml', '-p', 'compose-demo', 'restart'],
              display_command: 'docker compose -f compose.yaml -p compose-demo restart',
              kind: 'restart',
            },
          ],
        },
        redeploy: {
          action: 'redeploy',
          display_command:
            'docker compose -f compose.yaml -p compose-demo down\ndocker compose -f compose.yaml -p compose-demo up -d --remove-orphans',
          steps: [
            {
              argv: ['docker', 'compose', '-f', 'compose.yaml', '-p', 'compose-demo', 'down'],
              display_command: 'docker compose -f compose.yaml -p compose-demo down',
              kind: 'down',
            },
            {
              argv: ['docker', 'compose', '-f', 'compose.yaml', '-p', 'compose-demo', 'up', '-d', '--remove-orphans'],
              display_command: 'docker compose -f compose.yaml -p compose-demo up -d --remove-orphans',
              kind: 'up',
            },
          ],
        },
      },
    },
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

function buildRuntimeContainers() {
  return {
    items: [
      {
        id: 'container-1',
        image: 'demo:latest',
        names: ['compose-demo-1'],
        orchestrator: {
          group_scope_kind: 'compose_project',
          group_value: 'compose-demo',
          member_scope_kind: 'compose_service',
          member_value: 'app',
        },
        ports: [{ ip: '0.0.0.0', private_port: 8080, public_port: 8316, type: 'tcp' }],
        runtime: 'docker',
        state: 'running',
        status: 'Up 1 minute',
      },
      {
        id: 'worker-1',
        image: 'worker:latest',
        names: ['worker-1'],
        orchestrator: {
          group_scope_kind: 'compose_project',
          group_value: 'compose-demo',
          member_scope_kind: 'compose_service',
          member_value: 'worker',
        },
        ports: [],
        runtime: 'docker',
        state: 'created',
        status: 'Created',
      },
    ],
  };
}

function buildProjectLogs(entries: Array<{ line: string; occurred_at: string; stream?: 'stdout' | 'stderr' }>) {
  return {
    canonical_project_name: 'compose-demo',
    entries: entries.map((entry, index) => ({
      container_id: `container-${index + 1}`,
      container_name: `compose-demo-${index + 1}`,
      line: entry.line,
      occurred_at: entry.occurred_at,
      service_name: index % 2 === 0 ? 'app' : 'worker',
      source: {
        container_id: `container-${index + 1}`,
        container_name: `compose-demo-${index + 1}`,
        service_name: index % 2 === 0 ? 'app' : 'worker',
      },
      stream: entry.stream ?? 'stdout',
    })),
    project_id: 7,
    stderr: true,
    stdout: true,
    tail: 200,
    timestamps: true,
    truncated: false,
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
        't-input-number': TInputNumberStub,
        't-loading': slotStub('TLoading'),
        't-progress': slotStub('TProgress'),
        't-space': slotStub('TSpace'),
        't-tab-panel': slotStub('TTabPanel'),
        't-tabs': slotStub('TTabs'),
        't-tag': slotStub('TTag'),
        't-switch': TSwitchStub,
      },
    },
  });
}

vi.mock('@/modules/container/contract/project', () => ({
  batchContainerActions: containerApiMocks.batchContainerActions,
  getContainers: containerApiMocks.getContainers,
  getContainerEvents: containerApiMocks.getContainerEvents,
  getContainerLogs: containerApiMocks.getContainerLogs,
}));

vi.mock('../../api/project', () => ({
  getProject: projectApiMocks.getProject,
  getProjectConfiguration: projectApiMocks.getProjectConfiguration,
  getProjectLogs: projectApiMocks.getProjectLogs,
  getProjectOverview: projectApiMocks.getProjectOverview,
  getProjectServices: projectApiMocks.getProjectServices,
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
    props: {
      entries: { type: Array, default: () => [] },
    },
    setup(props, { slots }) {
      return () =>
        h('div', { 'data-stub': 'LogViewer' }, [
          ...(props.entries as Array<{ line: string }>).map((entry, index) =>
            h('div', { key: `${index}-${entry.line}`, 'data-log-line': String(index) }, entry.line),
          ),
          slots.default?.(),
        ]);
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
  createRealtimeSnapshotGate: ({ apply }: { apply: (snapshot: unknown) => void }) => ({
    clear: vi.fn(),
    commit: (snapshot: unknown) => apply(snapshot),
    dispose: vi.fn(),
    flush: vi.fn(),
  }),
  openRealtimeTopicSocket: vi.fn((options: Record<string, any>) => {
    const controller = { close: vi.fn() };
    realtimeMocks.sockets.push({ controller, options });
    return controller;
  }),
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
    realtimeMocks.sockets.length = 0;
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
      ownership_mode: 'external',
    });
    projectApiMocks.getProjectLogs.mockResolvedValue({ entries: [] });
    projectApiMocks.getProjectOverview.mockResolvedValue(buildProjectOverview());
    projectApiMocks.getProjectServices.mockResolvedValue(buildProjectServices());
    containerApiMocks.getContainers.mockResolvedValue(buildRuntimeContainers());
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

  it('redirects the legacy configuration tab query to the configuration workspace route', async () => {
    routeState.value.query = { name: 'Compose Demo', tab: 'configuration' };

    mountPage();
    await flushPromises();

    expect(routerMocks.replace).toHaveBeenCalledWith({
      name: 'ProjectConfigurationWorkspaceIndex',
      params: { id: '7' },
      query: { name: 'Compose Demo' },
    });
    expect(projectApiMocks.getProject).not.toHaveBeenCalled();
  });

  it('does not retry project logs bootstrap in a tight loop after a failed logs request', async () => {
    routeState.value.query = { tab: 'logs' };
    projectApiMocks.getProjectLogs.mockRejectedValue(new Error('boom'));

    mountPage();
    await flushPromises();
    await flushPromises();

    expect(projectApiMocks.getProjectLogs).toHaveBeenCalledTimes(1);
    expect(messageMocks.error).not.toHaveBeenCalled();
  });

  it('renders project log snapshots in chronological order', async () => {
    routeState.value.query = { tab: 'logs' };
    projectApiMocks.getProjectLogs.mockResolvedValue(
      buildProjectLogs([
        { line: 'oldest-entry', occurred_at: '2026-07-09T03:00:00Z' },
        { line: 'middle-entry', occurred_at: '2026-07-09T03:00:01Z' },
        { line: 'latest-entry', occurred_at: '2026-07-09T03:00:02Z' },
      ]),
    );

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.findAll('[data-log-line]').map((node) => node.text())).toEqual([
      JSON.stringify({
        container: 'compose-demo-1',
        container_id: 'container-1',
        message: 'oldest-entry',
        occurred_at: '2026-07-09T03:00:00Z',
        service: 'app',
        source: 'app · compose-demo-1',
        stream: 'stdout',
      }),
      JSON.stringify({
        container: 'compose-demo-2',
        container_id: 'container-2',
        message: 'middle-entry',
        occurred_at: '2026-07-09T03:00:01Z',
        service: 'worker',
        source: 'worker · compose-demo-2',
        stream: 'stdout',
      }),
      JSON.stringify({
        container: 'compose-demo-3',
        container_id: 'container-3',
        message: 'latest-entry',
        occurred_at: '2026-07-09T03:00:02Z',
        service: 'app',
        source: 'app · compose-demo-3',
        stream: 'stdout',
      }),
    ]);
  });

  it('appends realtime project logs after the snapshot tail', async () => {
    routeState.value.query = { tab: 'logs' };
    projectApiMocks.getProjectLogs.mockResolvedValue(
      buildProjectLogs([{ line: 'snapshot-entry', occurred_at: '2026-07-09T03:00:00Z' }]),
    );

    const wrapper = mountPage();
    await flushPromises();

    const logsSocket = realtimeMocks.sockets.find((socket) => socket.options.topic === 'project.logs:7');
    expect(logsSocket).toBeTruthy();

    logsSocket?.options.onMessage({
      entry: {
        container_id: 'container-9',
        container_name: 'compose-demo-9',
        line: 'realtime-entry',
        occurred_at: '2026-07-09T03:00:01Z',
        service_name: 'worker',
        source: {
          container_id: 'container-9',
          container_name: 'compose-demo-9',
          service_name: 'worker',
        },
        stream: 'stderr',
      },
    });
    await flushPromises();

    expect(wrapper.findAll('[data-log-line]').map((node) => node.text())).toEqual([
      JSON.stringify({
        container: 'compose-demo-1',
        container_id: 'container-1',
        message: 'snapshot-entry',
        occurred_at: '2026-07-09T03:00:00Z',
        service: 'app',
        source: 'app · compose-demo-1',
        stream: 'stdout',
      }),
      JSON.stringify({
        container: 'compose-demo-9',
        container_id: 'container-9',
        message: 'realtime-entry',
        occurred_at: '2026-07-09T03:00:01Z',
        service: 'worker',
        source: 'worker · compose-demo-9',
        stream: 'stderr',
      }),
    ]);
  });

  it('renders service columns with shared pagination chrome', async () => {
    const wrapper = mountPage();
    await flushPromises();

    const table = wrapper.findComponent(ManagementPagedTableStub);
    const columns = table.props('columns') as Array<{ title: string }>;

    expect(columns.map((column) => column.title)).toEqual([
      'Selection',
      'Service',
      'Status',
      'Image',
      'Health',
      'Ports',
      'Operation',
    ]);
    expect(table.props('paginationVisible')).toBe(true);
    expect(wrapper.text()).not.toContain('Containers');
    expect(wrapper.text()).not.toContain('Networks');
  });

  it('does not render the retired configuration tab label in detail tabs', async () => {
    const wrapper = mountPage();
    await flushPromises();

    const tabLabels = wrapper
      .findAll('[data-stub="TTabPanel"]')
      .map((item) => item.text().trim())
      .filter(Boolean);

    expect(tabLabels).not.toContain('Configuration');
  });

  it('renders lifecycle sections with grouped configuration and command previews', async () => {
    routeState.value.query = { tab: 'lifecycle' };
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.find('[data-testid="project-lifecycle-summary-card"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="project-lifecycle-command-card"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="project-lifecycle-configuration-card"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="project-lifecycle-config-group-base"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="project-lifecycle-config-group-redeploy"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="project-lifecycle-actions"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="project-lifecycle-command-preview-up"]').text()).toContain(
      'docker compose -f compose.yaml -p compose-demo up -d --remove-orphans',
    );
    expect(wrapper.find('[data-testid="project-lifecycle-command-preview-redeploy"]').text()).toContain(
      'docker compose -f compose.yaml -p compose-demo down',
    );
  });

  it('shows wait-timeout and destructive-volume warning only when enabled', async () => {
    routeState.value.query = { tab: 'lifecycle' };
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.find('[data-testid="project-lifecycle-wait-timeout-field"]').exists()).toBe(false);
    expect(wrapper.find('[data-testid="project-lifecycle-renew-anon-volumes-warning"]').exists()).toBe(false);

    await wrapper.get('[data-testid="project-lifecycle-wait-after-up-switch"]').trigger('click');
    await wrapper.get('[data-testid="project-lifecycle-renew-anon-volumes-switch"]').trigger('click');
    await flushPromises();

    expect(wrapper.find('[data-testid="project-lifecycle-wait-timeout-field"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="project-lifecycle-renew-anon-volumes-warning"]').exists()).toBe(true);
  });

  it('renders runtime published ports instead of declared compose mappings', async () => {
    const wrapper = mountPage();
    await flushPromises();

    expect(containerApiMocks.getContainers).toHaveBeenCalledWith({
      limit: 100,
      offset: 0,
      orchestrator: 'compose',
      source_scope: 'compose-demo',
      source_scope_kind: 'compose_project',
    });
    expect(wrapper.find('[data-row="app"]').text()).toContain('8316:8080 TCP');
    expect(wrapper.find('[data-row="app"]').text()).not.toContain('127.0.0.1:8080:8080');
    expect(wrapper.find('[data-row="worker"]').text()).toContain('-');
  });

  it('paginates service rows locally while preserving the project-service summary', async () => {
    projectApiMocks.getProjectServices.mockResolvedValueOnce({
      items: Array.from({ length: 21 }, (_, index) => ({
        build_context: null,
        container_members: [],
        declared_networks: [],
        declared_ports: [],
        declared_volumes: [],
        image: `svc-${index + 1}:latest`,
        running_count: 0,
        service_name: `service-${index + 1}`,
        stopped_count: 1,
      })),
    });
    containerApiMocks.getContainers.mockResolvedValueOnce({ items: [], total: 0 });

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.findAll('[data-row]').map((item) => item.attributes('data-row'))).toHaveLength(20);
    expect(wrapper.find('[data-summary]').attributes('data-summary')).toBe('21 Services');
  });

  it('supports batch service restart from the selection bar', async () => {
    const dialogInstance = {
      destroy: vi.fn(),
      setConfirmLoading: vi.fn(),
    };
    dialogMocks.confirm.mockReturnValueOnce(dialogInstance);
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('[data-select-row="app"]').trigger('click');
    await flushPromises();
    await wrapper.get('[data-testid="project-service-batch-restart"]').trigger('click');
    await flushPromises();

    expect(dialogMocks.confirm).toHaveBeenCalledTimes(1);
    const [dialogOptions] = dialogMocks.confirm.mock.calls[0] as [
      {
        onConfirm?: () => Promise<void> | void;
      },
    ];
    await dialogOptions.onConfirm?.();
    await flushPromises();

    expect(containerApiMocks.batchContainerActions).toHaveBeenCalledWith({
      action: 'restart',
      force: false,
      ids: ['container-1', 'container-2'],
    });
    expect(dialogInstance.setConfirmLoading).toHaveBeenNthCalledWith(1, true);
    expect(dialogInstance.setConfirmLoading).toHaveBeenNthCalledWith(2, false);
    expect(dialogInstance.destroy).toHaveBeenCalledTimes(1);
  });

  it('renders the service batch selection bar with dedicated action alignment wrappers', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('[data-select-row="app"]').trigger('click');
    await flushPromises();

    expect(wrapper.find('.project-service-batch-bar').exists()).toBe(true);
    expect(wrapper.find('.project-service-batch-bar__actions').exists()).toBe(true);
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

  it('opens the configuration workspace from the detail page action area', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('[data-testid="project-detail-action-open-configuration-workspace"]').trigger('click');

    expect(routerMocks.resolve).toHaveBeenCalledWith({
      name: 'ProjectConfigurationWorkspaceIndex',
      params: { id: '7' },
      query: undefined,
    });
    expect(tabsRouterStoreMock.appendTabRouterList).toHaveBeenCalledWith(
      expect.objectContaining({
        fullPath: '/ops/projects/7/configuration',
        path: '/ops/projects/7/configuration',
        title: {
          'en-US': 'Configuration Workspace - Compose Demo',
          'zh-CN': '配置工作台 - Compose Demo',
        },
      }),
    );
    expect(routerMocks.push).toHaveBeenCalledWith({
      name: 'ProjectConfigurationWorkspaceIndex',
      params: { id: '7' },
      query: undefined,
    });
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
