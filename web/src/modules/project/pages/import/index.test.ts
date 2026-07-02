import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, inject, nextTick, provide, reactive, ref } from 'vue';

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

const RadioGroupContextKey = Symbol('radio-group');

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

const TRadioGroupStub = defineComponent({
  name: 'TRadioGroupStub',
  props: {
    value: { type: String, default: '' },
  },
  emits: ['update:value'],
  setup(props, { emit, slots }) {
    provide(RadioGroupContextKey, {
      select: (value: string) => emit('update:value', value),
      value: props.value,
    });
    return () => h('div', { class: 't-radio-group-stub' }, slots.default?.());
  },
});

const TRadioButtonStub = defineComponent({
  name: 'TRadioButtonStub',
  props: {
    value: { type: String, required: true },
  },
  setup(props, { slots }) {
    const context = inject<{ select: (value: string) => void }>(RadioGroupContextKey);
    return () =>
      h(
        'button',
        {
          'data-testid': `candidate-view-${props.value}`,
          onClick: () => context?.select(props.value),
        },
        slots.default?.(),
      );
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

const TDrawerStub = defineComponent({
  name: 'TDrawerStub',
  props: {
    footer: { type: Boolean, default: false },
    header: { type: String, default: '' },
    visible: { type: Boolean, default: false },
  },
  setup(props, { slots }) {
    return () => (props.visible ? h('aside', [h('h3', props.header), slots.default?.()]) : null);
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
        't-drawer': TDrawerStub,
        't-empty': TEmptyStub,
        't-form': WrapperStub,
        't-form-item': WrapperStub,
        't-input': TInputStub,
        't-loading': WrapperStub,
        't-radio-button': TRadioButtonStub,
        't-radio-group': TRadioGroupStub,
        't-space': WrapperStub,
        't-steps': WrapperStub,
        't-tag': WrapperStub,
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
    });
    mocks.useProjectImportFlow.mockImplementation(() => createFlowState());
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

  it('keeps the operation column visible while column settings hide optional ready columns', async () => {
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.find('th[data-col="services"]').exists()).toBe(true);
    expect(wrapper.find('th[data-col="operation"]').exists()).toBe(true);

    await wrapper.get('[data-testid="table-column-settings"]').trigger('click');
    await nextTick();
    await wrapper.get('[data-testid="hide-services"]').trigger('click');
    await nextTick();

    expect(wrapper.find('th[data-col="services"]').exists()).toBe(false);
    expect(wrapper.find('th[data-col="operation"]').exists()).toBe(true);
  });

  it('switches to unavailable candidates and opens the diagnostics drawer from the operation column', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('[data-testid="candidate-view-unavailable"]').trigger('click');
    await nextTick();

    expect(wrapper.find('th[data-col="reason"]').exists()).toBe(true);
    const diagnosticsButton = wrapper.findAll('button').find((button) => button.text().includes('查看诊断'));
    expect(diagnosticsButton).toBeDefined();

    await diagnosticsButton!.trigger('click');
    await nextTick();

    expect(wrapper.text()).toContain('当前阻塞原因');
    expect(wrapper.text()).toContain('更多诊断');
  });

  it('does not render visible English Inspect copy in the zh-CN flow', async () => {
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).not.toContain('Inspect');
    expect(wrapper.text()).toContain('检查');
  });
});
