import { shallowMount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import { defineComponent, h } from 'vue';
import { createI18n } from 'vue-i18n';

import type { AuditLogListItem } from '../types/audit';
import AuditTable from './AuditTable.vue';

const TTableStub = defineComponent({
  name: 'TTableStub',
  props: ['columns', 'data'],
  emits: ['row-click'],
  setup(props, { emit, slots }) {
    const row = props.data?.[0] ?? auditRow();
    return () =>
      h('section', { 'data-testid': 'table' }, [
        h(
          'div',
          { 'data-testid': 'table-columns' },
          (props.columns ?? []).map((column: { colKey: string; fixed?: string }) =>
            h('span', { 'data-fixed': column.fixed ?? '' }, column.colKey),
          ),
        ),
        h('button', { 'data-testid': 'row-click', onClick: () => emit('row-click', { row }) }, 'open'),
        h('div', { 'data-testid': 'operation-slot' }, slots.operation?.({ row })),
      ]);
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
          { 'data-testid': 'view-access-log-action', onClick: () => emit('action', 'view-access-log') },
          props.actions[2].label,
        ),
        h(
          'button',
          { 'data-testid': 'view-app-log-action', onClick: () => emit('action', 'view-app-log') },
          props.actions[3].label,
        ),
        h(
          'button',
          { 'data-testid': 'view-security-event-action', onClick: () => emit('action', 'view-security-event') },
          props.actions[4].label,
        ),
      ]);
  },
});

const passthroughStub = defineComponent({
  name: 'PassthroughStub',
  setup(_, { slots }) {
    return () => h('div', slots.default?.());
  },
});

const translations: Record<string, string> = {
  'audit.common.source.REQUEST': 'Audit Event',
  'audit.common.source.SECURITY_EVENT': 'Security Event',
  'audit.common.source.DOMAIN_EVENT': 'Domain Audit',
  'audit.common.source.UNKNOWN': 'Unknown',
  'audit.common.unknownActor': 'Anonymous',
  'audit.common.unknownResource': 'Unknown resource',
  'audit.common.result.SUCCESS': 'Success',
  'audit.common.result.DENIED': 'Denied',
  'audit.common.result.FAILED': 'Failed',
  'audit.common.result.ERROR': 'Error',
  'audit.common.risk.LOW': 'Low',
  'audit.common.risk.HIGH': 'High',
  'audit.common.targetType.permission': 'Permission',
  'audit.logList.columns.action': 'Event',
  'audit.logList.columns.actor': 'Actor',
  'audit.logList.columns.resource': 'Audit Target',
  'audit.logList.columns.correlation': 'Request ID',
  'audit.logList.columns.sessionId': 'Session ID',
  'audit.logList.columns.ip': 'IP',
  'audit.logList.columns.result': 'Result',
  'audit.logList.columns.risk': 'Risk',
  'audit.logList.columns.createdAt': 'Time',
  'audit.logList.columns.operation': 'Operation',
  'audit.logList.detail': 'Detail',
  'audit.logList.more': 'More',
  'audit.logList.currentPageFiltered': 'Current page filter',
  'audit.logList.emptyTitle': 'No audit logs',
  'audit.logList.emptyDescription': 'Adjust filters and try again.',
  'audit.logList.reasonFallback': 'No additional reason',
  'audit.logList.actions.viewAccessLog': 'View Access Log',
  'audit.logList.actions.viewAppLog': 'View App Log',
  'audit.logList.actions.viewSecurityEvent': 'View Security Event',
  'audit.logList.drawer.actions.copyRequestId': 'Copy Request ID',
  'audit.logList.drawer.actions.copyRequestIdSuccess': 'Request ID copied',
  'audit.logList.drawer.actions.copyRequestIdFail': 'Failed to copy Request ID',
  'audit.actionLabel.auth.permission.denied': 'Permission Denied',
};

const i18n = createI18n({
  legacy: false,
  missingWarn: false,
  fallbackWarn: false,
  locale: 'en-US',
  messages: {
    'en-US': translations,
  },
});

function auditRow(): AuditLogListItem {
  return {
    action: 'auth.permission.denied',
    actor_display_name: 'Admin',
    actor_username: 'admin',
    actor_user_id: 1,
    created_at: '2026-06-13T08:00:00Z',
    id: 1,
    request_id: 'req-1',
    resource_id: 'rbac.role.read',
    resource_name: 'rbac.role.read',
    resource_type: 'permission',
    result: 'DENIED',
    risk_level: 'HIGH',
    source: 'SECURITY_EVENT',
    success: false,
  } as AuditLogListItem;
}

function mountTable() {
  return shallowMount(AuditTable, {
    global: {
      plugins: [i18n],
      stubs: {
        ManagementTableCard: passthroughStub,
        ManagementTablePagination: passthroughStub,
        TableActionMenu: TableActionMenuStub,
        TEmpty: passthroughStub,
        TPagination: passthroughStub,
        TTable: TTableStub,
        TTag: passthroughStub,
      },
    },
    props: {
      current: 1,
      footerSummary: '1 event',
      pageSize: 10,
      rows: [auditRow()],
      total: 1,
      visibleColumnKeys: ['action', 'actor', 'resource'],
    },
  });
}

describe('AuditTable', () => {
  it('keeps the fixed operation column while row click opens detail', async () => {
    const wrapper = mountTable();

    const operationColumn = wrapper.findAll('[data-testid="table-columns"] span').at(-1);
    expect(wrapper.get('[data-testid="table-columns"]').text()).toContain('operation');
    expect(operationColumn?.attributes('data-fixed')).toBe('right');
    expect(wrapper.find('.audit-log-table__host').classes()).toContain('graft-scrollbar');

    await wrapper.get('[data-testid="row-click"]').trigger('click');

    expect(wrapper.emitted('detail')?.[0]?.[0]).toMatchObject({ id: 1, request_id: 'req-1' });
  });

  it('emits non-destructive related log and raw JSON actions from the action menu', async () => {
    const wrapper = mountTable();

    expect(wrapper.get('[data-testid="action-menu"]').text()).toContain('Detail');
    expect(wrapper.get('[data-testid="action-menu"]').text()).toContain('Copy Request ID');
    expect(wrapper.get('[data-testid="action-menu"]').text()).toContain('View Access Log');
    expect(wrapper.get('[data-testid="action-menu"]').text()).toContain('View App Log');
    expect(wrapper.get('[data-testid="action-menu"]').text()).toContain('View Security Event');
    expect(wrapper.get('[data-testid="action-menu"]').text()).not.toContain('View Raw JSON');
    expect(wrapper.get('[data-testid="action-menu"]').text()).not.toContain('Delete');

    await wrapper.get('[data-testid="detail-action"]').trigger('click');
    await wrapper.get('[data-testid="view-access-log-action"]').trigger('click');
    await wrapper.get('[data-testid="view-app-log-action"]').trigger('click');
    await wrapper.get('[data-testid="view-security-event-action"]').trigger('click');

    expect(wrapper.emitted('detail')?.[0]?.[0]).toMatchObject({ id: 1 });
    expect(wrapper.emitted('view-access-log')?.[0]?.[0]).toMatchObject({ request_id: 'req-1' });
    expect(wrapper.emitted('view-app-log')?.[0]?.[0]).toMatchObject({ request_id: 'req-1' });
    expect(wrapper.emitted('view-security-event')?.[0]?.[0]).toMatchObject({ request_id: 'req-1' });
    expect(wrapper.emitted('delete')).toBeUndefined();
  });
});
