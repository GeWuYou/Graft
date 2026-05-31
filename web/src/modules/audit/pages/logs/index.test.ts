import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';
import { createI18n } from 'vue-i18n';
import { createMemoryHistory, createRouter } from 'vue-router';

import { resolveAuditPresetKey } from '../../contract/presets';
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
    props: ['presets', 'activePreset', 'modelValue'],
    emits: ['search', 'reset', 'apply-preset', 'update:modelValue'],
    setup(props, { emit }) {
      return () =>
        h('div', [
          h('span', { 'data-testid': 'audit-filter-model' }, JSON.stringify(props.modelValue)),
          h('button', { 'data-testid': 'audit-search', onClick: () => emit('search') }, 'search'),
          h('button', { 'data-testid': 'audit-reset', onClick: () => emit('reset') }, 'reset'),
          h('button', { 'data-testid': 'audit-preset', onClick: () => emit('apply-preset', 'high-risk') }, 'preset'),
          h(
            'button',
            {
              'data-testid': 'audit-route-sync',
              onClick: () =>
                emit('update:modelValue', {
                  ...props.modelValue,
                  actor: 'route-admin',
                  actorUserId: '7',
                  success: 'all',
                  createdRange: ['2026-05-01 10:00:00', '2026-05-02 18:30:00'],
                  actionPrefixes: [],
                  actionKeywords: [],
                  requestPathPrefixes: [],
                  resourceTypes: [],
                  result: 'FAILED',
                  results: [],
                  sorters: [{ field: 'created_at', direction: 'asc' }],
                  riskLevels: [],
                }),
            },
            'sync-route',
          ),
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
    props: ['visible', 'record', 'monitorOrigin'],
    setup(props) {
      return () => h('div', [String(props.visible), props.record?.request_id, JSON.stringify(props.monitorOrigin)]);
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
          description:
            'Query security audit events and inspect request and permission context; health checks, monitor polling, and bootstrap requests are not part of the default audit dataset.',
          refresh: 'Refresh',
          columnSettings: 'Columns',
          retry: 'Retry',
          detail: 'View Details',
          more: 'More',
          footerTotal: '{count} audit events total',
          footerFiltered: '{count} records matched on this page',
          currentPageFiltered: 'Current page filter',
          loadFailed: 'Failed to load audit logs',
          errorTitle: 'Audit logs are temporarily unavailable',
          emptyTitle: 'No audit logs',
          emptyDescription: 'Adjust filters and try again.',
          detailTitle: 'Audit Detail',
          presets: {
            all: 'All',
            failedOperations: 'Failed Operations',
            todayAnomalies: "Today's Security Anomalies",
            rbacChanges: 'Permission Configuration Changes',
            permissionDenied: 'Permission Denied',
            sensitiveOps: 'Sensitive Operations',
            authFailed: 'Auth Failed',
            highRisk: 'High Risk',
          },
          actions: {
            search: 'Search',
            reset: 'Reset',
            backToMonitor: 'Back to monitor',
            addFilter: 'Add filter',
            showAdvanced: 'Advanced Filters',
            hideAdvanced: 'Hide Advanced',
          },
          filters: {
            keywordPlaceholder: 'Keyword: action, request ID, audit target, operated object',
            actorPlaceholder: 'Actor',
            actionPlaceholder: 'Action type',
            actionPrefixesPlaceholder: 'Select action groups',
            actionKeywordsPlaceholder: 'Type an action keyword and press Enter',
            successPlaceholder: 'Success state',
            datePlaceholder: 'Time range',
            sourcePlaceholder: 'Source',
            resourceTypePlaceholder: 'Audit target type',
            resourceTypesPlaceholder: 'Select target type set',
            resourceNamePlaceholder: 'Audit target / target name',
            resourceIdPlaceholder: 'Resource ID',
            resultPlaceholder: 'Result',
            resultsPlaceholder: 'Select result set',
            riskPlaceholder: 'Risk',
            riskLevelsPlaceholder: 'Select risk level set',
            sessionPlaceholder: 'Session ID',
            requestIdPlaceholder: 'Request ID',
            requestPathPrefixesPlaceholder: 'Type a request path prefix and press Enter',
          },
          builder: {
            title: 'Filter fields',
            hint: 'Choose a field and set its value. Active conditions appear as removable tags.',
            fields: {
              success: 'Success state',
              action: 'Action type',
              actionPrefixes: 'Action groups',
              actionKeywords: 'Action keywords',
              result: 'Result',
              results: 'Result set',
              riskLevel: 'Risk level',
              riskLevels: 'Risk level set',
              source: 'Event type',
              actor: 'Actor',
              resourceName: 'Audit target',
              resourceType: 'Target type',
              resourceTypes: 'Target type set',
              requestPathPrefixes: 'Request path prefixes',
              requestId: 'Request ID',
              session: 'Session ID',
              resourceId: 'Resource ID',
            },
          },
          filterOptions: {
            allActions: 'All actions',
            allSource: 'All source',
            allResourceTypes: 'All target types',
            auth: 'Authentication',
            authPrefix: 'Authentication actions',
            rbacPrefix: 'Permission configuration actions',
            role: 'Role',
            rolePrefix: 'Role actions',
            permission: 'Permission',
            permissionPrefix: 'Permission actions',
            session: 'Session',
            userResource: 'User',
            roleResource: 'Role',
            permissionResource: 'Permission',
            authResource: 'Authentication',
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
            resource: 'Audit Target',
            correlation: 'Request ID',
            result: 'Result',
            risk: 'Risk',
            createdAt: 'Time',
          },
          drawer: {
            messageFallback: 'No additional message',
            sections: {
              basic: 'Event Summary',
              request: 'Request Context',
              correlation: 'Related Context',
              risk: 'Risk',
              metadata: 'Metadata',
            },
            fields: {
              target: 'Audit Target',
              result: 'Result',
              requestId: 'Request ID',
              sessionId: 'Session ID',
              ip: 'IP',
              userAgent: 'User-Agent',
              method: 'Method',
              path: 'Path',
              status: 'Status',
            },
            related: {
              sameRequest: 'Same Request ID',
              sameActor: 'Recent Actions by Actor',
              sameResource: 'Recent Changes on Audit Target',
              empty: 'No more related records in the current list',
            },
            risk: {
              failedOperation: 'Failed operation',
              sensitiveOperation: 'Sensitive write',
              requestTrace: 'Request available',
            },
            actions: {
              viewRelatedRequest: 'View Related Request',
              copyMetadata: 'Copy JSON',
              copyMetadataSuccess: 'Metadata JSON copied',
              copyMetadataFail: 'Failed to copy metadata JSON',
              toggleMetadata: 'Show raw metadata',
            },
            metadataEmpty: 'No metadata is available for this event.',
          },
        },
      },
    },
  },
});

describe('AuditLogsPage', () => {
  beforeEach(() => {
    auditApiMocks.getAuditLogs.mockClear();
  });

  async function mountPage(initialQuery: Record<string, string> = { preset: 'last_24h', results: 'DENIED' }) {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/audit/logs', component: AuditLogsPage }],
    });

    await router.push({
      path: '/audit/logs',
      query: initialQuery,
    });
    await router.isReady();

    const replaceSpy = vi.spyOn(router, 'replace');
    const wrapper = mount(AuditLogsPage, {
      global: {
        plugins: [i18n, router],
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
    return { router, replaceSpy, wrapper };
  }

  it('restores deep-link filters including created range and keeps backend request shape unchanged', async () => {
    const { wrapper } = await mountPage({
      username: 'alice',
      created_from: '2026-05-01T10:00:00Z',
      created_to: '2026-05-02T18:30:00Z',
      result: 'FAILED',
    });

    expect(wrapper.get('[data-testid="audit-filter-model"]').text()).toContain('"actor":"alice"');
    expect(wrapper.get('[data-testid="audit-filter-model"]').text()).toContain(
      '"createdRange":["2026-05-01 18:00:00","2026-05-03 02:30:00"]',
    );
    expect(auditApiMocks.getAuditLogs).toHaveBeenLastCalledWith({
      page: 1,
      page_size: 10,
      result: 'FAILED',
      created_from: '2026-05-01T10:00:00.000Z',
      created_to: '2026-05-02T18:30:00.000Z',
      sort_by: 'created_at',
      sort_order: 'desc',
    });
  });

  it('accepts legacy occurred range query while normalizing route writes to created range', async () => {
    const { replaceSpy, wrapper } = await mountPage({
      occurred_from: '2026-05-01T10:00:00Z',
      occurred_to: '2026-05-02T18:30:00Z',
    });

    expect(wrapper.get('[data-testid="audit-filter-model"]').text()).toContain(
      '"createdRange":["2026-05-01 18:00:00","2026-05-03 02:30:00"]',
    );

    replaceSpy.mockClear();
    await wrapper.get('[data-testid="audit-search"]').trigger('click');
    await flushPromises();

    expect(replaceSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        query: expect.objectContaining({
          created_from: '2026-05-01T10:00:00.000Z',
          created_to: '2026-05-02T18:30:00.000Z',
        }),
      }),
    );
  });

  it('loads preset-backed records and opens the detail drawer', async () => {
    const { wrapper } = await mountPage();

    expect(auditApiMocks.getAuditLogs).toHaveBeenCalledWith(
      expect.objectContaining({
        preset: 'last_24h',
        results: ['DENIED'],
        sort_by: 'created_at',
        sort_order: 'desc',
      }),
    );
    expect(wrapper.text()).not.toContain('security audit records shown');
    expect(wrapper.text()).not.toContain('Core fields only');
    expect(wrapper.text()).toContain('req-1');

    await wrapper.get('[data-testid="audit-detail"]').trigger('click');
    await flushPromises();
    expect(wrapper.text()).toContain('true');
    expect(wrapper.text()).toContain('req-1');
  });

  it('keeps monitor return context when syncing log filters', async () => {
    const { replaceSpy, router, wrapper } = await mountPage({
      preset: 'last_24h',
      results: 'DENIED',
      monitorView: 'overview',
      monitorTrendRange: '10m',
      monitorAnomalyKey: 'resource_cpu_pressure',
      monitorScopeRef: 'runtime:cpu',
    });

    auditApiMocks.getAuditLogs.mockClear();
    replaceSpy.mockClear();

    await wrapper.get('[data-testid="audit-route-sync"]').trigger('click');
    await wrapper.get('[data-testid="audit-search"]').trigger('click');
    await flushPromises();

    expect(replaceSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        path: '/audit/logs',
        query: expect.objectContaining({
          monitorView: 'overview',
          monitorTrendRange: '10m',
          monitorAnomalyKey: 'resource_cpu_pressure',
          monitorScopeRef: 'runtime:cpu',
        }),
      }),
    );
    expect(router.currentRoute.value.query).toMatchObject({
      monitorView: 'overview',
      monitorTrendRange: '10m',
      monitorAnomalyKey: 'resource_cpu_pressure',
      monitorScopeRef: 'runtime:cpu',
    });
    expect(wrapper.text()).toContain('"view":"overview"');
  });

  it('applies quick preset from filters and refetches with unchanged query contract', async () => {
    const { wrapper } = await mountPage();
    auditApiMocks.getAuditLogs.mockClear();

    await wrapper.get('[data-testid="audit-preset"]').trigger('click');
    await flushPromises();

    expect(auditApiMocks.getAuditLogs).toHaveBeenCalledWith({
      page: 1,
      page_size: 10,
      preset: 'last_24h',
      risk_levels: ['HIGH', 'CRITICAL'],
      sort_by: 'created_at',
      sort_order: 'desc',
    });
  });

  it('does not send an implicit preset when the route has no time range', async () => {
    const { wrapper } = await mountPage({});

    expect(auditApiMocks.getAuditLogs).toHaveBeenLastCalledWith({
      page: 1,
      page_size: 10,
      sort_by: 'created_at',
      sort_order: 'desc',
    });
    expect(wrapper.text()).toContain('req-1');
  });

  it('syncs interactive filters into route query for reload and sharing', async () => {
    const { replaceSpy, router, wrapper } = await mountPage();
    auditApiMocks.getAuditLogs.mockClear();
    replaceSpy.mockClear();

    await wrapper.get('[data-testid="audit-route-sync"]').trigger('click');
    await wrapper.get('[data-testid="audit-search"]').trigger('click');
    await flushPromises();

    expect(replaceSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        path: '/audit/logs',
        query: expect.objectContaining({
          username: 'route-admin',
          user_id: '7',
          created_from: '2026-05-01T02:00:00.000Z',
          created_to: '2026-05-02T10:30:00.000Z',
          result: 'FAILED',
          sort_by: 'created_at',
          sort_order: 'asc',
        }),
      }),
    );
    expect(router.currentRoute.value.query).toMatchObject({
      username: 'route-admin',
      user_id: '7',
      created_from: '2026-05-01T02:00:00.000Z',
      created_to: '2026-05-02T10:30:00.000Z',
      result: 'FAILED',
      sort_by: 'created_at',
      sort_order: 'asc',
    });
    expect(auditApiMocks.getAuditLogs).toHaveBeenLastCalledWith(
      expect.objectContaining({
        result: 'FAILED',
        created_from: '2026-05-01T02:00:00.000Z',
        created_to: '2026-05-02T10:30:00.000Z',
        sort_by: 'created_at',
        sort_order: 'asc',
      }),
    );
  });

  it('maps legacy overview preset keys to the canonical local preset authority', async () => {
    expect(resolveAuditPresetKey('failed-auth')).toBe('auth-failed');
    expect(resolveAuditPresetKey('rbac-changes')).toBe('rbac-changes');

    const { replaceSpy, router, wrapper } = await mountPage({
      preset: 'last_24h',
      success: 'false',
      resource_types: 'auth,session',
      action_keywords: 'auth,login',
      request_path_prefixes: '/api/auth',
    });
    replaceSpy.mockClear();
    auditApiMocks.getAuditLogs.mockClear();

    expect(wrapper.get('[data-testid="audit-filter-model"]').text()).not.toContain('"resourceType":"auth"');
    expect(wrapper.get('[data-testid="audit-filter-model"]').text()).toContain('"result":"all"');

    await wrapper.get('[data-testid="audit-search"]').trigger('click');
    await flushPromises();

    expect(replaceSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        path: '/audit/logs',
        query: expect.objectContaining({
          preset: 'last_24h',
          success: 'false',
          resource_types: 'auth,session',
          action_keywords: 'auth,login',
          request_path_prefixes: '/api/auth',
        }),
      }),
    );
    expect(router.currentRoute.value.query).toMatchObject({
      preset: 'last_24h',
      success: 'false',
      resource_types: 'auth,session',
      action_keywords: 'auth,login',
      request_path_prefixes: '/api/auth',
    });
    expect(auditApiMocks.getAuditLogs).toHaveBeenLastCalledWith(
      expect.objectContaining({
        preset: 'last_24h',
        success: false,
        resource_types: ['auth', 'session'],
        action_keywords: ['auth', 'login'],
        request_path_prefixes: ['/api/auth'],
      }),
    );
  });

  it('keeps explicit filters empty when rewriting a plain preset-only route query', async () => {
    const { replaceSpy, wrapper } = await mountPage({ preset: 'last_24h' });

    replaceSpy.mockClear();
    await wrapper.get('[data-testid="audit-search"]').trigger('click');
    await flushPromises();

    expect(replaceSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        query: expect.objectContaining({
          preset: 'last_24h',
        }),
      }),
    );
  });

  it('preserves failed-operations explicit filters from overview drilldown', async () => {
    const { replaceSpy, wrapper } = await mountPage({ preset: 'last_24h', success: 'false' });

    replaceSpy.mockClear();
    await wrapper.get('[data-testid="audit-search"]').trigger('click');
    await flushPromises();

    expect(replaceSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        query: expect.objectContaining({
          preset: 'last_24h',
          success: 'false',
        }),
      }),
    );
    expect(auditApiMocks.getAuditLogs).toHaveBeenLastCalledWith(
      expect.objectContaining({
        preset: 'last_24h',
        success: false,
      }),
    );
  });

  it('applies the canonical rbac preset to the backend query contract', async () => {
    const { wrapper } = await mountPage({ preset: 'last_24h', action_prefixes: 'rbac.,role.,permission.' });

    expect(wrapper.get('[data-testid="audit-filter-model"]').text()).not.toContain('"actionPrefix":"rbac."');
    expect(auditApiMocks.getAuditLogs).toHaveBeenLastCalledWith(
      expect.objectContaining({
        preset: 'last_24h',
        action_prefixes: ['rbac.', 'role.', 'permission.'],
      }),
    );
  });
});
