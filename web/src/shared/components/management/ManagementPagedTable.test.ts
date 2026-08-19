import { mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, nextTick, ref } from 'vue';

import ManagementPagedTable from './ManagementPagedTable.vue';

const tableHostWidth = ref(0);
const debugMocks = vi.hoisted(() => ({ emitDebugLog: vi.fn(), isDebugFlagEnabled: vi.fn(() => false) }));

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    locale: ref('zh-CN'),
    t: (key: string) =>
      ({
        'components.commonTable.columnSettings': '列设置',
        'components.commonTable.compactDensity': '紧凑密度',
        'components.commonTable.refresh': '刷新',
        'components.commonTable.resetColumns': '恢复默认列',
        'components.commonTable.standardDensity': '标准密度',
      })[key] ?? key,
  }),
}));

vi.mock('./use-table-host-width', () => ({
  useTableHostWidth: () => ({
    tableHostRef: ref(null),
    tableHostWidth,
  }),
}));
vi.mock('@/shared/debug/runtime', () => debugMocks);

const TTableStub = defineComponent({
  name: 'TTableStub',
  props: ['columns', 'data', 'selectedRowKeys', 'size', 'tableContentWidth'],
  emits: ['page-change', 'row-click', 'select-change', 'sort-change'],
  setup(props, { slots }) {
    return () =>
      h('div', { 'data-testid': 'paged-table' }, [
        h(
          'div',
          { 'data-testid': 'table-column-keys' },
          JSON.stringify((props.columns as Array<{ colKey: string }>).map((column) => column.colKey)),
        ),
        (props.data as Array<Record<string, unknown>>).map((row) =>
          h('div', { key: String(row.id) }, [slots.name?.({ row }), slots.operation?.({ row })]),
        ),
        h('div', { 'data-testid': 'table-empty' }, slots.empty?.()),
      ]);
  },
});

const TPaginationStub = defineComponent({
  name: 'TPaginationStub',
  props: ['current', 'pageSize', 'total', 'totalContent'],
  emits: ['change', 'update:current', 'update:pageSize'],
  setup(_props, { emit }) {
    return () =>
      h(
        'button',
        {
          'data-testid': 'pagination-change',
          onClick: () => emit('change', { current: 2, pageSize: 20, previous: 1 }),
        },
        'page',
      );
  },
});

describe('ManagementPagedTable', () => {
  beforeEach(() => {
    tableHostWidth.value = 0;
  });

  it('uses host-width columns for empty tables and restores wide columns with rows', async () => {
    const wrapper = mount(ManagementPagedTable, {
      global: { stubs: { 't-pagination': TPaginationStub, 't-table': TTableStub } },
      props: {
        columns: [
          { colKey: 'name', title: 'Name', width: 600 },
          { colKey: 'repository', title: 'Repository', width: 600 },
        ],
        current: 1,
        emptyDescription: 'No rows',
        emptyTitle: 'Empty',
        footerSummary: '0-0 / 0',
        pageSize: 10,
        rows: [],
        total: 0,
      },
    });

    await nextTick();

    expect(wrapper.get('.management-paged-table__table-host').attributes('data-table-mode')).toBe('fill');
    expect(wrapper.get('.management-paged-table__table-host').classes()).not.toContain('graft-scrollbar--horizontal');
    expect(wrapper.findComponent(TTableStub).props('tableContentWidth')).toBeUndefined();
    expect(wrapper.findComponent(TTableStub).props('columns')).toEqual([
      { colKey: 'name', title: 'Name' },
      { colKey: 'repository', title: 'Repository' },
    ]);

    tableHostWidth.value = 960;
    await nextTick();

    expect(wrapper.findComponent(TTableStub).props('tableContentWidth')).toBeUndefined();

    tableHostWidth.value = 840;
    await nextTick();

    expect(wrapper.findComponent(TTableStub).props('columns')).toEqual([
      { colKey: 'name', title: 'Name' },
      { colKey: 'repository', title: 'Repository' },
    ]);

    await wrapper.setProps({ rows: [{ id: 'build-1', name: 'web', repository: 'graft' }] });

    expect(wrapper.get('.management-paged-table__table-host').attributes('data-table-mode')).toBe('scroll');
    expect(wrapper.get('.management-paged-table__table-host').classes()).toContain('graft-scrollbar--horizontal');
    expect(wrapper.findComponent(TTableStub).props('tableContentWidth')).toBe('1200px');
    expect(wrapper.findComponent(TTableStub).props('columns')).toEqual([
      { colKey: 'name', title: 'Name', width: 600 },
      { colKey: 'repository', title: 'Repository', width: 600 },
    ]);
  });

  it('provides refresh, column settings, and labeled density controls by default', async () => {
    const wrapper = mount(ManagementPagedTable, {
      global: { stubs: { 't-pagination': TPaginationStub, 't-table': TTableStub } },
      props: {
        columns: [
          { colKey: 'name', title: 'Name' },
          { colKey: 'status', title: 'Status' },
        ],
        current: 2,
        emptyDescription: 'No rows',
        emptyTitle: 'Empty',
        footerSummary: '1-1 / 1',
        pageSize: 20,
        rows: [{ id: 'container-1', name: 'web', status: 'running' }],
        total: 1,
      },
    });

    const toolbar = wrapper.findComponent({ name: 'TableViewToolbar' });
    expect(toolbar.exists()).toBe(true);
    expect(toolbar.text()).toContain('紧凑密度');

    await toolbar.get('[aria-label="刷新"]').trigger('click');
    expect(wrapper.emitted('page-change')?.[0]).toEqual([{ current: 2, pageSize: 20, previous: 2 }]);

    await toolbar.get('[aria-label="紧凑密度"]').trigger('click');
    expect(wrapper.findComponent(TTableStub).props('size')).toBe('small');
    expect(toolbar.text()).toContain('标准密度');

    await toolbar.get('[aria-label="列设置"]').trigger('click');
    expect(wrapper.findComponent({ name: 'AdvancedQueryColumnDrawer' }).props('visible')).toBe(true);
  });

  it('routes table cell slots and renders the shared empty/pagination frame', async () => {
    const wrapper = mount(ManagementPagedTable, {
      global: {
        stubs: {
          't-empty': defineComponent({
            props: ['title', 'description'],
            setup:
              (props, { slots }) =>
              () =>
                h('div', { 'data-testid': 'empty-state' }, [props.title, props.description, slots.action?.()]),
          }),
          't-pagination': TPaginationStub,
          't-table': TTableStub,
        },
      },
      props: {
        columns: [
          { colKey: 'name', title: 'Name' },
          { colKey: 'operation', title: 'Operation' },
        ],
        current: 1,
        emptyDescription: 'No rows',
        emptyTitle: 'Empty',
        footerSummary: '1-1 / 1',
        pageSize: 10,
        rows: [{ id: 'container-1', name: 'web' }],
        total: 1,
      },
      slots: {
        name: ({ row }: { row: { name: string } }) => h('span', { 'data-testid': 'name-cell' }, row.name),
        operation: () => h('button', { 'data-testid': 'operation-cell' }, 'detail'),
      },
    });

    expect(wrapper.get('[data-testid="name-cell"]').text()).toBe('web');
    expect(wrapper.get('[data-testid="operation-cell"]').text()).toBe('detail');
    expect(wrapper.findComponent(TPaginationStub).props('totalContent')).toBe(false);

    await wrapper.get('[data-testid="pagination-change"]').trigger('click');

    expect(wrapper.emitted('page-change')?.[0]).toEqual([{ current: 2, pageSize: 20, previous: 1 }]);
  });

  it('forwards the undefined sort value emitted when sorting is cleared', async () => {
    const wrapper = mount(ManagementPagedTable, {
      global: { stubs: { 't-pagination': TPaginationStub, 't-table': TTableStub } },
      props: {
        columns: [{ colKey: 'name', title: 'Name' }],
        current: 1,
        emptyDescription: 'No rows',
        emptyTitle: 'Empty',
        footerSummary: '0-0 / 0',
        pageSize: 10,
        rows: [],
        total: 0,
      },
    });

    wrapper.findComponent(TTableStub).vm.$emit('sort-change', undefined);
    await nextTick();

    expect(wrapper.emitted('sort-change')).toEqual([[undefined]]);
  });

  it('cancels debug measurement work deferred past component unmount', async () => {
    debugMocks.isDebugFlagEnabled.mockReturnValue(true);
    const wrapper = mount(ManagementPagedTable, {
      global: { stubs: { 't-pagination': TPaginationStub, 't-table': TTableStub } },
      props: {
        columns: [{ colKey: 'name', title: 'Name' }],
        current: 1,
        emptyDescription: 'No rows',
        emptyTitle: 'Empty',
        footerSummary: '0-0 / 0',
        pageSize: 10,
        rows: [],
        total: 0,
      },
    });

    debugMocks.emitDebugLog.mockClear();
    wrapper.unmount();
    await nextTick();

    expect(debugMocks.emitDebugLog).not.toHaveBeenCalled();
  });

  it('lets callers override the default pagination total content', () => {
    const wrapper = mount(ManagementPagedTable, {
      global: { stubs: { 't-pagination': TPaginationStub, 't-table': TTableStub } },
      props: {
        columns: [{ colKey: 'name', title: 'Name' }],
        current: 1,
        emptyDescription: 'No rows',
        emptyTitle: 'Empty',
        footerSummary: '1-1 / 1',
        pageSize: 10,
        paginationProps: { totalContent: true },
        rows: [{ id: 'container-1', name: 'web' }],
        total: 1,
      },
    });

    expect(wrapper.findComponent(TPaginationStub).props('totalContent')).toBe(true);
  });

  it('forwards compact footer-summary visibility to the shared pagination frame', () => {
    const wrapper = mount(ManagementPagedTable, {
      global: { stubs: { 't-pagination': TPaginationStub, 't-table': TTableStub } },
      props: {
        columns: [{ colKey: 'name', title: 'Name' }],
        current: 1,
        emptyDescription: 'No rows',
        emptyTitle: 'Empty',
        footerSummary: '1-1 / 1',
        hideFooterSummaryOnCompact: true,
        pageSize: 10,
        rows: [{ id: 'application-1', name: 'web' }],
        total: 1,
      },
    });

    expect(wrapper.findComponent({ name: 'ManagementTablePagination' }).props('hideSummaryOnCompact')).toBe(true);
  });

  it('replaces the table with an explicit card slot without changing the shared frame', () => {
    const wrapper = mount(ManagementPagedTable, {
      global: {
        stubs: {
          't-pagination': TPaginationStub,
          't-table': TTableStub,
        },
      },
      props: {
        cardsVisible: true,
        columns: [{ colKey: 'name', title: 'Name' }],
        current: 1,
        emptyDescription: 'No rows',
        emptyTitle: 'Empty',
        footerSummary: '1-1 / 1',
        pageSize: 10,
        rows: [{ id: 'application-1', name: 'web' }],
        total: 1,
      },
      slots: {
        cards: '<article data-testid="application-card">web</article>',
      },
    });

    expect(wrapper.get('[data-testid="application-card"]').text()).toBe('web');
    expect(wrapper.find('[data-testid="paged-table"]').exists()).toBe(false);
    expect(wrapper.find('.management-table-pagination').exists()).toBe(true);
  });

  it('hides the pagination footer only when callers explicitly opt out', () => {
    const wrapper = mount(ManagementPagedTable, {
      global: { stubs: { 't-pagination': TPaginationStub, 't-table': TTableStub } },
      props: {
        columns: [{ colKey: 'name', title: 'Name' }],
        current: 1,
        emptyDescription: 'No rows',
        emptyTitle: 'Empty',
        footerSummary: '1-1 / 1',
        pageSize: 10,
        paginationVisible: false,
        rows: [{ id: 'container-1', name: 'web' }],
        total: 1,
      },
    });

    expect(wrapper.find('.management-table-pagination').exists()).toBe(false);
  });

  it('passes responsive entity semantics and density-specific column sets to the shared table boundary', () => {
    const wrapper = mount(ManagementPagedTable, {
      global: { stubs: { 't-pagination': TPaginationStub, 't-table': TTableStub } },
      props: {
        columnSets: { comfortable: ['name', 'operation'], spacious: ['select', 'name', 'operation'] },
        columns: [
          { colKey: 'select', title: 'Select' },
          { colKey: 'name', title: 'Name' },
          { colKey: 'operation', title: 'Operation' },
        ],
        current: 1,
        emptyDescription: 'No rows',
        emptyTitle: 'Empty',
        footerSummary: '1-1 / 1',
        pageSize: 10,
        presentation: 'entity',
        rows: [{ id: 'application-1', name: 'web' }],
        total: 1,
      },
      slots: { cards: '<article data-testid="application-card">web</article>' },
    });

    expect(wrapper.findComponent({ name: 'ResponsiveTable' }).props('presentation')).toBe('entity');
    expect(wrapper.findComponent({ name: 'ResponsiveTable' }).props('entityCardLayout')).toBe('compact');
  });

  it('derives entity cards from cardsVisible while preserving the caller data presentation default', () => {
    const wrapper = mount(ManagementPagedTable, {
      global: { stubs: { 't-pagination': TPaginationStub, 't-table': TTableStub } },
      props: {
        cardsVisible: true,
        columns: [{ colKey: 'name', title: 'Name' }],
        current: 1,
        emptyDescription: 'No rows',
        emptyTitle: 'Empty',
        footerSummary: '1-1 / 1',
        pageSize: 10,
        rows: [{ id: 'application-1', name: 'web' }],
        total: 1,
      },
      slots: { cards: '<article data-testid="application-card">web</article>' },
    });

    expect(wrapper.findComponent({ name: 'ResponsiveTable' }).props('presentation')).toBe('entity');
  });

  it('keeps compact entity-card selection out of the hidden table renderer', () => {
    const ResponsiveTableStub = defineComponent({
      name: 'ResponsiveTable',
      setup(_props, { slots }) {
        return () => slots.default?.({ variant: { density: 'compact' } });
      },
    });
    const wrapper = mount(ManagementPagedTable, {
      global: {
        stubs: { 't-pagination': TPaginationStub, 't-table': TTableStub, ResponsiveTable: ResponsiveTableStub },
      },
      props: {
        columns: [{ colKey: 'name', title: 'Name' }],
        current: 1,
        emptyDescription: 'No rows',
        emptyTitle: 'Empty',
        footerSummary: '1-1 / 1',
        pageSize: 10,
        presentation: 'entity',
        rows: [{ id: 'user-1', name: 'Admin' }],
        selectedRowKeys: ['user-1'],
        total: 1,
      },
    });

    expect(wrapper.findComponent(TTableStub).props('selectedRowKeys')).toEqual([]);
  });
});
