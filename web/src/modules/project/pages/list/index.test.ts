import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, KeepAlive, nextTick, ref } from 'vue';

import { PROJECT_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';
import ProjectListPage from './index.vue';

const projectApiMocks = vi.hoisted(() => ({
  getProjects: vi.fn(),
  deleteProjectSavedView: vi.fn(),
  getProjectSavedViews: vi.fn(),
  postProjectSavedView: vi.fn(),
  postProjectBatchActions: vi.fn(),
  postProjectDestroy: vi.fn(),
  postProjectStop: vi.fn(),
  postProjectRedeploy: vi.fn(),
  postProjectRestart: vi.fn(),
  postProjectUpdateDeploy: vi.fn(),
  postProjectUnregister: vi.fn(),
  postProjectUp: vi.fn(),
  putProjectSavedView: vi.fn(),
}));

const runtimeTargetMocks = vi.hoisted(() => ({ listRuntimeTargets: vi.fn() }));

const projectRealtimeMocks = vi.hoisted(() => ({
  acquireProjectListRealtime: vi.fn(),
  releaseProjectListRealtime: vi.fn(),
}));

const taskRequestMocks = vi.hoisted(() => ({ get: vi.fn() }));

const routerMocks = vi.hoisted(() => ({
  push: vi.fn(),
  resolve: vi.fn(() => ({
    fullPath: '/ops/projects',
    meta: {},
    name: 'ProjectListIndex',
    params: {},
    path: '/ops/projects',
    query: {},
  })),
}));

const tabsRouterStoreMock = vi.hoisted(() => ({
  appendTabRouterList: vi.fn(),
  setActiveTabKey: vi.fn(),
}));

const dialogConfirmMock = vi.hoisted(() => vi.fn());
const dialogAlertMock = vi.hoisted(() => vi.fn());

type BatchActionResponseMock = {
  blocked_count: number;
  completed_count: number;
  items: Array<{ action: string; message: string; project_id: number; result: string; skipped: boolean }>;
  skipped_count: number;
  total_count: number;
};

const listMessages = {
  'project.lifecycle.reviewStatus.confirmed': 'Lifecycle Confirmed',
  'project.lifecycle.reviewStatus.reviewRequired': 'Review Required',
  'project.list.actions.create': 'Choose Project Source',
  'project.list.actions.detail': 'Detail',
  'project.list.actions.stop': 'Stop',
  'project.list.actions.destroy': 'Destroy',
  'project.list.actions.import': 'Import Existing Project',
  'project.list.actions.operationMenu': 'Actions',
  'project.list.actions.refresh': 'Refresh',
  'project.list.actions.noTaskHistory': 'No Task History',
  'project.list.actions.taskHistoryLoadFailed': 'Latest Task Load Failed',
  'project.list.actions.redeploy': 'Redeploy',
  'project.list.actions.restart': 'Restart',
  'project.list.actions.unregister': 'Unregister',
  'project.list.actions.up': 'Up',
  'project.list.batch.cancelSelection': 'Clear Selection',
  'project.list.batch.destroy': 'Batch Destroy',
  'project.list.batch.destroyConfirm': 'Destroy {count} selected projects?',
  'project.list.batch.destroySingleSelection': 'Batch destroy requires exactly one selected project.',
  'project.list.batch.noActionableSelection': 'No actionable selection',
  'project.list.batch.noSelection': 'Select projects first.',
  'project.list.batch.partial': 'Partial {count} {skippedCount} {blockedCount}',
  'project.list.batch.redeploy': 'Batch Redeploy',
  'project.list.batch.redeployConfirm': 'Redeploy {count} selected projects?',
  'project.list.batch.restart': 'Batch Restart',
  'project.list.batch.restartConfirm': 'Restart {count} selected projects?',
  'project.list.batch.scope': '{actionableCount} actionable {selectedCount} selected {skippedCount} skipped',
  'project.list.batch.selected': '{count} Selected',
  'project.list.batch.skipInapplicable': 'Skipped inapplicable rows.',
  'project.list.batch.start': 'Batch Start',
  'project.list.batch.startConfirm': 'Start {count} selected projects?',
  'project.list.batch.stop': 'Batch Stop',
  'project.list.batch.stopConfirm': 'Stop {count} selected projects?',
  'project.list.batch.success': 'Completed {count}, skipped {skippedCount}.',
  'project.list.batch.unregister': 'Batch Unregister',
  'project.list.batch.unregisterConfirm': 'Unregister {count} selected projects?',
  'project.list.clearFilters': 'Clear Filters',
  'project.list.runtimeTargetsLoadFailed': 'Failed to load runtime targets.',
  'project.list.columnSettings': 'Column Settings',
  'project.list.columns.selection': 'Select',
  'project.list.columns.drift': 'Sync Status',
  'project.list.columns.name': 'Project',
  'project.list.columns.operation': 'Operation',
  'project.list.columns.resources': 'Resources',
  'project.list.columns.runtime': 'Status',
  'project.list.columns.source': 'Source',
  'project.list.projectCount': 'Total {count}',
  'project.list.resources.container': 'Containers',
  'project.list.resources.issue': 'Issue',
  'project.list.resources.running': 'Running',
  'project.list.resources.service': 'Services',
  'project.list.resources.statusValue': '{status} {count}',
  'project.list.resources.stopped': 'Stopped',
  'project.list.resources.transitioning': 'Transitioning',
  'project.list.resources.unknown': 'Unknown',
  'project.list.tableSummary': 'Total {count}',
  'project.list.status.runtimeDegraded': 'Degraded',
  'project.list.status.runtimeRunning': 'Running',
  'project.list.status.runtimeStopped': 'Stopped',
  'project.list.status.runtimeTransitioning': 'Transitioning',
  'project.list.status.runtimeUnknown': 'Unknown',
  'project.list.statusTooltip.lifecycleReviewRequired':
    'Lifecycle configuration is not confirmed yet. Open project details to complete the review.',
  'project.list.statusTooltip.runtimeDegraded': 'Current Page Degraded',
  'project.list.statusTooltip.runtimeRunning': 'Current Page Running',
  'project.list.statusTooltip.runtimeStopped': 'Current Page Stopped',
  'project.list.statusTooltip.runtimeTransitioning': 'Current Page Transitioning',
  'project.list.statusTooltip.runtimeUnknown': 'Current Page Unknown',
  'project.list.statusTooltip.taskInProgress': 'Task In Progress',
  'project.list.statusTooltip.viewLatestTask': 'View Latest Task',
  'project.list.savedViews.deleteConfirmDescription': 'Delete filter "{name}"? This cannot be undone.',
  'project.list.savedViews.deleteConfirmTitle': 'Delete Filter',
  'project.list.savedViews.loadFailed': 'Failed to load saved filters.',
  'app.savedQueryViews.actions.cancel': 'Cancel',
  'app.savedQueryViews.actions.delete': 'Delete Filter',
  'app.savedQueryViews.actions.save': 'Save',
  'app.savedQueryViews.actions.saveAs': 'Save Filter',
  'app.savedQueryViews.actions.update': 'Update Filter',
  'app.savedQueryViews.dialog.createTitle': 'Save Filter',
  'app.savedQueryViews.dialog.deleteDescription': 'Delete filter "{name}"? This cannot be undone.',
  'app.savedQueryViews.dialog.deleteTitle': 'Delete Filter',
  'app.savedQueryViews.dialog.updateTitle': 'Update Filter',
  'app.savedQueryViews.label': 'Saved Filters',
  'app.savedQueryViews.namePlaceholder': 'Enter a filter name',
  'app.savedQueryViews.nameRequired': 'Enter a filter name.',
  'app.savedQueryViews.placeholder': 'Select a saved filter',
} as const;

function slotStub(name: string) {
  return defineComponent({
    name,
    setup(_props, { slots }) {
      return () =>
        h('div', { 'data-stub': name }, [
          slots.meta?.(),
          slots.actions?.(),
          slots.filters?.(),
          slots.head?.(),
          slots.toolbar?.(),
          slots.batch?.(),
          slots.footer?.(),
          slots.default?.(),
        ]);
    },
  });
}

const TaskDetailDrawerStub = defineComponent({
  name: 'TaskDetailDrawer',
  props: {
    taskId: { type: Number, default: null },
    visible: { type: Boolean, default: false },
  },
  setup(props) {
    return () => h('div', { 'data-stub': 'TaskDetailDrawer', 'data-task-id': String(props.taskId ?? '') });
  },
});

const TButtonStub = defineComponent({
  name: 'TButtonStub',
  props: {
    disabled: { type: Boolean, default: false },
    loading: { type: Boolean, default: false },
  },
  emits: ['click'],
  setup(props, { emit, slots }) {
    return () =>
      h(
        'button',
        {
          disabled: props.disabled,
          'data-loading': props.loading ? 'true' : 'false',
          onClick: (event: MouseEvent) => emit('click', event),
        },
        slots.default?.(),
      );
  },
});

const TTooltipStub = defineComponent({
  name: 'TTooltipStub',
  props: {
    content: { type: String, default: '' },
  },
  setup(props, { slots }) {
    return () =>
      h(
        'div',
        {
          'data-stub': 'TTooltip',
          'data-tooltip-content': props.content,
        },
        slots.default?.(),
      );
  },
});

const TTagStub = defineComponent({
  name: 'TTagStub',
  emits: ['click'],
  setup(_props, { attrs, emit, slots }) {
    return () =>
      h(
        'div',
        {
          ...attrs,
          'data-stub': 'TTag',
          onClick: (event: MouseEvent) => emit('click', event),
        },
        slots.default?.(),
      );
  },
});

const TDialogStub = defineComponent({
  name: 'TDialogStub',
  props: {
    visible: { type: Boolean, default: false },
  },
  emits: ['confirm'],
  setup(props, { slots }) {
    return () => (props.visible ? h('div', { 'data-stub': 'TDialog' }, slots.default?.()) : null);
  },
});

const TInputStub = defineComponent({
  name: 'TInputStub',
  props: {
    modelValue: { type: String, default: '' },
  },
  emits: ['update:modelValue'],
  setup(props, { emit, attrs }) {
    return () =>
      h('input', {
        ...attrs,
        value: props.modelValue,
        onInput: (event: Event) => emit('update:modelValue', (event.target as HTMLInputElement).value),
      });
  },
});

const TSelectStub = defineComponent({
  name: 'TSelectStub',
  props: {
    modelValue: { type: [Number, String], default: undefined },
  },
  emits: ['change', 'update:modelValue'],
  setup(_props, { slots }) {
    return () => h('div', { 'data-stub': 'TSelect' }, slots.default?.());
  },
});

const TCheckboxStub = defineComponent({
  name: 'TCheckboxStub',
  props: {
    checked: { type: Boolean, default: false },
    disabled: { type: Boolean, default: false },
  },
  emits: ['change'],
  setup(props, { emit, attrs }) {
    return () =>
      h('input', {
        ...attrs,
        checked: props.checked,
        disabled: props.disabled,
        type: 'checkbox',
        onChange: () => emit('change', !props.checked),
      });
  },
});

const TTableStub = defineComponent({
  name: 'TTableStub',
  props: {
    columns: { type: Array, required: true },
    data: { type: Array, required: true },
    rowKey: { type: String, default: 'id' },
    selectedRowKeys: { type: Array, default: () => [] },
  },
  emits: ['select-change'],
  setup(props, { slots }) {
    return () =>
      h('table', { class: 't-table-stub' }, [
        h(
          'thead',
          h(
            'tr',
            (props.columns as Array<Record<string, unknown>>).map((column) =>
              h('th', { 'data-col': String(column.colKey) }, String(column.title ?? column.colKey)),
            ),
          ),
        ),
        h(
          'tbody',
          (props.data as Array<Record<string, unknown>>).map((row) =>
            h(
              'tr',
              { 'data-row-id': String(row[props.rowKey] ?? '') },
              (props.columns as Array<Record<string, unknown>>).map((column) => {
                const slotName = String(column.colKey);
                const slot = slots[slotName];
                return h('td', { 'data-col': slotName }, slot ? slot({ row }) : String(row[slotName] ?? ''));
              }),
            ),
          ),
        ),
        slots.empty?.(),
      ]);
  },
});

vi.mock('../../api/project', () => ({
  deleteProjectSavedView: projectApiMocks.deleteProjectSavedView,
  getProjects: projectApiMocks.getProjects,
  getProjectSavedViews: projectApiMocks.getProjectSavedViews,
  postProjectBatchActions: projectApiMocks.postProjectBatchActions,
  postProjectSavedView: projectApiMocks.postProjectSavedView,
  postProjectDestroy: projectApiMocks.postProjectDestroy,
  postProjectStop: projectApiMocks.postProjectStop,
  postProjectRedeploy: projectApiMocks.postProjectRedeploy,
  postProjectRestart: projectApiMocks.postProjectRestart,
  postProjectUpdateDeploy: projectApiMocks.postProjectUpdateDeploy,
  postProjectUnregister: projectApiMocks.postProjectUnregister,
  postProjectUp: projectApiMocks.postProjectUp,
  putProjectSavedView: projectApiMocks.putProjectSavedView,
}));

vi.mock('@/modules/runtime-target/api/runtime-target', () => ({
  listRuntimeTargets: runtimeTargetMocks.listRuntimeTargets,
}));

vi.mock('../../shared/list-realtime', () => ({
  acquireProjectListRealtime: projectRealtimeMocks.acquireProjectListRealtime,
  releaseProjectListRealtime: projectRealtimeMocks.releaseProjectListRealtime,
}));

vi.mock('@/utils/request', () => ({
  request: {
    get: taskRequestMocks.get,
  },
}));

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>();
  const locale = ref('en-US');
  return {
    ...actual,
    useI18n: () => ({
      locale,
      t: (key: string, params?: Record<string, string | number>) => {
        const template = listMessages[key as keyof typeof listMessages] ?? key;
        if (!params) {
          return template;
        }

        let message = String(template);
        for (const [name, value] of Object.entries(params)) {
          message = message.replace(`{${name}}`, String(value));
        }
        return message;
      },
    }),
  };
});

vi.mock('vue-router', () => ({
  useRouter: () => routerMocks,
}));

vi.mock('@/store/modules/tabs-router', () => ({
  useTabsRouterStore: () => tabsRouterStoreMock,
}));

vi.mock('@/store', () => ({
  useRealtimeSchedulerStore: () => ({
    allowPolling: true,
  }),
}));

vi.mock('@/shared/localized-api-error', () => ({
  resolveLocalizedErrorMessage: (_t: unknown, _error: unknown, fallback: string) => fallback,
}));

vi.mock('@/utils/logger', () => ({
  createLogger: () => ({
    error: vi.fn(),
  }),
}));

vi.mock('@/utils/route/title', () => ({
  localizeRouteTitleKey: (key: string) => key,
}));

vi.mock('../../shared/navigation', () => ({
  appendResolvedTab: vi.fn(),
  buildDetailTitleWithFallback: (key: string) => key,
}));

vi.mock('tdesign-vue-next/es/dialog', () => ({
  DialogPlugin: {
    alert: (...args: unknown[]) => dialogAlertMock(...args),
    confirm: (...args: unknown[]) => dialogConfirmMock(...args),
  },
}));

vi.mock('tdesign-vue-next/es/message', () => ({
  MessagePlugin: {
    error: vi.fn(),
    success: vi.fn(),
  },
}));

vi.mock('@/shared/components/management', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/shared/components/management')>();
  const TableActionMenu = defineComponent({
    name: 'TableActionMenuStub',
    props: {
      actions: { type: Array, required: true },
    },
    emits: ['action'],
    setup(props, { emit }) {
      return () =>
        h(
          'div',
          { class: 'table-action-menu-stub' },
          (props.actions as Array<{ label: string; value: string }>).map((action) =>
            h(
              'button',
              {
                'data-testid': `row-action-${action.value}`,
                onClick: () => emit('action', action.value),
              },
              action.label,
            ),
          ),
        );
    },
  });

  return {
    ...actual,
    ManagementEmptyState: slotStub('ManagementEmptyState'),
    ManagementPageContent: slotStub('ManagementPageContent'),
    ManagementPageHeader: slotStub('ManagementPageHeader'),
    ManagementTableCard: slotStub('ManagementTableCard'),
    ManagementTablePagination: slotStub('ManagementTablePagination'),
    ManagementToolbar: slotStub('ManagementToolbar'),
    TableActionMenu,
    TableViewToolbar: slotStub('TableViewToolbar'),
    resolveTableWidthPolicy: () => ({ mode: 'fit', tableContentWidth: 'auto' }),
    useTableHostWidth: () => ({ tableHostRef: ref(null), tableHostWidth: ref(1280) }),
  };
});

function buildProjectRow(overrides: Record<string, unknown>) {
  return {
    activity_authority: 'frontend-fanout',
    application_id: '1',
    application_type: 'compose',
    compose_project_name: 'alpha',
    compose_project_name_source: 'explicit',
    container_counts: { issue: 0, running: 3, stopped: 0, total: 3, transitioning: 0 },
    display_name: 'Alpha',
    drift_status: 'clean',
    host_scope: 'local',
    lifecycle_review_status: 'confirmed',
    ownership_mode: 'external',
    runtime_status: 'running',
    service_count: 3,
    source_kind: 'imported',
    workspace_path: '/srv/alpha',
    ...overrides,
  };
}

function mountPage() {
  return mount(ProjectListPage, {
    global: {
      renderStubDefaultSlot: true,
      stubs: {
        'project-list-entry-actions': slotStub('ProjectListEntryActions'),
        'task-detail-drawer': TaskDetailDrawerStub,
        't-button': TButtonStub,
        't-checkbox': TCheckboxStub,
        't-checkbox-group': slotStub('TCheckboxGroup'),
        't-dialog': TDialogStub,
        't-drawer': slotStub('TDrawer'),
        't-empty': slotStub('TEmpty'),
        't-input': TInputStub,
        't-option': slotStub('TOption'),
        't-pagination': slotStub('TPagination'),
        't-select': TSelectStub,
        't-space': slotStub('TSpace'),
        't-table': TTableStub,
        't-tag': TTagStub,
        't-tooltip': TTooltipStub,
      },
    },
  });
}

function mountKeepAlivePage() {
  const visible = ref(true);
  const wrapper = mount(
    defineComponent({
      name: 'ProjectListKeepAliveHost',
      setup() {
        return () => h('div', [h(KeepAlive, null, () => (visible.value ? h(ProjectListPage) : null))]);
      },
    }),
    {
      global: {
        renderStubDefaultSlot: true,
        stubs: {
          'project-list-entry-actions': slotStub('ProjectListEntryActions'),
          'task-detail-drawer': TaskDetailDrawerStub,
          't-button': TButtonStub,
          't-checkbox': TCheckboxStub,
          't-checkbox-group': slotStub('TCheckboxGroup'),
          't-dialog': TDialogStub,
          't-drawer': slotStub('TDrawer'),
          't-empty': slotStub('TEmpty'),
          't-input': TInputStub,
          't-option': slotStub('TOption'),
          't-pagination': slotStub('TPagination'),
          't-select': TSelectStub,
          't-space': slotStub('TSpace'),
          't-table': TTableStub,
          't-tag': TTagStub,
          't-tooltip': TTooltipStub,
        },
      },
    },
  );

  return {
    wrapper,
    async deactivate() {
      visible.value = false;
      await nextTick();
    },
    async activate() {
      visible.value = true;
      await nextTick();
    },
  };
}

describe('Project list page', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.useRealTimers();
    projectRealtimeMocks.acquireProjectListRealtime.mockImplementation(() => undefined);
    projectRealtimeMocks.releaseProjectListRealtime.mockImplementation(() => undefined);
    runtimeTargetMocks.listRuntimeTargets.mockResolvedValue([]);
    projectApiMocks.getProjectSavedViews.mockResolvedValue([]);

    projectApiMocks.getProjects.mockResolvedValue({
      items: [
        buildProjectRow({}),
        buildProjectRow({
          application_id: '2',
          compose_project_name: 'beta',
          container_counts: { issue: 0, running: 0, stopped: 2, total: 2, transitioning: 0 },
          display_name: 'Beta',
          runtime_status: 'degraded',
          service_count: 2,
          source_kind: 'managed',
          workspace_path: '/srv/beta',
        }),
        buildProjectRow({
          application_id: '3',
          compose_project_name: 'gamma',
          container_counts: { issue: 0, running: 0, stopped: 1, total: 2, transitioning: 1 },
          display_name: 'Gamma',
          runtime_status: 'transitioning',
          source_kind: 'template',
          workspace_path: '/srv/gamma',
        }),
        buildProjectRow({
          application_id: '4',
          compose_project_name: 'delta',
          container_counts: { issue: 1, running: 0, stopped: 1, total: 1, transitioning: 0 },
          display_name: 'Delta',
          drift_status: 'unknown',
          runtime_status: 'unknown',
          service_count: 1,
          source_kind: 'template',
          workspace_path: '/srv/delta',
        }),
      ],
      limit: 20,
      offset: 0,
      total: 4,
    });

    dialogConfirmMock.mockImplementation((options: Record<string, (...args: never[]) => void>) => {
      const dialog = { destroy: vi.fn() };
      queueMicrotask(() => options.onConfirm?.());
      return dialog;
    });
  });

  it('renders only non-zero runtime summary badges in the header', async () => {
    const wrapper = mountPage();
    await flushPromises();

    expect(projectApiMocks.getProjects).toHaveBeenCalledTimes(1);
    expect(wrapper.get('[data-testid="project-status-summary-total"]').text()).toBe('Total 4');
    expect(wrapper.find('[data-testid="project-status-summary-running"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="project-status-summary-degraded"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="project-status-summary-transitioning"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="project-status-summary-unknown"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="project-status-summary-stopped"]').exists()).toBe(false);
  });

  it('uses response total for total-like summary copy instead of current page length', async () => {
    projectApiMocks.getProjects.mockResolvedValueOnce({
      items: [buildProjectRow({ application_id: '1' }), buildProjectRow({ application_id: '2' })],
      limit: 2,
      offset: 0,
      total: 42,
    });

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.get('[data-testid="project-status-summary-total"]').text()).toBe('Total 42');
    expect(wrapper.get('[data-testid="project-table-summary"]').text()).toBe('Total 42');
  });

  it('opens the shared saved-view dialog from the query builder', async () => {
    const wrapper = mountPage();
    await flushPromises();

    const saveButton = wrapper.findAll('button').find((button) => button.text() === 'Save Filter');
    expect(saveButton).toBeDefined();

    await saveButton?.trigger('click');
    expect(wrapper.find('[data-stub="TDialog"]').exists()).toBe(true);
  });

  it('creates saved views from the query builder', async () => {
    const initialView = {
      created_at: '2026-07-12T00:00:00Z',
      id: 8,
      name: 'Operations',
      page_size: 20,
      query_state: {},
      updated_at: '2026-07-12T00:00:00Z',
      visible_columns: ['name'],
    };
    const createdView = { ...initialView, id: 9, name: 'Production' };
    projectApiMocks.postProjectSavedView.mockResolvedValueOnce(createdView);
    projectApiMocks.getProjectSavedViews.mockResolvedValue([initialView, createdView]);

    const wrapper = mountPage();
    await flushPromises();

    const saveButton = wrapper.findAll('button').find((button) => button.text() === 'Save Filter');
    await saveButton?.trigger('click');
    await wrapper.findAllComponents(TInputStub).at(-1)?.setValue('Production');
    wrapper.getComponent(TDialogStub).vm.$emit('confirm');
    await flushPromises();

    expect(projectApiMocks.postProjectSavedView).toHaveBeenCalledWith(
      expect.objectContaining({
        name: 'Production',
        page_size: 20,
        query_state: { sort: ['created_at:desc'] },
        visible_columns: [
          'row-select',
          'name',
          'applicationType',
          'runtimeTarget',
          'provider',
          'source',
          'runtime',
          'resources',
          'drift',
          'operation',
        ],
      }),
    );
  });

  it('applies a saved view to filters, page size, and visible columns', async () => {
    const view = {
      created_at: '2026-07-12T00:00:00Z',
      id: 11,
      name: 'Docker runtime',
      page_size: 50,
      query_state: { keyword: 'api', provider: 'docker', runtime_status: 'running' },
      updated_at: '2026-07-12T00:00:00Z',
      visible_columns: ['name', 'runtime'],
    };
    projectApiMocks.getProjectSavedViews.mockResolvedValue([view]);
    const wrapper = mountPage();
    await flushPromises();

    const savedViewSelect = wrapper.findAllComponents(TSelectStub).at(-1);
    if (!savedViewSelect) {
      throw new Error('saved view select not found');
    }
    savedViewSelect.vm.$emit('update:modelValue', view.id);
    await flushPromises();

    expect(projectApiMocks.getProjects).toHaveBeenLastCalledWith(
      expect.objectContaining({ keyword: 'api', limit: 50, provider: 'docker', runtime_status: 'running' }),
    );
    expect(wrapper.findAll('th[data-col]').map((column) => column.attributes('data-col'))).toEqual(['name', 'runtime']);
  });

  it('requires confirmation before deleting a selected saved view', async () => {
    const view = {
      created_at: '2026-07-12T00:00:00Z',
      id: 12,
      name: 'Disposable',
      page_size: 20,
      query_state: {},
      updated_at: '2026-07-12T00:00:00Z',
      visible_columns: ['name'],
    };
    projectApiMocks.getProjectSavedViews.mockResolvedValue([view]);
    const wrapper = mountPage();
    await flushPromises();
    const savedViewSelect = wrapper.findAllComponents(TSelectStub).at(-1);
    if (!savedViewSelect) {
      throw new Error('saved view select not found');
    }
    savedViewSelect.vm.$emit('update:modelValue', view.id);
    await flushPromises();

    const deleteButton = wrapper.findAll('button').find((button) => button.text() === 'Delete Filter');
    await deleteButton?.trigger('click');
    await nextTick();

    expect(wrapper.find('[data-stub="TDialog"]').exists()).toBe(true);
    expect(projectApiMocks.deleteProjectSavedView).not.toHaveBeenCalled();

    const deleteDialog = wrapper.findAllComponents(TDialogStub).find((dialog) => dialog.isVisible());
    deleteDialog?.vm.$emit('confirm');
    await flushPromises();

    expect(projectApiMocks.deleteProjectSavedView).toHaveBeenCalledWith(view.id);
  });

  it('renders dynamic container resource badges with four-way semantics', async () => {
    const wrapper = mountPage();
    await flushPromises();

    const runningRow = wrapper.get('tr[data-row-id="1"]');
    expect(runningRow.get('[data-testid="project-resource-badge-running-1"]').text()).toContain('3');
    expect(runningRow.find('[data-testid="project-resource-badge-stopped-1"]').exists()).toBe(false);
    expect(runningRow.find('[data-testid="project-resource-badge-transitioning-1"]').exists()).toBe(false);
    expect(runningRow.find('[data-testid="project-resource-badge-issue-1"]').exists()).toBe(false);

    const stoppedRow = wrapper.get('tr[data-row-id="2"]');
    const stoppedBadge = stoppedRow.get('[data-testid="project-resource-badge-stopped-2"]');
    expect(stoppedBadge.text()).toContain('2');
    expect(stoppedBadge.attributes('title')).toBe('Stopped 2');
    expect(stoppedRow.find('[data-testid="project-resource-badge-transitioning-2"]').exists()).toBe(false);

    const transitioningRow = wrapper.get('tr[data-row-id="3"]');
    expect(transitioningRow.get('[data-testid="project-resource-badge-stopped-3"]').text()).toContain('1');
    expect(transitioningRow.get('[data-testid="project-resource-badge-transitioning-3"]').text()).toContain('1');
    expect(transitioningRow.find('[data-testid="project-resource-badge-issue-3"]').exists()).toBe(false);

    const issueRow = wrapper.get('tr[data-row-id="4"]');
    expect(issueRow.get('[data-testid="project-resource-badge-stopped-4"]').text()).toContain('1');
    expect(issueRow.get('[data-testid="project-resource-badge-issue-4"]').text()).toContain('1');
    expect(issueRow.find('[data-testid="project-resource-badge-running-4"]').exists()).toBe(false);
  });

  it('renders an unknown resource badge when runtime members are absent', async () => {
    projectApiMocks.getProjects.mockResolvedValueOnce({
      items: [
        buildProjectRow({
          container_counts: { issue: 0, running: 0, stopped: 0, total: 0, transitioning: 0 },
          application_id: '11',
          runtime_status: 'unknown',
        }),
      ],
      limit: 20,
      offset: 0,
      total: 1,
    });

    const wrapper = mountPage();
    await flushPromises();

    const row = wrapper.get('tr[data-row-id="11"]');
    const badge = row.get('[data-testid="project-resource-badge-unknown-11"]');
    expect(badge.text()).toContain('0');
    expect(badge.attributes('title')).toBe('Unknown 0');
    expect(row.find('[data-testid="project-resource-badge-stopped-11"]').exists()).toBe(false);
  });

  it('renders a fallback resource badge when all container counts are zero for known runtime rows', async () => {
    projectApiMocks.getProjects.mockResolvedValueOnce({
      items: [
        buildProjectRow({
          container_counts: { issue: 0, running: 0, stopped: 0, total: 0, transitioning: 0 },
          application_id: '12',
          runtime_status: 'running',
        }),
      ],
      limit: 20,
      offset: 0,
      total: 1,
    });

    const wrapper = mountPage();
    await flushPromises();

    const row = wrapper.get('tr[data-row-id="12"]');
    const badge = row.get('[data-testid="project-resource-badge-running-12"]');
    expect(badge.text()).toContain('0');
    expect(badge.attributes('title')).toBe('Running 0');
  });

  it('does not keep polling after the initial HTTP seed', async () => {
    vi.useFakeTimers();
    const wrapper = mountPage();
    await flushPromises();

    expect(projectApiMocks.getProjects).toHaveBeenCalledTimes(1);

    await vi.advanceTimersByTimeAsync(15_000);
    await flushPromises();

    expect(projectApiMocks.getProjects).toHaveBeenCalledTimes(1);
    wrapper.unmount();
  });

  it('fetches immediately when the keep-alive page is re-activated', async () => {
    const page = mountKeepAlivePage();
    await flushPromises();

    const initialCallCount = projectApiMocks.getProjects.mock.calls.length;

    await page.deactivate();
    await flushPromises();
    await page.activate();
    await flushPromises();

    expect(projectApiMocks.getProjects).toHaveBeenCalledTimes(initialCallCount + 1);
    page.wrapper.unmount();
  });

  it('subscribes to project list realtime and applies pushed row updates', async () => {
    const wrapper = mountPage();
    await flushPromises();

    expect(projectRealtimeMocks.acquireProjectListRealtime).toHaveBeenCalledTimes(1);
    const listener = projectRealtimeMocks.acquireProjectListRealtime.mock.calls[0]?.[0] as
      ((items: Array<Record<string, unknown>>) => void) | undefined;
    expect(typeof listener).toBe('function');

    listener?.([
      {
        container_counts: { issue: 0, running: 0, stopped: 3, total: 3, transitioning: 0 },
        drift_status: 'changed',
        application_id: '1',
        runtime_status: 'stopped',
        service_count: 4,
      },
    ]);
    await nextTick();

    const row = wrapper.get('tr[data-row-id="1"]');
    expect(row.get('[data-testid="project-runtime-status-1"]').text()).toBe('Stopped');
    expect(row.text()).toContain('Stopped');
    expect(row.find('[data-testid="project-resource-badge-stopped-1"]').exists()).toBe(true);
    wrapper.unmount();
  });

  it('opens the latest owner task directly from a settled runtime status', async () => {
    const applicationID = 'app_01ARZ3NDEKTSV4RRFFQ69G5FAV';
    taskRequestMocks.get.mockResolvedValue({
      items: [{ id: 19, owner_id: applicationID, owner_type: 'compose_project', status: 'success' }],
      limit: 1,
      offset: 0,
      total: 1,
    });
    projectApiMocks.getProjects.mockResolvedValueOnce({
      items: [buildProjectRow({ application_id: applicationID })],
      limit: 20,
      offset: 0,
      total: 1,
    });
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get(`[data-testid="project-runtime-status-${applicationID}"]`).trigger('click');
    await flushPromises();

    expect(taskRequestMocks.get).toHaveBeenCalledWith({
      params: { limit: 1, owner_id: applicationID, owner_type: 'compose_project' },
      url: '/api/tasks',
    });
    const taskDrawer = wrapper.getComponent({ name: 'TaskDetailDrawer' });
    expect(taskDrawer.props('taskId')).toBe(19);
    expect(taskDrawer.props('visible')).toBe(true);
    wrapper.unmount();
  });

  it('keeps the newest task selection when earlier owner lookups resolve late', async () => {
    let resolveFirst!: (value: { items: Array<{ id: number }>; limit: number; offset: number; total: number }) => void;
    let resolveSecond!: (value: { items: Array<{ id: number }>; limit: number; offset: number; total: number }) => void;
    taskRequestMocks.get
      .mockReturnValueOnce(
        new Promise((resolve) => {
          resolveFirst = resolve;
        }),
      )
      .mockReturnValueOnce(
        new Promise((resolve) => {
          resolveSecond = resolve;
        }),
      );
    projectApiMocks.getProjects.mockResolvedValueOnce({
      items: [buildProjectRow({ application_id: '1' }), buildProjectRow({ application_id: '2' })],
      limit: 20,
      offset: 0,
      total: 2,
    });

    const wrapper = mountPage();
    await flushPromises();
    await wrapper.get('[data-testid="project-runtime-status-1"]').trigger('click');
    await wrapper.get('[data-testid="project-runtime-status-2"]').trigger('click');

    resolveSecond({ items: [{ id: 22 }], limit: 1, offset: 0, total: 1 });
    await flushPromises();
    resolveFirst({ items: [{ id: 11 }], limit: 1, offset: 0, total: 1 });
    await flushPromises();

    expect(wrapper.getComponent({ name: 'TaskDetailDrawer' }).props('taskId')).toBe(22);
    expect(wrapper.get('[data-testid="project-runtime-status-1"]').attributes('disabled')).toBeUndefined();
    expect(wrapper.get('[data-testid="project-runtime-status-2"]').attributes('disabled')).toBeUndefined();
    wrapper.unmount();
  });

  it('shows a runtime loading spinner after restart until refreshed status data changes', async () => {
    vi.useFakeTimers();
    const restartGate = {} as { resolve?: () => void };
    projectApiMocks.postProjectRestart.mockImplementation(
      () =>
        new Promise<void>((resolve) => {
          restartGate.resolve = resolve;
        }),
    );
    projectApiMocks.getProjects
      .mockResolvedValueOnce({
        items: [buildProjectRow({ runtime_status: 'running' })],
        limit: 20,
        offset: 0,
        total: 1,
      })
      .mockResolvedValueOnce({
        items: [buildProjectRow({ runtime_status: 'transitioning' })],
        limit: 20,
        offset: 0,
        total: 1,
      });

    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('[data-testid="row-action-restart"]').trigger('click');
    await flushPromises();

    expect(wrapper.find('[data-testid="project-runtime-status-loading-1"]').exists()).toBe(true);
    expect(wrapper.get('[data-testid="project-runtime-status-1"]').text()).toBe('');

    restartGate.resolve?.();
    await flushPromises();

    expect(projectApiMocks.postProjectRestart).toHaveBeenCalledWith('1');
    expect(projectApiMocks.getProjects).toHaveBeenCalledTimes(2);
    expect(wrapper.find('[data-testid="project-runtime-status-loading-1"]').exists()).toBe(false);
    wrapper.unmount();
  });

  it('clears the runtime loading spinner after 15 seconds when refreshed status data never changes', async () => {
    vi.useFakeTimers();
    const restartGate = {} as { resolve?: () => void };
    projectApiMocks.postProjectRestart.mockImplementation(
      () =>
        new Promise<void>((resolve) => {
          restartGate.resolve = resolve;
        }),
    );
    projectApiMocks.getProjects
      .mockResolvedValueOnce({
        items: [buildProjectRow({ runtime_status: 'running' })],
        limit: 20,
        offset: 0,
        total: 1,
      })
      .mockResolvedValueOnce({
        items: [buildProjectRow({ runtime_status: 'running' })],
        limit: 20,
        offset: 0,
        total: 1,
      });

    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('[data-testid="row-action-restart"]').trigger('click');
    await flushPromises();

    expect(wrapper.find('[data-testid="project-runtime-status-loading-1"]').exists()).toBe(true);

    restartGate.resolve?.();
    await flushPromises();

    expect(wrapper.find('[data-testid="project-runtime-status-loading-1"]').exists()).toBe(true);

    await vi.advanceTimersByTimeAsync(15_000);
    await flushPromises();

    expect(wrapper.find('[data-testid="project-runtime-status-loading-1"]').exists()).toBe(false);
    expect(wrapper.get('[data-testid="project-runtime-status-1"]').text()).toBe('Running');
    wrapper.unmount();
  });

  it('hides lifecycle actions by runtime status without a refresh column', async () => {
    projectApiMocks.getProjects.mockResolvedValueOnce({
      items: [
        buildProjectRow({ application_id: '1', runtime_status: 'running' }),
        buildProjectRow({ application_id: '2', runtime_status: 'degraded' }),
        buildProjectRow({ application_id: '3', runtime_status: 'stopped' }),
      ],
      limit: 20,
      offset: 0,
      total: 3,
    });

    const wrapper = mountPage();
    await flushPromises();

    const columnHeaders = wrapper.findAll('th').map((cell) => cell.text());
    expect(columnHeaders).not.toContain('Last Project Refresh');

    const runningRow = wrapper.get('tr[data-row-id="1"]');
    expect(runningRow.find('[data-testid="row-action-refresh"]').exists()).toBe(false);
    expect(runningRow.find('[data-testid="row-action-up"]').exists()).toBe(false);
    expect(runningRow.find('[data-testid="row-action-stop"]').exists()).toBe(true);
    expect(runningRow.find('[data-testid="row-action-restart"]').exists()).toBe(true);

    const degradedRow = wrapper.get('tr[data-row-id="2"]');
    expect(degradedRow.find('[data-testid="row-action-up"]').exists()).toBe(false);
    expect(degradedRow.find('[data-testid="row-action-stop"]').exists()).toBe(true);
    expect(degradedRow.find('[data-testid="row-action-restart"]').exists()).toBe(true);

    const stoppedRow = wrapper.get('tr[data-row-id="3"]');
    expect(stoppedRow.find('[data-testid="row-action-up"]').exists()).toBe(true);
    expect(stoppedRow.find('[data-testid="row-action-stop"]').exists()).toBe(false);
    expect(stoppedRow.find('[data-testid="row-action-restart"]').exists()).toBe(true);

    wrapper.unmount();
  });

  it('adds the row actions for redeploy and destroy after lifecycle review is confirmed', async () => {
    const wrapper = mountPage();
    await flushPromises();

    const runningRow = wrapper.get('tr[data-row-id="1"]');
    expect(runningRow.find('[data-testid="row-action-redeploy"]').exists()).toBe(true);
    expect(runningRow.find('[data-testid="row-action-destroy"]').exists()).toBe(true);
  });

  it('shows a lifecycle review badge and hides compose actions for review-required imported projects', async () => {
    projectApiMocks.getProjects.mockResolvedValueOnce({
      items: [buildProjectRow({ application_id: '9', lifecycle_review_status: 'review_required' })],
      limit: 20,
      offset: 0,
      total: 1,
    });

    const wrapper = mountPage();
    await flushPromises();

    const row = wrapper.get('tr[data-row-id="9"]');
    expect(row.text()).toContain('Review Required');
    expect(row.get('[data-stub="TTooltip"]').attributes('data-tooltip-content')).toBe(
      'Lifecycle configuration is not confirmed yet. Open project details to complete the review.',
    );
    expect(row.find('[data-testid="row-action-up"]').exists()).toBe(false);
    expect(row.find('[data-testid="row-action-stop"]').exists()).toBe(false);
    expect(row.find('[data-testid="row-action-restart"]').exists()).toBe(false);
    expect(row.find('[data-testid="row-action-redeploy"]').exists()).toBe(false);
    expect(row.find('[data-testid="row-action-unregister"]').exists()).toBe(true);
  });

  it('opens the lifecycle tab when the lifecycle review tag is clicked', async () => {
    projectApiMocks.getProjects.mockResolvedValueOnce({
      items: [
        buildProjectRow({ application_id: '9', display_name: 'Dockge', lifecycle_review_status: 'review_required' }),
      ],
      limit: 20,
      offset: 0,
      total: 1,
    });

    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('tr[data-row-id="9"] [data-testid="project-lifecycle-review-tag"]').trigger('click');

    expect(routerMocks.push).toHaveBeenCalledWith({
      name: PROJECT_BOOTSTRAP_ROUTE.DETAIL.pageRouteName,
      params: { id: '9' },
      query: {
        name: 'Dockge',
        tab: 'lifecycle',
      },
    });
  });

  it('does not render the lifecycle review tag for confirmed projects', async () => {
    const wrapper = mountPage();
    await flushPromises();

    const row = wrapper.get('tr[data-row-id="1"]');
    expect(row.find('[data-testid="project-lifecycle-review-tag"]').exists()).toBe(false);
  });

  it('submits only actionable selected rows for batch stop and reports skipped rows', async () => {
    projectApiMocks.postProjectBatchActions.mockResolvedValueOnce({
      blocked_count: 0,
      completed_count: 1,
      items: [{ action: 'stop', message: '', project_id: 2, result: 'completed', skipped: false }],
      skipped_count: 1,
      total_count: 2,
    });

    projectApiMocks.getProjects.mockResolvedValueOnce({
      items: [
        buildProjectRow({ application_id: '1', runtime_status: 'stopped' }),
        buildProjectRow({ application_id: '2', runtime_status: 'running' }),
      ],
      limit: 20,
      offset: 0,
      total: 2,
    });

    const wrapper = mountPage();
    await flushPromises();

    wrapper.getComponent(TTableStub).vm.$emit('select-change', ['1', '2']);
    await flushPromises();
    await wrapper.get('[data-testid="project-batch-stop"]').trigger('click');
    await flushPromises();

    expect(projectApiMocks.postProjectBatchActions).toHaveBeenCalledWith({
      action: 'stop',
      auto_unregister: false,
      confirm_canonical_project_name: undefined,
      delete_working_directory: false,
      image_prune: false,
      application_ids: ['2'],
      remove_named_volumes: false,
    });
  });

  it('shows row-level loading for actionable batch rows while the batch request is running', async () => {
    let resolveBatchAction!: (value: BatchActionResponseMock) => void;
    projectApiMocks.postProjectBatchActions.mockReturnValueOnce(
      new Promise<BatchActionResponseMock>((resolve) => {
        resolveBatchAction = resolve;
      }),
    );
    projectApiMocks.getProjects.mockResolvedValueOnce({
      items: [
        buildProjectRow({ application_id: '1', runtime_status: 'stopped' }),
        buildProjectRow({ application_id: '2', runtime_status: 'running' }),
      ],
      limit: 20,
      offset: 0,
      total: 2,
    });

    const wrapper = mountPage();
    await flushPromises();

    wrapper.getComponent(TTableStub).vm.$emit('select-change', ['1', '2']);
    await flushPromises();
    await wrapper.get('[data-testid="project-batch-stop"]').trigger('click');
    await flushPromises();

    expect(wrapper.get('[data-testid="project-batch-stop"]').attributes('data-loading')).toBe('true');
    expect(wrapper.find('[data-testid="project-runtime-status-loading-1"]').exists()).toBe(false);
    expect(wrapper.find('[data-testid="project-runtime-status-loading-2"]').exists()).toBe(true);

    resolveBatchAction({
      blocked_count: 0,
      completed_count: 1,
      items: [{ action: 'stop', message: '', project_id: 2, result: 'completed', skipped: false }],
      skipped_count: 1,
      total_count: 2,
    });
    await flushPromises();

    wrapper.unmount();
  });

  it('disables batch destroy when multiple rows are selected', async () => {
    const wrapper = mountPage();
    await flushPromises();

    wrapper.getComponent(TTableStub).vm.$emit('select-change', ['1', '2']);
    await flushPromises();

    expect(wrapper.get('[data-testid="project-batch-destroy"]').attributes('disabled')).toBeDefined();

    await wrapper.get('[data-testid="project-batch-destroy"]').trigger('click');
    await flushPromises();

    expect(dialogConfirmMock).not.toHaveBeenCalled();
    expect(projectApiMocks.postProjectBatchActions).not.toHaveBeenCalled();
  });

  it('closes the confirm dialog before the batch request settles', async () => {
    let resolveBatchAction!: (value: BatchActionResponseMock) => void;
    projectApiMocks.postProjectBatchActions.mockReturnValueOnce(
      new Promise<BatchActionResponseMock>((resolve) => {
        resolveBatchAction = resolve;
      }),
    );

    const destroySpy = vi.fn();
    dialogConfirmMock.mockImplementationOnce((options: Record<string, (...args: never[]) => void>) => {
      const dialog = { destroy: destroySpy };
      queueMicrotask(() => options.onConfirm?.());
      return dialog;
    });

    const wrapper = mountPage();
    await flushPromises();

    wrapper.getComponent(TTableStub).vm.$emit('select-change', ['1']);
    await flushPromises();
    await wrapper.get('[data-testid="project-batch-stop"]').trigger('click');
    await flushPromises();

    expect(destroySpy).toHaveBeenCalledTimes(1);
    expect(dialogAlertMock).not.toHaveBeenCalled();

    resolveBatchAction({
      blocked_count: 0,
      completed_count: 1,
      items: [{ action: 'stop', message: '', project_id: 1, result: 'completed', skipped: false }],
      skipped_count: 0,
      total_count: 1,
    });
    await flushPromises();
  });

  it('renders only blocked batch items in the alert summary with preserved line breaks', async () => {
    projectApiMocks.postProjectBatchActions.mockResolvedValueOnce({
      blocked_count: 1,
      completed_count: 1,
      items: [
        {
          action: 'stop',
          message: '',
          message_key: 'project.list.batch.skipInapplicable',
          project_id: 1,
          result: 'blocked',
          skipped: true,
        },
        { action: 'stop', message: '', project_id: 2, result: 'completed', skipped: false },
        { action: 'stop', message: 'docker compose failed', project_id: 3, result: 'blocked', skipped: false },
      ],
      skipped_count: 1,
      total_count: 3,
    });

    const wrapper = mountPage();
    await flushPromises();

    wrapper.getComponent(TTableStub).vm.$emit('select-change', ['1', '2', '3']);
    await flushPromises();
    await wrapper.get('[data-testid="project-batch-stop"]').trigger('click');
    await flushPromises();

    expect(dialogAlertMock).toHaveBeenCalledTimes(1);
    const [options] = dialogAlertMock.mock.calls[0] as [{ body: () => ReturnType<typeof h> }];
    expect(typeof options.body).toBe('function');

    const bodyVNode = options.body();
    expect(bodyVNode.props?.style).toEqual({ whiteSpace: 'pre-line' });
    expect(bodyVNode.children).toBe('3: docker compose failed');

    wrapper.unmount();
  });
});
