import { flushPromises, mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';
import { createI18n } from 'vue-i18n';

import AuditLogsPage from './index.vue';

const auditApiMocks = vi.hoisted(() => ({
  getAuditLogs: vi.fn(async () => ({
    items: [
      {
        id: 1,
        actor_user_id: 1,
        actor_username: 'admin',
        actor_display_name: 'Admin',
        action: 'role.delete',
        resource_type: 'role',
        resource_id: '12',
        resource_name: 'Ops Admin',
        success: false,
        result: 'DENIED',
        risk_level: 'CRITICAL',
        target_type: 'ROLE',
        target_label: '角色',
        request_id: 'req-1',
        trace_id: 'trace-1',
        session_id: 'sess-1',
        ip: '127.0.0.1',
        user_agent: 'vitest',
        request_method: 'POST',
        request_path: '/api/roles/12/delete',
        status_code: 403,
        message: 'role removed',
        metadata: {
          trace_id: 'trace-1',
          session_id: 'sess-1',
        },
        created_at: '2026-05-27T08:00:00Z',
      },
    ],
    total: 1,
    page: 1,
    page_size: 10,
  })),
}));

vi.mock('../../api/audit', () => ({
  getAuditLogs: auditApiMocks.getAuditLogs,
}));

vi.mock('@/modules/shared/localized-api-error', () => ({
  resolveLocalizedErrorMessage: () => 'load failed',
}));

vi.mock('@/utils/logger', () => ({
  createLogger: () => ({
    error: vi.fn(),
  }),
}));

vi.mock('../../components/AuditFilters.vue', () => ({
  default: defineComponent({
    name: 'AuditFiltersStub',
    emits: ['search', 'reset', 'toggle-advanced', 'update:modelValue'],
    setup(_, { emit }) {
      return () =>
        h('div', [
          h('button', { 'data-testid': 'audit-search', onClick: () => emit('search') }, 'search'),
          h('button', { 'data-testid': 'audit-reset', onClick: () => emit('reset') }, 'reset'),
        ]);
    },
  }),
}));

vi.mock('../../components/AuditTable.vue', () => ({
  default: defineComponent({
    name: 'AuditTableStub',
    props: ['rows', 'summary', 'footerSummary'],
    emits: ['detail', 'update:current', 'update:pageSize', 'page-change'],
    setup(props, { emit }) {
      return () =>
        h('div', [
          props.summary,
          props.footerSummary,
          h('span', JSON.stringify(props.rows)),
          h('button', { 'data-testid': 'audit-detail', onClick: () => emit('detail', props.rows?.[0]) }, 'detail'),
        ]);
    },
  }),
}));

vi.mock('../../components/AuditDetailDrawer.vue', () => ({
  default: defineComponent({
    name: 'AuditDetailDrawerStub',
    props: ['visible', 'record'],
    setup(props) {
      return () => h('div', [String(props.visible), props.record?.request_id]);
    },
  }),
}));

vi.mock('vue-router', () => ({
  useRoute: () => ({
    query: {
      preset: 'permission-denied',
    },
  }),
}));

const passthroughStub = defineComponent({
  name: 'PassthroughStub',
  props: ['title', 'description'],
  setup(props, { slots }) {
    return () => h('div', [props.title, props.description, slots.default?.(), slots.actions?.()]);
  },
});

const buttonStub = defineComponent({
  name: 'TButtonStub',
  emits: ['click'],
  setup(_, { emit, slots, attrs }) {
    return () => h('button', { ...attrs, onClick: () => emit('click') }, slots.default?.());
  },
});

const i18n = createI18n({
  legacy: false,
  locale: 'en-US',
  messages: {
    'en-US': {
      menu: {
        audit: {
          title: 'Security Audit',
          logs: {
            title: 'Audit Logs',
          },
        },
      },
      components: {
        commonTable: {
          operation: 'Operation',
        },
      },
      audit: {
        common: {
          unknownActor: 'Anonymous',
          unknownResource: 'Unknown resource',
          result: { SUCCESS: 'Success', FAILED: 'Business Failed', DENIED: 'Denied', ERROR: 'System Error' },
          risk: { LOW: 'Low', MEDIUM: 'Medium', HIGH: 'High', CRITICAL: 'Critical' },
        },
        logList: {
          title: 'Audit Logs',
          description: 'Query system operation logs and inspect request context.',
          refresh: 'Refresh',
          retry: 'Retry',
          detail: 'View Details',
          more: 'More',
          summary: '{count} records shown',
          tableHint: 'Core fields only',
          footerTotal: '{count} logs total',
          footerFiltered: '{count} records matched on this page',
          currentPageFiltered: 'Current page filter',
          loadFailed: 'Failed to load audit logs',
          errorTitle: 'Audit logs are temporarily unavailable',
          emptyTitle: 'No audit logs',
          emptyDescription: 'Adjust filters and try again.',
          detailTitle: 'Audit Detail',
          presets: {
            all: 'All',
            todayAnomalies: "Today's Anomalies",
            permissionDenied: 'Permission Denied',
            sensitiveOps: 'Sensitive Operations',
            authFailed: 'Auth Failed',
            highRisk: 'High Risk',
          },
          actions: {
            search: 'Search',
            reset: 'Reset',
            showAdvanced: 'Advanced Filters',
            hideAdvanced: 'Hide Advanced',
          },
          filters: {
            keywordPlaceholder: 'Keyword',
            actorPlaceholder: 'Actor',
            actionPlaceholder: 'Action type',
            datePlaceholder: 'Time range',
            resourcePlaceholder: 'Target Object',
            resultPlaceholder: 'Result',
            riskPlaceholder: 'Risk',
            sessionPlaceholder: 'Session ID',
            traceIdPlaceholder: 'Trace ID',
          },
          filterOptions: {
            allActions: 'All actions',
            auth: 'Authentication',
            role: 'Role',
            permission: 'Permission',
            session: 'Session',
            allResults: 'All results',
            SUCCESS: 'Success',
            FAILED: 'Business Failed',
            DENIED: 'Denied',
            ERROR: 'System Error',
            allRisk: 'All risk',
            LOW: 'Low',
            MEDIUM: 'Medium',
            HIGH: 'High',
            CRITICAL: 'Critical',
          },
          columns: {
            action: 'Action',
            actor: 'Actor',
            resource: 'Target Object',
            result: 'Result',
            risk: 'Risk',
            createdAt: 'Time',
          },
          drawer: {
            messageFallback: 'No additional message',
            sections: {
              basic: 'Basic Info',
              request: 'Request Info',
              correlation: 'Correlation',
              risk: 'Risk',
              metadata: 'Metadata',
            },
            fields: {
              result: 'Result',
              requestId: 'Request ID',
              traceId: 'Trace ID',
              sessionId: 'Session ID',
              ip: 'IP',
              userAgent: 'User-Agent',
              method: 'Method',
              path: 'Path',
              status: 'Status',
              latency: 'Latency',
            },
            related: {
              sameRequest: 'Same Request Chain',
              sameActor: 'Recent Actions by Actor',
              sameResource: 'Recent Changes on Resource',
              empty: 'No more related records in the current list',
            },
            risk: {
              failedOperation: 'Failed operation',
              sensitiveOperation: 'Sensitive write',
              requestTrace: 'Request trace available',
            },
          },
        },
      },
    },
  },
});

describe('AuditLogsPage', () => {
  it('loads preset-backed records and opens the detail drawer', async () => {
    const wrapper = mount(AuditLogsPage, {
      global: {
        plugins: [i18n],
        stubs: {
          'management-empty-state': passthroughStub,
          'management-page-content': passthroughStub,
          'management-page-header': passthroughStub,
          't-button': buttonStub,
          't-space': passthroughStub,
        },
      },
    });

    await flushPromises();

    expect(auditApiMocks.getAuditLogs).toHaveBeenCalledWith(
      expect.objectContaining({
        result: 'DENIED',
      }),
    );
    expect(wrapper.text()).toContain('1 records shown');
    expect(wrapper.text()).toContain('req-1');

    await wrapper.get('[data-testid="audit-detail"]').trigger('click');
    await flushPromises();
    expect(wrapper.text()).toContain('true');
    expect(wrapper.text()).toContain('req-1');
  });
});
