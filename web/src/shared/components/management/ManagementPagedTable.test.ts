import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import { defineComponent, h } from 'vue';

import ManagementPagedTable from './ManagementPagedTable.vue';

const TTableStub = defineComponent({
  name: 'TTableStub',
  props: ['columns', 'data', 'selectedRowKeys'],
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
