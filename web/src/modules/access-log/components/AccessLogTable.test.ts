// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { shallowMount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import { defineComponent, h } from 'vue';
import { createI18n } from 'vue-i18n';

import type { AccessLogItem } from '../types/access-log';
import AccessLogTable from './AccessLogTable.vue';

const AdvancedQueryPagedTableStub = defineComponent({
  name: 'AdvancedQueryPagedTableStub',
  props: ['cellSlotNames', 'columns', 'rows'],
  emits: ['row-click'],
  setup(props, { emit, slots }) {
    return () =>
      h('section', { 'data-testid': 'paged-table' }, [
        h('div', { 'data-testid': 'table-toolbar' }, slots.toolbar?.()),
        h(
          'div',
          { 'data-testid': 'table-columns' },
          (props.columns ?? []).map((column: { colKey: string; fixed?: string }) =>
            h('span', { 'data-fixed': column.fixed ?? '' }, column.colKey),
          ),
        ),
        h('button', { 'data-testid': 'row-click', onClick: () => emit('row-click', accessLogRow()) }, 'open'),
        h(
          'div',
          { 'data-testid': 'operation-slot' },
          props.cellSlotNames?.includes('operation')
            ? slots.operation?.({ row: props.rows?.[0] ?? accessLogRow() })
            : undefined,
        ),
      ]);
  },
});

const translations: Record<string, string> = {
  'accessLog.columns.durationMs': '耗时',
  'accessLog.columns.method': '方法',
  'accessLog.columns.occurredAt': '发生时间',
  'accessLog.columns.operation': '操作',
  'accessLog.columns.path': '路径',
  'accessLog.columns.requestId': '请求 ID',
  'accessLog.columns.startedAt': '开始时间',
  'accessLog.columns.statusCode': '状态码',
  'accessLog.columns.user': '用户',
  'accessLog.columns.clientIp': '客户端 IP',
  'accessLog.columns.userAgent': '用户代理',
  'accessLog.actions.copyFail': '复制失败',
  'accessLog.actions.copyPath': '复制路径',
  'accessLog.actions.copyRequestId': '复制请求 ID',
  'accessLog.actions.copySuccess': '已复制',
  'accessLog.actions.detail': '详情',
  'accessLog.actions.more': '更多',
  'accessLog.actions.viewRelatedAppLogs': '查看关联应用日志',
  'accessLog.actions.viewRelatedAuditEvents': '查看关联审计事件',
  'accessLog.page.emptyTitle': '暂无访问日志',
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
          { 'data-testid': 'copy-request-id-action', onClick: () => emit('action', 'copy-request-id') },
          props.actions[1].label,
        ),
        h(
          'button',
          { 'data-testid': 'copy-path-action', onClick: () => emit('action', 'copy-path') },
          props.actions[2].label,
        ),
        h(
          'button',
          { 'data-testid': 'view-app-log-action', onClick: () => emit('action', 'view-app-log') },
          props.actions[3].label,
        ),
        h(
          'button',
          { 'data-testid': 'view-audit-action', onClick: () => emit('action', 'view-audit') },
          props.actions[4].label,
        ),
      ]);
  },
});

function accessLogRow(): AccessLogItem {
  return {
    duration_ms: 1,
    id: 1,
    method: 'GET',
    occurred_at: '2026-06-13T08:00:00Z',
    path: '/api/access-log',
    request_id: 'req-1',
    route: '',
    started_at: '2026-06-13T08:00:00Z',
    status_code: 200,
    user_id: 1,
    username: 'graft',
  } as AccessLogItem;
}

describe('AccessLogTable', () => {
  it('forwards the table toolbar slot into the shared paged table header', () => {
    const wrapper = shallowMount(AccessLogTable, {
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
        description: '列表说明',
        emptyDescription: '暂无数据',
        footerSummary: '共 1 条',
        pageSize: 20,
        rows: [accessLogRow()],
        summary: '当前 1 条',
        total: 1,
        visibleColumnKeys: ['started_at', 'method', 'path'],
      },
      slots: {
        toolbar: '<button data-testid="table-refresh">刷新</button>',
      },
    });

    expect(wrapper.get('[data-testid="table-toolbar"]').text()).toContain('刷新');
    expect(wrapper.get('[data-testid="table-refresh"]').text()).toBe('刷新');
  });

  it('keeps the fixed operation column while row click opens detail', async () => {
    const wrapper = shallowMount(AccessLogTable, {
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
        description: '列表说明',
        emptyDescription: '暂无数据',
        footerSummary: '共 1 条',
        pageSize: 20,
        rows: [accessLogRow()],
        summary: '当前 1 条',
        total: 1,
        visibleColumnKeys: ['started_at', 'method', 'path', 'status_code', 'duration_ms'],
      },
    });

    const operationColumn = wrapper.findAll('[data-testid="table-columns"] span').at(-1);
    expect(wrapper.get('[data-testid="table-columns"]').text()).toContain('operation');
    expect(operationColumn?.attributes('data-fixed')).toBe('right');

    await wrapper.get('[data-testid="row-click"]').trigger('click');

    expect(wrapper.emitted('detail')?.[0]?.[0]).toMatchObject({ id: 1, request_id: 'req-1' });
  });

  it('emits detail and related navigation operations from the row action menu', async () => {
    const wrapper = shallowMount(AccessLogTable, {
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
        description: '列表说明',
        emptyDescription: '暂无数据',
        footerSummary: '共 1 条',
        pageSize: 20,
        rows: [accessLogRow()],
        summary: '当前 1 条',
        total: 1,
        visibleColumnKeys: ['started_at', 'method', 'path'],
      },
    });

    expect(wrapper.get('[data-testid="action-menu"]').text()).toContain('详情');
    expect(wrapper.get('[data-testid="action-menu"]').text()).toContain('复制请求 ID');
    expect(wrapper.get('[data-testid="action-menu"]').text()).toContain('复制路径');
    expect(wrapper.get('[data-testid="action-menu"]').text()).toContain('查看关联应用日志');
    expect(wrapper.get('[data-testid="action-menu"]').text()).toContain('查看关联审计事件');

    await wrapper.get('[data-testid="detail-action"]').trigger('click');
    await wrapper.get('[data-testid="view-app-log-action"]').trigger('click');
    await wrapper.get('[data-testid="view-audit-action"]').trigger('click');

    expect(wrapper.emitted('detail')?.[0]?.[0]).toMatchObject({ id: 1, request_id: 'req-1' });
    expect(wrapper.emitted('view-app-log')?.[0]?.[0]).toMatchObject({ id: 1, request_id: 'req-1' });
    expect(wrapper.emitted('view-audit')?.[0]?.[0]).toMatchObject({ id: 1, request_id: 'req-1' });
  });
});
