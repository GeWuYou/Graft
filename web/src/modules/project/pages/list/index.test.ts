import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, KeepAlive, nextTick, ref } from 'vue';

import ProjectListPage from './index.vue';

const projectApiMocks = vi.hoisted(() => ({
  getProjects: vi.fn(),
  postProjectBatchActions: vi.fn(),
  postProjectDestroy: vi.fn(),
  postProjectDown: vi.fn(),
  postProjectRedeploy: vi.fn(),
  postProjectRefresh: vi.fn(),
  postProjectRestart: vi.fn(),
  postProjectUpdateDeploy: vi.fn(),
  postProjectUnregister: vi.fn(),
  postProjectUp: vi.fn(),
}));

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
  'project.list.actions.create': 'Choose Project Source',
  'project.list.actions.detail': 'Detail',
  'project.list.actions.down': 'Down',
  'project.list.actions.destroy': 'Destroy',
  'project.list.actions.import': 'Import Existing Project',
  'project.list.actions.operationMenu': 'Actions',
  'project.list.actions.refresh': 'Refresh',
  'project.list.actions.redeploy': 'Redeploy',
  'project.list.actions.restart': 'Restart',
  'project.list.actions.unregister': 'Unregister',
  'project.list.actions.updateDeploy': 'Update Deploy',
  'project.list.actions.up': 'Up',
  'project.list.batch.cancelSelection': 'Clear Selection',
  'project.list.batch.destroy': 'Batch Destroy',
  'project.list.batch.destroyConfirm': 'Destroy {count} selected projects?',
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
  'project.list.batch.updateDeploy': 'Batch Update Deploy',
  'project.list.batch.updateDeployConfirm': 'Update deploy {count} selected projects?',
  'project.list.clearFilters': 'Clear Filters',
  'project.list.columnSettings': 'Column Settings',
  'project.list.columns.selection': 'Select',
  'project.list.columns.drift': 'Sync Status',
  'project.list.columns.name': 'Project',
  'project.list.columns.operation': 'Operation',
  'project.list.columns.refresh': 'Last Project Refresh',
  'project.list.columns.resources': 'Resources',
  'project.list.columns.runtime': 'Status',
  'project.list.columns.source': 'Source',
  'project.list.refresh': 'Refresh',
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
  'project.list.statusTooltip.runtimeDegraded': 'Current Page Degraded',
  'project.list.statusTooltip.runtimeRunning': 'Current Page Running',
  'project.list.statusTooltip.runtimeStopped': 'Current Page Stopped',
  'project.list.statusTooltip.runtimeTransitioning': 'Current Page Transitioning',
  'project.list.statusTooltip.runtimeUnknown': 'Current Page Unknown',
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
              { 'data-row-id': String(row.id ?? '') },
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
  getProjects: projectApiMocks.getProjects,
  postProjectBatchActions: projectApiMocks.postProjectBatchActions,
  postProjectDestroy: projectApiMocks.postProjectDestroy,
  postProjectDown: projectApiMocks.postProjectDown,
  postProjectRedeploy: projectApiMocks.postProjectRedeploy,
  postProjectRefresh: projectApiMocks.postProjectRefresh,
  postProjectRestart: projectApiMocks.postProjectRestart,
  postProjectUpdateDeploy: projectApiMocks.postProjectUpdateDeploy,
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
    canonical_project_name: 'alpha',
    container_counts: { issue: 0, running: 3, stopped: 0, total: 3, transitioning: 0 },
    display_name: 'Alpha',
    drift_status: 'clean',
    id: 1,
    last_refresh_at: '2026-07-03T10:00:00Z',
    last_refresh_status: 'success',
    ownership_mode: 'external',
    runtime_status: 'running',
    service_count: 3,
    source_kind: 'imported',
    working_directory: '/srv/alpha',
    ...overrides,
  };
}

function mountPage() {
  return mount(ProjectListPage, {
    global: {
      renderStubDefaultSlot: true,
      stubs: {
        'project-list-entry-actions': slotStub('ProjectListEntryActions'),
        'refresh-icon': true,
        't-button': TButtonStub,
        't-checkbox': TCheckboxStub,
        't-checkbox-group': slotStub('TCheckboxGroup'),
        't-drawer': slotStub('TDrawer'),
        't-empty': slotStub('TEmpty'),
        't-input': slotStub('TInput'),
        't-option': slotStub('TOption'),
        't-pagination': slotStub('TPagination'),
        't-select': slotStub('TSelect'),
        't-space': slotStub('TSpace'),
        't-table': TTableStub,
        't-tag': slotStub('TTag'),
        't-tooltip': slotStub('TTooltip'),
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
          'refresh-icon': true,
          't-button': TButtonStub,
          't-checkbox': TCheckboxStub,
          't-checkbox-group': slotStub('TCheckboxGroup'),
          't-drawer': slotStub('TDrawer'),
          't-empty': slotStub('TEmpty'),
          't-input': slotStub('TInput'),
          't-option': slotStub('TOption'),
          't-pagination': slotStub('TPagination'),
          't-select': slotStub('TSelect'),
          't-space': slotStub('TSpace'),
          't-table': TTableStub,
          't-tag': slotStub('TTag'),
          't-tooltip': slotStub('TTooltip'),
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

    projectApiMocks.getProjects.mockResolvedValue({
      items: [
        buildProjectRow({}),
        buildProjectRow({
          canonical_project_name: 'beta',
          container_counts: { issue: 0, running: 0, stopped: 2, total: 2, transitioning: 0 },
          display_name: 'Beta',
          id: 2,
          last_refresh_at: '2026-07-03T10:05:00Z',
          runtime_status: 'degraded',
          service_count: 2,
          source_kind: 'managed',
          working_directory: '/srv/beta',
        }),
        buildProjectRow({
          canonical_project_name: 'gamma',
          container_counts: { issue: 0, running: 0, stopped: 1, total: 2, transitioning: 1 },
          display_name: 'Gamma',
          id: 3,
          last_refresh_at: '2026-07-03T10:10:00Z',
          last_refresh_status: 'failed',
          runtime_status: 'transitioning',
          source_kind: 'git',
          working_directory: '/srv/gamma',
        }),
        buildProjectRow({
          canonical_project_name: 'delta',
          container_counts: { issue: 1, running: 0, stopped: 1, total: 1, transitioning: 0 },
          display_name: 'Delta',
          drift_status: 'unknown',
          id: 4,
          last_refresh_at: null,
          last_refresh_status: 'never',
          runtime_status: 'unknown',
          service_count: 1,
          source_kind: 'template',
          working_directory: '/srv/delta',
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
      items: [buildProjectRow({ id: 1 }), buildProjectRow({ id: 2 })],
      limit: 2,
      offset: 0,
      total: 42,
    });

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.get('[data-testid="project-status-summary-total"]').text()).toBe('Total 42');
    expect(wrapper.get('[data-testid="project-table-summary"]').text()).toBe('Total 42');
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
          id: 11,
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
          id: 12,
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

  it('polls the current page so refreshed timestamps can advance without manual input', async () => {
    vi.useFakeTimers();
    const wrapper = mountPage();
    await flushPromises();

    expect(projectApiMocks.getProjects).toHaveBeenCalledTimes(1);

    await vi.advanceTimersByTimeAsync(5000);
    await flushPromises();

    expect(projectApiMocks.getProjects).toHaveBeenCalledTimes(2);
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
        items: [buildProjectRow({ last_refresh_at: '2026-07-03T10:05:00Z', runtime_status: 'transitioning' })],
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

    expect(projectApiMocks.postProjectRestart).toHaveBeenCalledWith(1);
    expect(projectApiMocks.getProjects).toHaveBeenCalledTimes(2);
    expect(wrapper.find('[data-testid="project-runtime-status-loading-1"]').exists()).toBe(false);
    wrapper.unmount();
  });

  it('renames the refresh column and hides lifecycle actions by runtime status', async () => {
    projectApiMocks.getProjects.mockResolvedValueOnce({
      items: [
        buildProjectRow({ id: 1, runtime_status: 'running' }),
        buildProjectRow({ id: 2, runtime_status: 'degraded' }),
        buildProjectRow({ id: 3, runtime_status: 'stopped' }),
      ],
      limit: 20,
      offset: 0,
      total: 3,
    });

    const wrapper = mountPage();
    await flushPromises();

    const columnHeaders = wrapper.findAll('th').map((cell) => cell.text());
    expect(columnHeaders).toContain('Last Project Refresh');

    const runningRow = wrapper.get('tr[data-row-id="1"]');
    expect(runningRow.find('[data-testid="row-action-up"]').exists()).toBe(false);
    expect(runningRow.find('[data-testid="row-action-down"]').exists()).toBe(true);
    expect(runningRow.find('[data-testid="row-action-restart"]').exists()).toBe(true);

    const degradedRow = wrapper.get('tr[data-row-id="2"]');
    expect(degradedRow.find('[data-testid="row-action-up"]').exists()).toBe(false);
    expect(degradedRow.find('[data-testid="row-action-down"]').exists()).toBe(true);
    expect(degradedRow.find('[data-testid="row-action-restart"]').exists()).toBe(true);

    const stoppedRow = wrapper.get('tr[data-row-id="3"]');
    expect(stoppedRow.find('[data-testid="row-action-up"]').exists()).toBe(true);
    expect(stoppedRow.find('[data-testid="row-action-down"]').exists()).toBe(false);
    expect(stoppedRow.find('[data-testid="row-action-restart"]').exists()).toBe(true);

    wrapper.unmount();
  });

  it('adds the new row actions for redeploy, update deploy, and destroy', async () => {
    const wrapper = mountPage();
    await flushPromises();

    const runningRow = wrapper.get('tr[data-row-id="1"]');
    expect(runningRow.find('[data-testid="row-action-redeploy"]').exists()).toBe(true);
    expect(runningRow.find('[data-testid="row-action-update_deploy"]').exists()).toBe(true);
    expect(runningRow.find('[data-testid="row-action-destroy"]').exists()).toBe(true);
  });

  it('submits only actionable selected rows for batch stop and reports skipped rows', async () => {
    projectApiMocks.postProjectBatchActions.mockResolvedValueOnce({
      blocked_count: 0,
      completed_count: 1,
      items: [{ action: 'down', message: '', project_id: 2, result: 'completed', skipped: false }],
      skipped_count: 1,
      total_count: 2,
    });

    projectApiMocks.getProjects.mockResolvedValueOnce({
      items: [
        buildProjectRow({ id: 1, runtime_status: 'stopped' }),
        buildProjectRow({ id: 2, runtime_status: 'running' }),
      ],
      limit: 20,
      offset: 0,
      total: 2,
    });

    const wrapper = mountPage();
    await flushPromises();

    wrapper.getComponent(TTableStub).vm.$emit('select-change', [1, 2]);
    await flushPromises();
    await wrapper.get('[data-testid="project-batch-stop"]').trigger('click');
    await flushPromises();

    expect(projectApiMocks.postProjectBatchActions).toHaveBeenCalledWith({
      action: 'stop',
      auto_unregister: false,
      confirm_canonical_project_name: undefined,
      delete_working_directory: false,
      image_prune: false,
      project_ids: [2],
      remove_named_volumes: false,
    });
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

    wrapper.getComponent(TTableStub).vm.$emit('select-change', [1]);
    await flushPromises();
    await wrapper.get('[data-testid="project-batch-stop"]').trigger('click');
    await flushPromises();

    expect(destroySpy).toHaveBeenCalledTimes(1);
    expect(dialogAlertMock).not.toHaveBeenCalled();

    resolveBatchAction({
      blocked_count: 0,
      completed_count: 1,
      items: [{ action: 'down', message: '', project_id: 1, result: 'completed', skipped: false }],
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
          action: 'down',
          message: '',
          message_key: 'project.list.batch.skipInapplicable',
          project_id: 1,
          result: 'blocked',
          skipped: true,
        },
        { action: 'down', message: '', project_id: 2, result: 'completed', skipped: false },
        { action: 'down', message: 'docker compose failed', project_id: 3, result: 'blocked', skipped: false },
      ],
      skipped_count: 1,
      total_count: 3,
    });

    const wrapper = mountPage();
    await flushPromises();

    wrapper.getComponent(TTableStub).vm.$emit('select-change', [1, 2, 3]);
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
