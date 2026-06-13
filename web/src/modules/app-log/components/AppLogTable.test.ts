// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { shallowMount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';
import { createI18n } from 'vue-i18n';

import type { AppLogItem } from '../types/app-log';
import AppLogTable from './AppLogTable.vue';

vi.mock('@/store', () => ({
  usePermissionStore: () => ({
    hasPermission: (code: string) => code === 'app_log.delete',
  }),
}));

const AdvancedQueryPagedTableStub = defineComponent({
  name: 'AdvancedQueryPagedTableStub',
  props: ['cellSlotNames', 'columns', 'rows'],
  emits: ['row-click', 'select-change'],
  setup(props, { emit, slots }) {
    return () =>
      h('section', { 'data-testid': 'paged-table' }, [
        h('div', { 'data-testid': 'table-toolbar' }, slots.toolbar?.()),
        h('div', { 'data-testid': 'table-batch' }, slots.batch?.()),
        h(
          'div',
          { 'data-testid': 'table-columns' },
          (props.columns ?? []).map((column: { colKey: string }) => h('span', column.colKey)),
        ),
        h('button', { 'data-testid': 'row-click', onClick: () => emit('row-click', appLogRow()) }, 'open'),
        h('button', { 'data-testid': 'select-change', onClick: () => emit('select-change', [1]) }, 'select'),
        h(
          'div',
          { 'data-testid': 'actions-slot' },
          props.cellSlotNames?.includes('actions')
            ? slots.actions?.({ row: props.rows?.[0] ?? appLogRow() })
            : undefined,
        ),
      ]);
  },
});

const translations: Record<string, string> = {
  'appLog.columns.actions': '操作',
  'appLog.columns.component': '组件',
  'appLog.columns.correlation': '关联字段',
  'appLog.columns.fields': '字段',
  'appLog.columns.message': '消息',
  'appLog.columns.occurredAt': '发生时间',
  'appLog.columns.operation': '操作',
  'appLog.columns.requestId': '请求 ID',
  'appLog.columns.severity': '级别',
  'appLog.actions.copyFail': '复制失败',
  'appLog.actions.copySuccess': '已复制',
  'appLog.actions.delete': '删除',
  'appLog.actions.detail': '详情',
  'appLog.actions.more': '更多',
  'appLog.page.emptyTitle': '暂无应用日志',
};

const i18n = createI18n({
  legacy: false,
  locale: 'zh-CN',
  messages: {
    'zh-CN': translations,
  },
});

const TTagStub = defineComponent({
  name: 'TTagStub',
  setup(_props, { slots }) {
    return () => h('span', slots.default?.());
  },
});

const TableActionMenuStub = defineComponent({
  name: 'TableActionMenuStub',
  props: ['actions'],
  emits: ['action'],
  setup(props, { emit }) {
    return () =>
      h('div', { 'data-testid': 'action-menu' }, [
        h(
          'button',
          { 'data-testid': 'detail-action', onClick: () => emit('action', 'detail') },
          props.actions[0].label,
        ),
        h(
          'button',
          { 'data-testid': 'delete-action', onClick: () => emit('action', 'delete') },
          props.actions[1].label,
        ),
      ]);
  },
});

function appLogRow(): AppLogItem {
  return {
    component: 'internal.dashboard',
    error: '',
    fields: {},
    id: 1,
    message: 'dashboard widget loaded',
    method: 'GET',
    occurred_at: '2026-06-13T08:00:00Z',
    operation: 'dashboard_widget_load',
    request_id: 'req-1',
    severity: 'debug',
  } as AppLogItem;
}

describe('AppLogTable', () => {
  it('forwards the table toolbar slot into the shared paged table header', () => {
    const wrapper = shallowMount(AppLogTable, {
      global: {
        plugins: [i18n],
        stubs: {
          AdvancedQueryPagedTable: AdvancedQueryPagedTableStub,
          TableActionMenu: TableActionMenuStub,
          TTag: TTagStub,
        },
      },
      props: {
        current: 1,
        emptyDescription: '暂无数据',
        footerSummary: '共 1 条',
        pageSize: 20,
        rows: [appLogRow()],
        total: 1,
        visibleColumnKeys: ['occurred_at', 'severity', 'component'],
      },
      slots: {
        toolbar: '<button data-testid="table-refresh">刷新</button>',
      },
    });

    expect(wrapper.get('[data-testid="table-toolbar"]').text()).toContain('刷新');
    expect(wrapper.get('[data-testid="table-refresh"]').text()).toBe('刷新');
  });

  it('keeps selection and fixed action columns while row click opens detail', async () => {
    const wrapper = shallowMount(AppLogTable, {
      global: {
        plugins: [i18n],
        stubs: {
          AdvancedQueryPagedTable: AdvancedQueryPagedTableStub,
          TableActionMenu: TableActionMenuStub,
          TTag: TTagStub,
        },
      },
      props: {
        current: 1,
        emptyDescription: '暂无数据',
        footerSummary: '共 1 条',
        pageSize: 20,
        rows: [appLogRow()],
        total: 1,
        visibleColumnKeys: ['occurred_at', 'severity', 'component', 'operation', 'message'],
      },
    });

    const columnText = wrapper.get('[data-testid="table-columns"]').text();
    expect(columnText).toContain('row-select');
    expect(columnText).toContain('actions');

    await wrapper.get('[data-testid="row-click"]').trigger('click');
    await wrapper.get('[data-testid="select-change"]').trigger('click');

    expect(wrapper.emitted('detail')?.[0]?.[0]).toMatchObject({ id: 1, operation: 'dashboard_widget_load' });
    expect(wrapper.emitted('select-change')?.[0]?.[0]).toEqual([1]);
  });

  it('emits detail and delete operations from the action menu without a raw JSON row action', async () => {
    const wrapper = shallowMount(AppLogTable, {
      global: {
        plugins: [i18n],
        stubs: {
          AdvancedQueryPagedTable: AdvancedQueryPagedTableStub,
          TableActionMenu: TableActionMenuStub,
          TTag: TTagStub,
        },
      },
      props: {
        current: 1,
        emptyDescription: '暂无数据',
        footerSummary: '共 1 条',
        pageSize: 20,
        rows: [appLogRow()],
        total: 1,
        visibleColumnKeys: ['occurred_at', 'severity', 'component', 'operation', 'message'],
      },
    });

    await wrapper.get('[data-testid="detail-action"]').trigger('click');
    await wrapper.get('[data-testid="delete-action"]').trigger('click');

    expect(wrapper.emitted('detail')?.[0]?.[0]).toMatchObject({ id: 1 });
    expect(wrapper.emitted('delete')?.[0]?.[0]).toMatchObject({ id: 1 });
    expect(wrapper.find('[data-testid="raw-json-action"]').exists()).toBe(false);
    expect(wrapper.emitted('raw-json')).toBeUndefined();
  });
});
