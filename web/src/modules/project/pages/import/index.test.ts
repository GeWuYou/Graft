import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, nextTick, reactive, ref } from 'vue';

import zhProjectLocale from '../../locales/zh-CN.json';
import type { ProjectImportRuntimeCandidate } from '../../types/import';
import ProjectImportIndex from './index.vue';

const mocks = vi.hoisted(() => ({
  getProjectImportRuntimeCandidates: vi.fn(),
  replace: vi.fn(),
  push: vi.fn(),
  resolve: vi.fn(() => ({ href: '/ops/projects/1' })),
  useProjectImportFlow: vi.fn(),
}));

const routeState = reactive({
  name: 'ProjectImportIndex',
  params: {},
  query: {},
});

vi.mock('../../api/import', () => ({
  getProjectImportRuntimeCandidates: mocks.getProjectImportRuntimeCandidates,
}));

vi.mock('../../shared/useProjectImportFlow', () => ({
  useProjectImportFlow: mocks.useProjectImportFlow,
}));

vi.mock('../../shared/page-context', () => ({
  useProjectPageContext: () => ({
    router: {
      push: mocks.push,
      replace: mocks.replace,
      resolve: mocks.resolve,
    },
    tabsRouterStore: {},
    t: translate,
  }),
}));

vi.mock('vue-router', async () => {
  const actual = await vi.importActual<typeof import('vue-router')>('vue-router');
  return {
    ...actual,
    useRoute: () => routeState,
  };
});

vi.mock('tdesign-vue-next/es/message', () => ({
  MessagePlugin: {
    error: vi.fn(),
    success: vi.fn(),
  },
}));

vi.mock('@/shared/components/management', async () => {
  const actual = await vi.importActual<typeof import('@/shared/components/management')>(
    '@/shared/components/management',
  );
  const { defineComponent, h } = await import('vue');

  const ManagementPageContent = defineComponent({
    name: 'ManagementPageContentStub',
    setup(_props, { slots }) {
      return () => h('section', { class: 'management-page-content-stub' }, slots.default?.());
    },
  });

  const ManagementPageHeader = defineComponent({
    name: 'ManagementPageHeaderStub',
    setup(_props, { slots }) {
      return () =>
        h('header', [
          h('div', { class: 'management-page-header-meta' }, slots.meta?.()),
          h('div', { class: 'management-page-header-actions' }, slots.actions?.()),
        ]);
    },
  });

  const ManagementToolbar = defineComponent({
    name: 'ManagementToolbarStub',
    setup(_props, { slots }) {
      return () =>
        h('div', { class: 'management-toolbar-stub' }, [
          h('div', { class: 'management-toolbar-filters' }, slots.filters?.()),
          h('div', { class: 'management-toolbar-actions' }, slots.actions?.()),
        ]);
    },
  });

  const ManagementEmptyState = defineComponent({
    name: 'ManagementEmptyStateStub',
    props: {
      description: { type: String, default: '' },
      title: { type: String, default: '' },
    },
    setup(props, { slots }) {
      return () =>
        h('div', { class: 'management-empty-state-stub' }, [
          h('p', props.title),
          h('p', props.description),
          slots.actions?.(),
        ]);
    },
  });

  const TableViewToolbar = defineComponent({
    name: 'TableViewToolbarStub',
    props: {
      columnSettingsLabel: { type: String, default: '' },
      refreshLabel: { type: String, default: '' },
    },
    emits: ['column-settings', 'refresh'],
    setup(props, { emit, slots }) {
      return () =>
        h('div', { class: 'table-view-toolbar-stub' }, [
          slots.before?.(),
          h(
            'button',
            {
              'data-testid': 'table-refresh',
              onClick: () => emit('refresh'),
            },
            props.refreshLabel,
          ),
          h(
            'button',
            {
              'data-testid': 'table-column-settings',
              onClick: () => emit('column-settings'),
            },
            props.columnSettingsLabel,
          ),
          slots.default?.(),
        ]);
    },
  });

  return {
    ...actual,
    ManagementEmptyState,
    ManagementPageContent,
    ManagementPageHeader,
    ManagementToolbar,
    TableViewToolbar,
  };
});

vi.mock('@/shared/components/query-list', async () => {
  const actual = await vi.importActual<typeof import('@/shared/components/query-list')>(
    '@/shared/components/query-list',
  );
  const { defineComponent, h } = await import('vue');

  const AdvancedQueryPagedTable = defineComponent({
    name: 'AdvancedQueryPagedTableStub',
    props: {
      columns: { type: Array, required: true },
      description: { type: String, default: '' },
      emptyDescription: { type: String, default: '' },
      emptyTitle: { type: String, default: '' },
      rows: { type: Array, required: true },
      summary: { type: String, default: '' },
    },
    setup(props, { slots }) {
      return () =>
        h('section', { class: 'advanced-query-paged-table-stub' }, [
          h('div', { class: 'advanced-query-paged-table-toolbar' }, slots.toolbar?.()),
          h('p', { class: 'advanced-query-paged-table-summary' }, props.summary),
          h('p', { class: 'advanced-query-paged-table-description' }, props.description),
          props.rows.length
            ? h('table', [
                h(
                  'thead',
                  h(
                    'tr',
                    (props.columns as Array<Record<string, unknown>>).map((column) =>
                      h(
                        'th',
                        {
                          'data-col': String(column.colKey),
                        },
                        String(column.title ?? column.colKey),
                      ),
                    ),
                  ),
                ),
                h(
                  'tbody',
                  (props.rows as Array<Record<string, unknown>>).map((row) =>
                    h(
                      'tr',
                      (props.columns as Array<Record<string, unknown>>).map((column) => {
                        const slotName = String(column.colKey);
                        const slot = slots[slotName];
                        return h(
                          'td',
                          {
                            'data-col': slotName,
                          },
                          slot ? slot({ row }) : String(row[slotName] ?? ''),
                        );
                      }),
                    ),
                  ),
                ),
              ])
            : h('div', [h('p', props.emptyTitle), h('p', props.emptyDescription)]),
        ]);
    },
  });

  const AdvancedQueryColumnDrawer = defineComponent({
    name: 'AdvancedQueryColumnDrawerStub',
    props: {
      columns: { type: Array, required: true },
      selectedKeys: { type: Array, required: true },
      visible: { type: Boolean, default: false },
    },
    emits: ['update:selectedKeys', 'update:visible'],
    setup(props, { emit }) {
      function hideColumn(value: string) {
        emit(
          'update:selectedKeys',
          (props.selectedKeys as string[]).filter((key) => key !== value),
        );
      }

      return () =>
        props.visible
          ? h(
              'div',
              { 'data-testid': 'column-drawer' },
              (props.columns as Array<{ value: string }>).map((column) =>
                h(
                  'button',
                  {
                    'data-testid': `hide-${column.value}`,
                    onClick: () => hideColumn(column.value),
                  },
                  `hide-${column.value}`,
                ),
              ),
            )
          : null;
    },
  });

  return {
    ...actual,
    AdvancedQueryColumnDrawer,
    AdvancedQueryPagedTable,
  };
});

vi.mock('../../components/ProjectImportInspectOverview.vue', async () => {
  const { defineComponent, h } = await import('vue');

  return {
    default: defineComponent({
      name: 'ProjectImportInspectOverviewStub',
      props: {
        canImport: { type: Boolean, default: false },
        result: { type: Object, required: true },
      },
      setup(props) {
        return () =>
          h('section', { 'data-testid': 'inspect-overview-stub' }, [
            h('h3', translate('project.import.preview.overviewTitle')),
            h('p', String((props.result as Record<string, unknown>).canonical_project_name ?? '')),
            h(
              'p',
              !props.canImport
                ? translate('project.import.preview.blockedDescription')
                : translate('project.import.preview.noDiagnostics'),
            ),
          ]);
      },
    }),
  };
});

vi.mock('../../components/ProjectImportInspectResources.vue', async () => {
  const { defineComponent, h } = await import('vue');

  return {
    default: defineComponent({
      name: 'ProjectImportInspectResourcesStub',
      props: {
        result: { type: Object, default: null },
      },
      setup(props) {
        return () =>
          h('section', { 'data-testid': 'inspect-resources-stub' }, [
            h('h3', translate('project.import.preview.resourcesTitle')),
            h(
              'p',
              Array.isArray((props.result as Record<string, unknown> | null)?.runtime_members) &&
                ((props.result as Record<string, unknown>).runtime_members as unknown[]).length === 0
                ? translate('project.import.preview.resources.empty.containers.description')
                : translate('project.import.preview.resources.tabs.containers'),
            ),
          ]);
      },
    }),
  };
});

function translate(key: string, params?: Record<string, unknown>) {
  const value = key.split('.').reduce<unknown>((current, segment) => {
    if (current && typeof current === 'object' && segment in current) {
      return (current as Record<string, unknown>)[segment];
    }
    return undefined;
  }, zhProjectLocale);

  if (typeof value !== 'string') {
    return key;
  }

  return value.replace(/\{(\w+)\}/g, (_match, name: string) => String(params?.[name] ?? ''));
}

function buildCandidate(overrides: Partial<ProjectImportRuntimeCandidate>): ProjectImportRuntimeCandidate {
  return {
    candidate_key: 'runtime:demo',
    canonical_project_name: 'demo',
    config_files: ['/srv/demo/compose.yaml'],
    container_counts: { running: 1, stopped: 0, total: 1 },
    importable: true,
    runtime_type: 'docker',
    runtime_version: '29.2.1',
    service_names: ['web', 'worker'],
    status: 'ready',
    status_reason_codes: [],
    warnings: [],
    working_directory: '/srv/demo',
    working_directory_source: 'runtime_label',
    ...overrides,
  };
}

function createFlowState() {
  return {
    canImport: ref(false),
    canonicalProjectNameOverride: ref(''),
    displayName: ref(''),
    hasPreview: ref(false),
    importError: ref(''),
    importLoading: ref(false),
    inspectCandidate: vi.fn(),
    inspectError: ref(''),
    inspectLoading: ref(false),
    inspectResult: ref(null),
    refreshInspect: vi.fn(),
    reset: vi.fn(),
    selectedCandidateKey: ref('runtime:demo'),
    submitImport: vi.fn(),
  };
}

const TButtonStub = defineComponent({
  name: 'TButtonStub',
  props: {
    disabled: { type: Boolean, default: false },
  },
  emits: ['click'],
  setup(props, { emit, slots }) {
    return () =>
      h(
        'button',
        {
          disabled: props.disabled,
          onClick: (event: MouseEvent) => emit('click', event),
        },
        slots.default?.(),
      );
  },
});

const TSelectStub = defineComponent({
  name: 'TSelectStub',
  props: {
    modelValue: { type: String, default: '' },
  },
  emits: ['update:modelValue', 'change'],
  setup(props, { emit, slots, attrs }) {
    return () =>
      h(
        'select',
        {
          ...attrs,
          value: props.modelValue,
          onChange: (event: Event) => {
            const value = (event.target as HTMLSelectElement).value;
            emit('update:modelValue', value);
            emit('change', value);
          },
        },
        slots.default?.(),
      );
  },
});

const TOptionStub = defineComponent({
  name: 'TOptionStub',
  props: {
    value: { type: String, required: true },
    label: { type: String, default: '' },
  },
  setup(props, { slots }) {
    return () => h('option', { value: props.value }, slots.default?.() ?? props.label);
  },
});

const TInputStub = defineComponent({
  name: 'TInputStub',
  props: {
    modelValue: { type: String, default: '' },
  },
  emits: ['update:modelValue'],
  setup(props, { emit, slots }) {
    return () =>
      h('label', [
        slots['prefix-icon']?.(),
        h('input', {
          value: props.modelValue,
          onInput: (event: Event) => emit('update:modelValue', (event.target as HTMLInputElement).value),
        }),
      ]);
  },
});

const WrapperStub = defineComponent({
  name: 'WrapperStub',
  setup(_props, { slots }) {
    return () => h('div', slots.default?.());
  },
});

const TCardStub = defineComponent({
  name: 'TCardStub',
  props: {
    title: { type: String, default: '' },
  },
  setup(props, { slots }) {
    return () => h('section', [props.title ? h('h2', props.title) : null, slots.default?.()]);
  },
});

const TAlertStub = defineComponent({
  name: 'TAlertStub',
  props: {
    message: { type: String, default: '' },
    title: { type: String, default: '' },
  },
  setup(props) {
    return () => h('div', [props.title, props.message]);
  },
});

const TEmptyStub = defineComponent({
  name: 'TEmptyStub',
  props: {
    description: { type: String, default: '' },
    title: { type: String, default: '' },
  },
  setup(props) {
    return () => h('div', [props.title, props.description]);
  },
});

const TDescriptionsItemStub = defineComponent({
  name: 'TDescriptionsItemStub',
  props: {
    label: { type: String, default: '' },
  },
  setup(props, { slots }) {
    return () => h('div', [h('strong', props.label), slots.default?.()]);
  },
});

const TTooltipStub = defineComponent({
  name: 'TTooltipStub',
  props: {
    content: { type: String, default: '' },
  },
  setup(props, { slots }) {
    return () => h('span', { class: 't-tooltip-stub', 'data-tooltip': props.content }, slots.default?.());
  },
});

const TTableStub = defineComponent({
  name: 'TTableStub',
  props: {
    columns: { type: Array, required: true },
    data: { type: Array, required: true },
  },
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
              (props.columns as Array<Record<string, unknown>>).map((column) => {
                const slotName = String(column.colKey);
                const slot = slots[slotName];
                return h('td', { 'data-col': slotName }, slot ? slot({ row }) : String(row[slotName] ?? ''));
              }),
            ),
          ),
        ),
      ]);
  },
});

const TPaginationStub = defineComponent({
  name: 'TPaginationStub',
  props: {
    current: { type: Number, default: 1 },
    pageSize: { type: Number, default: 10 },
    total: { type: Number, default: 0 },
  },
  emits: ['update:current', 'update:pageSize', 'change'],
  setup(props, { emit }) {
    const maxPage = () => Math.max(1, Math.ceil(props.total / props.pageSize));
    const changePage = (next: number) => {
      emit('update:current', next);
      emit('change', { current: next, previous: props.current, pageSize: props.pageSize });
    };
    const changePageSize = (next: number) => {
      emit('update:pageSize', next);
      emit('change', { current: 1, previous: props.current, pageSize: next });
    };

    return () =>
      h('div', { class: 't-pagination-stub' }, [
        h('span', { 'data-testid': 'runtime-members-page' }, `${props.current}/${maxPage()}`),
        h(
          'button',
          {
            'data-testid': 'runtime-members-next-page',
            disabled: props.current >= maxPage(),
            onClick: () => changePage(Math.min(maxPage(), props.current + 1)),
          },
          'next',
        ),
        h(
          'button',
          {
            'data-testid': 'runtime-members-page-size-5',
            onClick: () => changePageSize(5),
          },
          'size-5',
        ),
      ]);
  },
});

function mountPage() {
  return mount(ProjectImportIndex, {
    global: {
      stubs: {
        SearchIcon: true,
        't-alert': TAlertStub,
        't-button': TButtonStub,
        't-card': TCardStub,
        't-descriptions': WrapperStub,
        't-descriptions-item': TDescriptionsItemStub,
        't-empty': TEmptyStub,
        't-form': WrapperStub,
        't-form-item': WrapperStub,
        't-input': TInputStub,
        't-loading': WrapperStub,
        't-option': TOptionStub,
        't-pagination': TPaginationStub,
        't-select': TSelectStub,
        't-space': WrapperStub,
        't-steps': WrapperStub,
        't-table': TTableStub,
        't-tag': WrapperStub,
        't-tooltip': TTooltipStub,
      },
    },
  });
}

describe('ProjectImportIndex', () => {
  beforeEach(() => {
    routeState.query = {};
    mocks.getProjectImportRuntimeCandidates.mockReset();
    mocks.push.mockReset();
    mocks.replace.mockReset();
    mocks.useProjectImportFlow.mockReset();
    window.localStorage.clear();

    mocks.getProjectImportRuntimeCandidates.mockResolvedValue({
      items: [
        buildCandidate({}),
        buildCandidate({
          candidate_key: 'runtime:blocked',
          canonical_project_name: 'blocked',
          config_files: ['/srv/blocked/compose.yaml'],
          importable: false,
          service_names: ['api'],
          status: 'broken_compose',
          status_reason_codes: ['broken_compose'],
          warnings: ['working_directory_derived_from_config_files'],
          working_directory: '/srv/blocked',
          working_directory_source: 'derived_from_config_files',
        }),
      ],
      total: 2,
      limit: 10,
      offset: 0,
      filter_counts: {
        all: 2,
        ready: 1,
        unavailable: 1,
      },
    });
    const flowState = createFlowState();
    mocks.useProjectImportFlow.mockImplementation(() => flowState);
  });

  it('moves refresh and column settings into the table toolbar and removes the list back action', async () => {
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.get('[data-testid="table-refresh"]').text()).toContain('刷新候选');
    expect(wrapper.get('[data-testid="table-column-settings"]').text()).toContain('列设置');

    const header = wrapper.get('header');
    expect(header.text()).not.toContain('返回列表');
    expect(header.text()).not.toContain('刷新候选');
  });

  it('keeps the reason and operation columns visible while column settings hide optional columns', async () => {
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.find('th[data-col="services"]').exists()).toBe(true);
    expect(wrapper.find('th[data-col="reason"]').exists()).toBe(true);
    expect(wrapper.find('th[data-col="operation"]').exists()).toBe(true);

    await wrapper.get('[data-testid="table-column-settings"]').trigger('click');
    await nextTick();
    await wrapper.get('[data-testid="hide-services"]').trigger('click');
    await nextTick();

    expect(wrapper.find('th[data-col="services"]').exists()).toBe(false);
    expect(wrapper.find('th[data-col="reason"]').exists()).toBe(true);
    expect(wrapper.find('th[data-col="operation"]').exists()).toBe(true);
  });

  it('reloads runtime candidates from the server when the status filter changes', async () => {
    const wrapper = mountPage();
    await flushPromises();

    expect(mocks.getProjectImportRuntimeCandidates).toHaveBeenNthCalledWith(1, {
      keyword: undefined,
      availability: undefined,
      limit: 10,
      offset: 0,
    });

    await wrapper.get('[data-testid="candidate-status-filter"]').setValue('unavailable');
    await flushPromises();

    expect(mocks.getProjectImportRuntimeCandidates).toHaveBeenNthCalledWith(2, {
      keyword: undefined,
      availability: 'unavailable',
      limit: 10,
      offset: 0,
    });

    await wrapper.get('[data-testid="candidate-status-filter"]').setValue('ready');
    await flushPromises();

    expect(mocks.getProjectImportRuntimeCandidates).toHaveBeenNthCalledWith(3, {
      keyword: undefined,
      availability: 'ready',
      limit: 10,
      offset: 0,
    });
  });

  it('disables inspect actions for unavailable candidates while keeping the row visible', async () => {
    const flowState = createFlowState();
    mocks.useProjectImportFlow.mockImplementation(() => flowState);

    const wrapper = mountPage();
    await flushPromises();
    await wrapper.get('[data-testid="candidate-status-filter"]').setValue('unavailable');
    await flushPromises();

    const blockedRow = wrapper.findAll('tr').find((row) => row.text().includes('blocked'));
    const blockedInspectButton = blockedRow?.find('button');

    expect(blockedInspectButton).toBeDefined();
    expect(blockedInspectButton?.attributes('disabled')).toBeDefined();
    expect(wrapper.text()).toContain('Compose 文件无法通过导入检查');
    expect(flowState.inspectCandidate).not.toHaveBeenCalled();
  });

  it('normalizes nullable diagnostics arrays from the API response', async () => {
    mocks.getProjectImportRuntimeCandidates.mockResolvedValueOnce({
      items: [
        buildCandidate({
          candidate_key: 'runtime:null-arrays',
          canonical_project_name: 'null-arrays',
          importable: false,
          status: 'broken_compose',
          status_reason_codes: null as unknown as string[],
          warnings: null as unknown as string[],
        }),
      ],
      total: 1,
      limit: 10,
      offset: 0,
      filter_counts: {
        all: 1,
        ready: 0,
        unavailable: 1,
      },
    });

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain('null-arrays');
    expect(wrapper.text()).toContain('Compose 文件无法通过导入检查');
  });

  it('renders inspect preview safely when flow state contains nullable inspect arrays', async () => {
    routeState.query = {
      step: 'inspect',
      candidate: 'runtime:demo',
    };

    const flowState = createFlowState();
    flowState.hasPreview.value = true;
    flowState.canImport.value = true;
    flowState.inspectResult.value = {
      inspection_id: 'inspect-null',
      candidate_key: 'runtime:demo',
      directory_ref: { provider: 'local', root_id: 'managed-root', path: 'demo' },
      resolved_working_directory: '/srv/demo',
      canonical_project_name: 'demo',
      canonical_project_name_source: 'computed',
      display_name_suggested: 'Demo',
      compose_files: null,
      env_files: null,
      services: null,
      networks: null,
      volumes: null,
      runtime_members: null,
      warnings: null,
      conflicts: null,
      validation_status: 'ready',
      config_hash: 'hash-demo',
    } as never;
    flowState.inspectCandidate.mockResolvedValue('applied');
    mocks.useProjectImportFlow.mockImplementation(() => flowState);

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain('demo');
    expect(wrapper.text()).toContain('当前没有额外警告或冲突。');
    expect(wrapper.text()).toContain('检查概览');
    expect(wrapper.text()).toContain('资源');
    expect(wrapper.text()).toContain('当前候选没有可展示的容器资源。');
    expect(flowState.inspectCandidate).not.toHaveBeenCalled();
  });

  it('renders dedicated overview and resources sections in inspect preview', async () => {
    routeState.query = {
      step: 'inspect',
      candidate: 'runtime:demo',
    };

    const flowState = createFlowState();
    flowState.hasPreview.value = true;
    flowState.canImport.value = true;
    flowState.inspectResult.value = {
      inspection_id: 'inspect-preview',
      candidate_key: 'runtime:demo',
      resolved_working_directory: '/srv/projects/import-preview-example/very/long/path/for/compose/runtime/demo',
      canonical_project_name: 'demo',
      canonical_project_name_source: 'computed',
      display_name_suggested: 'Demo',
      compose_files: [
        {
          kind: 'compose',
          role: 'primary',
          absolute_path: '/srv/demo/compose.yaml',
          display_path: '/srv/demo/compose.yaml',
          order_index: 0,
          exists_on_last_refresh: true,
        },
      ],
      env_files: [],
      services: ['web', 'worker'],
      networks: ['default', 'internal'],
      volumes: ['data', 'cache'],
      runtime_members: [],
      warnings: [],
      conflicts: [],
      validation_status: 'ready',
      config_hash: 'hash-demo',
    } as never;
    flowState.inspectCandidate.mockResolvedValue('applied');
    mocks.useProjectImportFlow.mockImplementation(() => flowState);

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.get('[data-testid="inspect-overview-stub"]').text()).toContain('检查概览');
    expect(wrapper.get('[data-testid="inspect-resources-stub"]').text()).toContain('资源');
    expect(wrapper.text()).toContain('继续确认导入');
  });

  it('renders a review-focused confirm step from the existing inspect payload without re-inspecting', async () => {
    routeState.query = {
      step: 'confirm',
      candidate: 'runtime:demo',
    };

    const flowState = createFlowState();
    flowState.hasPreview.value = true;
    flowState.canImport.value = true;
    flowState.displayName.value = 'Demo Project';
    flowState.inspectResult.value = {
      inspection_id: 'inspect-confirm',
      candidate_key: 'runtime:demo',
      resolved_working_directory: '/srv/demo',
      canonical_project_name: 'demo',
      canonical_project_name_source: 'computed',
      display_name_suggested: 'Demo Project',
      compose_files: [
        {
          kind: 'compose',
          role: 'primary',
          absolute_path: '/srv/demo/compose.yaml',
          display_path: '/srv/demo/compose.yaml',
          order_index: 0,
          exists_on_last_refresh: true,
        },
      ],
      env_files: [],
      services: ['web', 'worker', 'cron', 'redis'],
      networks: ['default', 'internal'],
      volumes: ['data', 'cache'],
      runtime_members: [{ container_id: '1', container_name: 'demo-web-1', service_name: 'web', state: 'running' }],
      warnings: ['working directory derived from compose metadata'],
      conflicts: [],
      validation_status: 'ready',
      config_hash: 'hash-confirm',
    } as never;
    mocks.useProjectImportFlow.mockImplementation(() => flowState);

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain('最终确认');
    expect(wrapper.text()).toContain('项目标识');
    expect(wrapper.text()).toContain('运行时摘要');
    expect(wrapper.text()).toContain('检查摘要');
    expect(wrapper.text()).toContain('项目预览');
    expect(wrapper.text()).toContain('导入后将执行');
    expect(wrapper.text()).toContain('inspect-confirm');
    expect(wrapper.text()).toContain('web, worker, cron 等另外 1 项');
    expect(flowState.inspectCandidate).not.toHaveBeenCalled();
  });

  it('recovers a routed ready candidate even when it is outside the current page payload', async () => {
    routeState.query = {
      step: 'inspect',
      candidate: 'runtime:recover-me',
    };

    const flowState = createFlowState();
    flowState.inspectCandidate.mockResolvedValue('applied');
    mocks.useProjectImportFlow.mockImplementation(() => flowState);
    mocks.getProjectImportRuntimeCandidates
      .mockResolvedValueOnce({
        items: [buildCandidate({})],
        total: 25,
        limit: 10,
        offset: 0,
        filter_counts: {
          all: 25,
          ready: 25,
          unavailable: 0,
        },
      })
      .mockResolvedValueOnce({
        items: [
          buildCandidate({
            candidate_key: 'runtime:recover-me',
            canonical_project_name: 'recover-me',
            config_files: ['/srv/recover-me/compose.yaml'],
            service_names: ['web'],
            working_directory: '/srv/recover-me',
          }),
        ],
        total: 25,
        limit: 50,
        offset: 0,
        filter_counts: {
          all: 25,
          ready: 25,
          unavailable: 0,
        },
      });

    mountPage();
    await flushPromises();
    await flushPromises();

    expect(mocks.getProjectImportRuntimeCandidates).toHaveBeenNthCalledWith(2, {
      availability: 'ready',
      limit: 50,
      offset: 0,
    });
  });

  it('falls back to select and surfaces an error when routed candidate recovery fails', async () => {
    routeState.query = {
      step: 'inspect',
      candidate: 'runtime:recover-me',
    };

    const flowState = createFlowState();
    mocks.useProjectImportFlow.mockImplementation(() => flowState);
    mocks.getProjectImportRuntimeCandidates
      .mockResolvedValueOnce({
        items: [buildCandidate({})],
        total: 25,
        limit: 10,
        offset: 0,
        filter_counts: {
          all: 25,
          ready: 25,
          unavailable: 0,
        },
      })
      .mockRejectedValueOnce(new Error('route recovery failed'));

    const wrapper = mountPage();
    await flushPromises();
    await flushPromises();
    await nextTick();
    await flushPromises();

    expect(mocks.replace).toHaveBeenCalledWith({
      name: 'ProjectImportIndex',
      params: {},
      query: {},
    });
    expect(flowState.inspectCandidate).not.toHaveBeenCalled();
    expect(wrapper.text()).toContain('第 1 步 · 选择项目');
  });

  it('ignores stale candidate list responses when a newer filter request finishes first', async () => {
    const wrapper = mountPage();
    await flushPromises();

    let resolveUnavailable: (value: Record<string, unknown>) => void = () => {};
    let resolveReady: (value: Record<string, unknown>) => void = () => {};

    mocks.getProjectImportRuntimeCandidates
      .mockImplementationOnce(
        () =>
          new Promise((resolve) => {
            resolveUnavailable = resolve;
          }),
      )
      .mockImplementationOnce(
        () =>
          new Promise((resolve) => {
            resolveReady = resolve;
          }),
      );

    await wrapper.get('[data-testid="candidate-status-filter"]').setValue('unavailable');
    await nextTick();
    await wrapper.get('[data-testid="candidate-status-filter"]').setValue('ready');
    await nextTick();

    resolveReady({
      items: [
        buildCandidate({
          candidate_key: 'runtime:latest',
          canonical_project_name: 'latest',
        }),
      ],
      total: 1,
      limit: 10,
      offset: 0,
      filter_counts: {
        all: 1,
        ready: 1,
        unavailable: 0,
      },
    });
    await flushPromises();

    expect(wrapper.text()).toContain('latest');

    resolveUnavailable({
      items: [
        buildCandidate({
          candidate_key: 'runtime:stale',
          canonical_project_name: 'stale',
          importable: false,
          status: 'broken_compose',
          status_reason_codes: ['broken_compose'],
        }),
      ],
      total: 1,
      limit: 10,
      offset: 0,
      filter_counts: {
        all: 1,
        ready: 0,
        unavailable: 1,
      },
    });
    await flushPromises();

    expect(wrapper.text()).toContain('latest');
    expect(wrapper.text()).not.toContain('runtime:stale');
    expect(wrapper.text()).not.toContain('stale');
  });

  it('does not render visible English Inspect copy in the zh-CN flow', async () => {
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).not.toContain('Inspect');
    expect(wrapper.text()).toContain('检查');
  });
});
