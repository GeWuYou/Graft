import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import { defineComponent, h } from 'vue';

import ManagementPagedTable from './ManagementPagedTable.vue';

const TTableStub = defineComponent({
  name: 'TTableStub',
  props: ['columns', 'data'],
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
  props: ['current', 'pageSize', 'total'],
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

    await wrapper.get('[data-testid="pagination-change"]').trigger('click');

    expect(wrapper.emitted('page-change')?.[0]).toEqual([{ current: 2, pageSize: 20, previous: 1 }]);
  });
});
