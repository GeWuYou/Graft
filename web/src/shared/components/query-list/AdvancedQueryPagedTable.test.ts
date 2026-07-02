import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import AdvancedQueryPagedTable from './AdvancedQueryPagedTable.vue';

vi.mock('@/shared/components/management', () => {
  const ManagementPagedTable = defineComponent({
    name: 'ManagementPagedTableStub',
    props: ['rows'],
    emits: ['page-change', 'row-click', 'select-change', 'update:current', 'update:pageSize'],
    setup(props, { emit, slots }) {
      return () =>
        h('section', { 'data-testid': 'management-paged-table' }, [
          h('div', { 'data-testid': 'toolbar-slot' }, slots.toolbar?.()),
          h('div', { 'data-testid': 'pagination-slot' }, slots.pagination?.()),
          ...((props.rows as Array<Record<string, unknown>>) ?? []).map((row) =>
            h('article', { key: String(row.candidate_key), 'data-testid': `row-${row.candidate_key}` }, [
              h('div', { 'data-testid': `project-${row.candidate_key}` }, slots.project?.({ row })),
              h('div', { 'data-testid': `status-${row.candidate_key}` }, slots.status?.({ row })),
            ]),
          ),
          h(
            'button',
            {
              'data-testid': 'page-change',
              onClick: () => emit('page-change'),
            },
            'page',
          ),
        ]);
    },
  });

  return { ManagementPagedTable };
});
describe('AdvancedQueryPagedTable', () => {
  it('passes configured cell slots through to ManagementPagedTable', async () => {
    const wrapper = mount(AdvancedQueryPagedTable, {
      props: {
        cellSlotNames: ['project', 'status'],
        columns: [
          { colKey: 'project', title: 'Project' },
          { colKey: 'status', title: 'Status' },
        ],
        current: 1,
        description: 'Candidates',
        emptyDescription: 'No rows',
        emptyTitle: 'Empty',
        footerSummary: '1-1 / 1',
        headLabel: 'candidate-table',
        pageSize: 10,
        rows: [
          {
            candidate_key: 'runtime_demo',
            canonical_project_name: 'demo',
            status: 'ready',
          },
        ],
        total: 1,
      },
      slots: {
        toolbar: () => h('span', { 'data-testid': 'toolbar-content' }, 'toolbar'),
        pagination: () => h('span', { 'data-testid': 'pagination-content' }, 'pagination'),
        project: ({ row }: { row: { canonical_project_name: string } }) =>
          h('strong', { 'data-testid': 'project-content' }, row.canonical_project_name),
        status: ({ row }: { row: { status: string } }) => h('span', { 'data-testid': 'status-content' }, row.status),
      },
    });

    expect(wrapper.get('[data-testid="toolbar-content"]').text()).toBe('toolbar');
    expect(wrapper.get('[data-testid="pagination-content"]').text()).toBe('pagination');
    expect(wrapper.get('[data-testid="project-runtime_demo"]').text()).toContain('demo');
    expect(wrapper.get('[data-testid="status-runtime_demo"]').text()).toContain('ready');

    await wrapper.get('[data-testid="page-change"]').trigger('click');

    expect(wrapper.emitted('page-change')).toHaveLength(1);
  });
});
