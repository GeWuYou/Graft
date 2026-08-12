import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, nextTick } from 'vue';

import RegistryListPage from './index.vue';

const apiMocks = vi.hoisted(() => ({
  deleteRegistry: vi.fn(),
  getRegistries: vi.fn(),
  verifyRegistry: vi.fn(),
}));

vi.mock('../../api/registry', () => ({
  createRegistry: vi.fn(),
  createRegistryRepository: vi.fn(),
  deleteRegistry: apiMocks.deleteRegistry,
  deleteRegistryRepository: vi.fn(),
  getRegistries: apiMocks.getRegistries,
  getRegistryRepositories: vi.fn(),
  getRegistryRepositoryAssignments: vi.fn(),
  grantRegistryRepositoryAssignment: vi.fn(),
  revokeRegistryRepositoryAssignment: vi.fn(),
  updateRegistry: vi.fn(),
  updateRegistryRepository: vi.fn(),
  verifyRegistry: apiMocks.verifyRegistry,
}));
vi.mock('@/shared/components/management', () => ({
  createActionColumn: (title: string, width: number) => ({ colKey: 'actions', fixed: 'right', title, width }),
  ManagementPageContent: defineComponent({
    name: 'ManagementPageContent',
    setup(_props, { slots }) {
      return () => h('div', slots.default?.());
    },
  }),
  ManagementPageHeader: defineComponent({
    name: 'ManagementPageHeader',
    setup(_props, { slots }) {
      return () => h('div', [slots.default?.(), slots.actions?.()]);
    },
  }),
  TableActionMenu: defineComponent({
    name: 'TableActionMenu',
    props: { actions: { type: Array, default: () => [] } },
    emits: ['action'],
    setup(_props, { emit }) {
      return () => h('button', { 'data-testid': 'registry-row-actions', onClick: () => emit('action', 'delete') });
    },
  }),
  TableViewToolbar: defineComponent({
    name: 'TableViewToolbar',
    setup(_props, { slots }) {
      return () => h('div', slots.default?.());
    },
  }),
}));
vi.mock('@/shared/components/query-list', () => ({
  ResourceQueryPanel: defineComponent({
    name: 'ResourceQueryPanel',
    props: {
      config: { type: Object, required: true },
      loading: { type: Boolean, default: false },
      modelValue: { type: Object, required: true },
    },
    emits: ['reset', 'search', 'update:modelValue'],
    setup() {
      return () => h('div');
    },
  }),
}));
vi.mock('@/shared/components/management/ManagementPagedTable.vue', () => ({
  default: defineComponent({
    name: 'ManagementPagedTable',
    props: {
      columns: { type: Array, default: () => [] },
      current: { type: Number, default: 1 },
      footerSummary: { type: String, default: '' },
      pageSize: { type: Number, default: 20 },
      rows: { type: Array, default: () => [] },
      total: { type: Number, default: 0 },
    },
    emits: ['page-change', 'update:current', 'update:pageSize'],
    setup(props, { slots }) {
      return () =>
        h('div', [
          slots.feedback?.(),
          slots.toolbar?.(),
          ...(props.rows as Array<Record<string, unknown>>).flatMap((row) => [
            slots.credential?.({ row }),
            slots.status?.({ row }),
            slots.verified?.({ row }),
            slots.actions?.({ row }),
          ]),
          slots['empty-action']?.(),
        ]);
    },
  }),
}));
vi.mock('@/shared/localized-api-error', () => ({
  resolveLocalizedErrorMessage: (_t: unknown, _error: unknown, fallback: string) => fallback,
}));
vi.mock('@/shared/observability', () => ({ formatLocaleDateTime: () => '' }));
vi.mock('vue-i18n', () => ({
  useI18n: () => ({ locale: { value: 'zh-CN' }, t: (key: string) => key }),
}));

const passthrough = (name: string) =>
  defineComponent({
    name,
    setup(_props, { slots }) {
      return () => h('div', slots.default?.());
    },
  });

const buttonStub = defineComponent({
  name: 'TButton',
  emits: ['click'],
  setup(_props, { emit, slots }) {
    return () => h('button', { type: 'button', onClick: () => emit('click') }, slots.default?.());
  },
});

const drawerStub = defineComponent({
  name: 'TDrawer',
  props: { visible: { type: Boolean, default: false } },
  setup(props, { slots }) {
    return () => h('aside', { 'data-visible': String(props.visible) }, slots.default?.());
  },
});

const tableStub = defineComponent({
  name: 'TTable',
  props: { columns: { type: Array, default: () => [] }, data: { type: Array, default: () => [] }, loading: Boolean },
  setup(props, { slots }) {
    return () =>
      h('div', [
        ...(props.data as Array<Record<string, unknown>>).flatMap((row) => [
          slots.status?.({ row }),
          slots.verified?.({ row }),
          slots.created_at?.({ row }),
        ]),
        slots.default?.(),
      ]);
  },
});

const dialogStub = defineComponent({
  name: 'TDialog',
  props: { visible: { type: Boolean, default: false } },
  emits: ['confirm'],
  setup() {
    return () => h('div');
  },
});

function mountPage() {
  return mount(RegistryListPage, {
    global: {
      stubs: {
        't-alert': passthrough('TAlert'),
        't-button': buttonStub,
        't-drawer': drawerStub,
        't-dialog': dialogStub,
        't-empty': passthrough('TEmpty'),
        't-form': passthrough('TForm'),
        't-form-item': passthrough('TFormItem'),
        't-input': passthrough('TInput'),
        't-input-number': passthrough('TInputNumber'),
        't-popconfirm': passthrough('TPopconfirm'),
        't-space': passthrough('TSpace'),
        't-switch': passthrough('TSwitch'),
        't-table': tableStub,
        't-tag': passthrough('TTag'),
        't-textarea': passthrough('TTextarea'),
      },
    },
  });
}

describe('RegistryListPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    apiMocks.getRegistries.mockResolvedValue({ items: [], total: 0 });
  });

  it('opens the Connection create drawer from the management action', async () => {
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.findAll('aside').at(0)?.attributes('data-visible')).toBe('false');
    await wrapper.get('[data-testid="registry-create"]').trigger('click');

    expect(wrapper.findAll('aside').at(0)?.attributes('data-visible')).toBe('true');
  });

  it('uses the shared resource query panel with the registry keyword query contract', async () => {
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.findComponent({ name: 'ResourceQueryPanel' }).props('config')).toMatchObject({
      resource: 'registry.list',
      search: true,
      filterBuilder: { enabled: false },
      placeholder: 'registry.list.search',
    });
  });

  it('resets pagination when resetting the query and queries the first page', async () => {
    const wrapper = mountPage();
    await flushPromises();

    const pagination = wrapper.findComponent({ name: 'ManagementPagedTable' });
    pagination.vm.$emit('page-change', { current: 3, pageSize: 10 });
    await flushPromises();
    wrapper.findComponent({ name: 'ResourceQueryPanel' }).vm.$emit('reset', {
      keyword: '',
      filters: {},
      page: 1,
      pageSize: 10,
    });
    await flushPromises();

    expect(apiMocks.getRegistries).toHaveBeenLastCalledWith({ limit: 10, offset: 0, search: undefined });
  });

  it('applies a keyword query through the shared resource query panel', async () => {
    const wrapper = mountPage();
    await flushPromises();

    wrapper.findComponent({ name: 'ResourceQueryPanel' }).vm.$emit('search', {
      keyword: 'registry-a',
      filters: {},
      page: 1,
      pageSize: 10,
    });
    await flushPromises();

    expect(apiMocks.getRegistries).toHaveBeenLastCalledWith({ limit: 10, offset: 0, search: 'registry-a' });
  });

  it('keeps the newest page response when an older request resolves later', async () => {
    let resolveFirst: ((value: { items: Array<{ connection_ref: string }>; total: number }) => void) | undefined;
    let resolveSecond: ((value: { items: Array<{ connection_ref: string }>; total: number }) => void) | undefined;
    apiMocks.getRegistries.mockReset();
    apiMocks.getRegistries
      .mockImplementationOnce(
        () =>
          new Promise((resolve) => {
            resolveFirst = resolve;
          }),
      )
      .mockImplementationOnce(
        () =>
          new Promise((resolve) => {
            resolveSecond = resolve;
          }),
      );

    const wrapper = mountPage();
    await nextTick();
    wrapper.findComponent({ name: 'ResourceQueryPanel' }).vm.$emit('search', {
      keyword: 'newer',
      filters: {},
      page: 1,
      pageSize: 20,
    });
    await nextTick();
    resolveSecond?.({ items: [{ connection_ref: 'newer' }], total: 1 });
    await flushPromises();
    resolveFirst?.({ items: [{ connection_ref: 'older' }], total: 1 });
    await flushPromises();

    expect(wrapper.findComponent({ name: 'ManagementPagedTable' }).props('rows')).toEqual([
      { connection_ref: 'newer' },
    ]);
  });

  it('uses the shared paged table with registry identity, summary, and standardized row actions', async () => {
    apiMocks.getRegistries.mockResolvedValue({
      items: [{ connection_ref: 'registry-a', verification_status: 'unknown' }],
      total: 1,
    });
    const wrapper = mountPage();
    await flushPromises();

    const pagedTable = wrapper.findComponent({ name: 'ManagementPagedTable' });
    expect(pagedTable.props('rows')).toEqual([{ connection_ref: 'registry-a', verification_status: 'unknown' }]);
    expect(pagedTable.props('total')).toBe(1);
    expect(pagedTable.props('footerSummary')).toBe('registry.list.summary');
    expect(pagedTable.props('columns')).toEqual(
      expect.arrayContaining([expect.objectContaining({ colKey: 'actions' })]),
    );
    expect(wrapper.findComponent({ name: 'TableActionMenu' }).props('actions')).toEqual([
      expect.objectContaining({ value: 'edit' }),
      expect.objectContaining({ value: 'verify' }),
      expect.objectContaining({ danger: true, value: 'delete' }),
    ]);
  });

  it('requires the page-level danger dialog before deleting a registry', async () => {
    apiMocks.getRegistries.mockResolvedValue({ items: [{ connection_ref: 'registry-a' }], total: 1 });
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('[data-testid="registry-row-actions"]').trigger('click');
    expect(apiMocks.deleteRegistry).not.toHaveBeenCalled();
    wrapper.findComponent(dialogStub).vm.$emit('confirm');
    await flushPromises();

    expect(apiMocks.deleteRegistry).toHaveBeenCalledWith('registry-a');
  });
});
