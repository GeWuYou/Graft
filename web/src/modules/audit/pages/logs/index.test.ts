// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mount } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, KeepAlive, resolveComponent } from 'vue';
import { createI18n } from 'vue-i18n';
import { createMemoryHistory, createRouter } from 'vue-router';

import { localDateTimeToUtcIso, normalizeRouteRangeForPageState } from '@/shared/observability';

import type { AuditLogListResponse } from '../../types/audit';
import AuditLogsPage from './index.vue';

const { getAuditLogsMock } = vi.hoisted(() => ({
  getAuditLogsMock: vi.fn(),
}));

function createAuditLogsResponse(overrides: Partial<AuditLogListResponse> = {}): AuditLogListResponse {
  return {
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
        target: {
          kind: 'resource',
          type: 'role',
          id: '12',
          label: 'Ops Admin',
        },
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
    applied_scope: undefined,
    scope_projection: undefined,
    convertible_filters: undefined,
    ...overrides,
  };
}

vi.mock('../../api/audit', () => ({
  getAuditLogs: getAuditLogsMock,
}));

vi.mock('@/shared/localized-api-error', () => ({
  resolveLocalizedErrorMessage: () => 'load failed',
}));

vi.mock('@/utils/logger', () => ({
  createLogger: () => ({
    debug: vi.fn(),
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
            { 'data-testid': 'audit-sensitive-preset', onClick: () => emit('apply-preset', 'sensitive-ops') },
            'sensitive-preset',
          ),
          h(
            'button',
            { 'data-testid': 'audit-security-preset', onClick: () => emit('apply-preset', 'security-events') },
            'security-preset',
          ),
          h(
            'button',
            {
              'data-testid': 'audit-route-sync',
              onClick: () =>
                emit('update:modelValue', {
                  ...props.modelValue,
                  actor: 'route-admin',
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

const checkboxGroupStub = defineComponent({
  name: 'TCheckboxGroupStub',
  setup(_, { slots }) {
    return () => h('div', slots.default?.());
  },
});

const checkboxStub = defineComponent({
  name: 'TCheckboxStub',
  setup(_, { slots }) {
    return () => h('label', slots.default?.());
  },
});

const drawerStub = defineComponent({
  name: 'TDrawerStub',
  props: ['visible', 'header'],
  setup(_, { slots }) {
    return () => h('div', slots.default?.());
  },
});

const tagStub = defineComponent({
  name: 'TTagStub',
  setup(_, { slots }) {
    return () => h('span', slots.default?.());
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
            securityEvents: 'Security Events',
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
          scope: {
            drilldownTag: 'Drilldown: {name}',
            conditionInline: 'Condition: {condition}',
            exitAction: 'Exit drilldown',
            convertAction: 'Convert to normal filters',
            unknownValue: 'Unnamed condition',
          },
          businessCategory: {
            sensitiveOperations: 'Sensitive operations',
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
              businessCategory: 'Business category',
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
              security: 'Security Event Context',
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
              eventType: 'Event Type',
              permission: 'Permission',
              securityTarget: 'Security Target',
              traceId: 'Trace ID',
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
              securityEvent: 'Security Event',
            },
            actions: {
              viewRelatedRequest: 'View Related Request',
              viewAccessLogRequest: 'View Access Log',
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
    getAuditLogsMock.mockReset();
    getAuditLogsMock.mockResolvedValue(createAuditLogsResponse());
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  async function mountPage(
    initialQuery: Record<string, string> = {
      created_from: '2026-05-30T07:21:04.000Z',
      created_to: '2026-05-31T07:21:04.000Z',
      results: 'DENIED',
    },
  ) {
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
          't-checkbox': checkboxStub,
          't-checkbox-group': checkboxGroupStub,
          't-drawer': drawerStub,
          't-space': passthroughStub,
          't-tag': tagStub,
        },
      },
    });

    await flushPromises();
    return { router, replaceSpy, wrapper };
  }

  async function mountKeepAliveHost(initialQuery: Record<string, string> = {}) {
    const OtherPage = defineComponent({
      name: 'OtherPageStub',
      setup: () => () => h('div', { 'data-testid': 'other-page' }, 'other'),
    });

    const RouterHost = defineComponent({
      name: 'RouterHost',
      setup() {
        return () =>
          h(resolveComponent('RouterView'), null, {
            default: ({ Component }: { Component: unknown }) => h(KeepAlive, null, () => [h(Component as never)]),
          });
      },
    });

    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/audit/logs', name: 'AuditLogList', component: AuditLogsPage },
        { path: '/users', name: 'UsersIndex', component: OtherPage },
      ],
    });

    await router.push({
      path: '/audit/logs',
      query: initialQuery,
    });
    await router.isReady();

    const replaceSpy = vi.spyOn(router, 'replace');
    const wrapper = mount(RouterHost, {
      global: {
        plugins: [i18n, router],
        stubs: {
          'management-empty-state': passthroughStub,
          'management-page-content': passthroughStub,
          'management-page-header': passthroughStub,
          't-button': buttonStub,
          't-checkbox': checkboxStub,
          't-checkbox-group': checkboxGroupStub,
          't-drawer': drawerStub,
          't-space': passthroughStub,
          't-tag': tagStub,
        },
      },
    });

    await flushPromises();
    return { router, replaceSpy, wrapper };
  }

  it('restores deep-link filters including created range and keeps backend request shape unchanged', async () => {
    const expectedCreatedRange = normalizeRouteRangeForPageState(['2026-05-01T10:00:00Z', '2026-05-02T18:30:00Z']);
    const { wrapper } = await mountPage({
      actor: 'alice',
      created_from: '2026-05-01T10:00:00Z',
      created_to: '2026-05-02T18:30:00Z',
      result: 'FAILED',
    });

    expect(wrapper.get('[data-testid="audit-filter-model"]').text()).toContain('"actor":"alice"');
    expect(JSON.parse(wrapper.get('[data-testid="audit-filter-model"]').text()).createdRange).toEqual(
      expectedCreatedRange,
    );
    expect(getAuditLogsMock).toHaveBeenLastCalledWith({
      page: 1,
      page_size: 10,
      actor: 'alice',
      result: 'FAILED',
      created_from: '2026-05-01T10:00:00.000Z',
      created_to: '2026-05-02T18:30:00.000Z',
      sort: ['created_at:desc'],
    });
  });

  it('loads explicit-range records and opens the detail drawer', async () => {
    const { wrapper } = await mountPage();

    expect(getAuditLogsMock).toHaveBeenCalledWith(
      expect.objectContaining({
        created_from: '2026-05-30T07:21:04.000Z',
        created_to: '2026-05-31T07:21:04.000Z',
        results: ['DENIED'],
        sort: ['created_at:desc'],
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
      created_from: '2026-05-30T07:21:04.000Z',
      created_to: '2026-05-31T07:21:04.000Z',
      results: 'DENIED',
      monitorView: 'overview',
      monitorTrendRange: '10m',
      monitorAnomalyKey: 'resource_cpu_pressure',
      monitorScopeRef: 'runtime:cpu',
    });

    getAuditLogsMock.mockClear();
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
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-05-31T07:21:04Z'));
    const { wrapper } = await mountPage();
    getAuditLogsMock.mockClear();

    await wrapper.get('[data-testid="audit-preset"]').trigger('click');
    await flushPromises();

    expect(getAuditLogsMock).toHaveBeenCalledWith({
      page: 1,
      page_size: 10,
      business_category: 'high_risk_operations',
      created_from: '2026-05-30T07:21:04.000Z',
      created_to: '2026-05-31T07:21:04.000Z',
      preset: 'last_24h',
      risk_levels: ['HIGH', 'CRITICAL'],
      sort: ['created_at:desc'],
    });
  });

  it('requests scope in business-drilldown mode and renders readonly projection metadata', async () => {
    getAuditLogsMock.mockResolvedValueOnce(
      createAuditLogsResponse({
        items: [],
        total: 15,
        applied_scope: {
          module: 'audit',
          scope: 'sensitive_operations',
          name: 'Sensitive Operations',
          description: 'Sensitive write actions',
          owned_fields: ['business_category'],
        },
        scope_projection: {
          title: 'Sensitive Operations',
          items: [
            {
              key: 'business_category',
              label_key: 'audit.logList.builder.fields.businessCategory',
              kind: 'enum',
              values: ['sensitive_operations'],
              locked: true,
            },
          ],
        },
        convertible_filters: {
          preset: 'last_24h',
          business_category: 'sensitive_operations',
        },
      }),
    );

    const { wrapper } = await mountPage({
      preset: 'last_24h',
      scope: 'sensitive_operations',
    });

    expect(getAuditLogsMock).toHaveBeenLastCalledWith({
      page: 1,
      page_size: 10,
      preset: 'last_24h',
      scope: 'sensitive_operations',
      sort: ['created_at:desc'],
    });
    expect(wrapper.text()).toContain('Drilldown: Sensitive Operations');
    expect(wrapper.text()).not.toContain('Condition:');
    expect(wrapper.text()).not.toContain('sensitive_operations');
  });

  it('exits business drilldown by removing scope only', async () => {
    getAuditLogsMock.mockResolvedValue(
      createAuditLogsResponse({
        items: [],
        total: 15,
        applied_scope: {
          module: 'audit',
          scope: 'sensitive_operations',
          name: 'Sensitive Operations',
          owned_fields: ['business_category'],
        },
        scope_projection: {
          title: 'Sensitive Operations',
          items: [],
        },
        convertible_filters: {
          preset: 'last_24h',
          business_category: 'sensitive_operations',
        },
      }),
    );

    const { router, wrapper } = await mountPage({
      preset: 'last_24h',
      scope: 'sensitive_operations',
      actor: 'admin',
    });

    await wrapper.get('button').trigger('click');
    const exitButton = wrapper.findAll('button').find((item) => item.text().includes('Exit drilldown'));
    expect(exitButton).toBeTruthy();
    await exitButton!.trigger('click');
    await flushPromises();

    expect(router.currentRoute.value.query).toMatchObject({
      preset: 'last_24h',
      actor: 'admin',
    });
    expect(router.currentRoute.value.query).not.toHaveProperty('scope');
  });

  it('converts scope to normal filters by removing scope and writing canonical filters to route', async () => {
    getAuditLogsMock.mockResolvedValue(
      createAuditLogsResponse({
        items: [],
        total: 15,
        applied_scope: {
          module: 'audit',
          scope: 'sensitive_operations',
          name: 'Sensitive Operations',
          owned_fields: ['action_keywords'],
        },
        scope_projection: {
          title: 'Sensitive Operations',
          items: [],
        },
        convertible_filters: {
          preset: 'last_24h',
          business_category: 'sensitive_operations',
        },
      }),
    );

    const { router, wrapper } = await mountPage({
      preset: 'last_24h',
      scope: 'sensitive_operations',
    });

    const convertButton = wrapper.findAll('button').find((item) => item.text().includes('Convert to normal filters'));
    expect(convertButton).toBeTruthy();
    await convertButton!.trigger('click');
    await flushPromises();

    expect(router.currentRoute.value.query).toMatchObject({
      preset: 'last_24h',
      business_category: 'sensitive_operations',
    });
    expect(router.currentRoute.value.query).not.toHaveProperty('scope');
  });

  it('maps the sensitive quick preset to normal filters instead of drilldown scope', async () => {
    const { router, wrapper } = await mountPage();
    getAuditLogsMock.mockClear();

    await wrapper.get('[data-testid="audit-sensitive-preset"]').trigger('click');
    await flushPromises();

    expect(router.currentRoute.value.query).toMatchObject({
      preset: 'last_24h',
      business_category: 'sensitive_operations',
    });
    expect(router.currentRoute.value.query).not.toHaveProperty('scope');
    expect(router.currentRoute.value.query).not.toHaveProperty('action_keywords');
  });

  it('maps the security-event quick preset to source and result filters', async () => {
    const { router, wrapper } = await mountPage();
    getAuditLogsMock.mockClear();

    await wrapper.get('[data-testid="audit-security-preset"]').trigger('click');
    await flushPromises();

    expect(router.currentRoute.value.query).toMatchObject({
      preset: 'last_24h',
      source: 'SECURITY_EVENT',
      results: 'DENIED,FAILED,ERROR',
    });
    expect(router.currentRoute.value.query).not.toHaveProperty('scope');
    expect(getAuditLogsMock).toHaveBeenLastCalledWith(
      expect.objectContaining({
        preset: 'last_24h',
        source: 'SECURITY_EVENT',
        results: ['DENIED', 'FAILED', 'ERROR'],
      }),
    );
  });

  it('keeps single-condition drilldown compact without collapse scaffolding', async () => {
    getAuditLogsMock.mockResolvedValueOnce(
      createAuditLogsResponse({
        items: [],
        total: 15,
        applied_scope: {
          module: 'audit',
          scope: 'sensitive_operations',
          name: 'Sensitive Operations',
          owned_fields: ['business_category'],
        },
        scope_projection: {
          title: 'Sensitive Operations',
          items: [
            {
              key: 'business_category',
              label_key: 'audit.logList.builder.fields.businessCategory',
              kind: 'enum',
              values: ['sensitive_operations'],
              locked: true,
            },
          ],
        },
        convertible_filters: {
          preset: 'last_24h',
          business_category: 'sensitive_operations',
        },
      }),
    );

    const { wrapper } = await mountPage({
      preset: 'last_24h',
      scope: 'sensitive_operations',
    });

    expect(wrapper.text()).not.toContain('Scope conditions');
    expect(wrapper.text()).not.toContain('Collapse conditions');
    expect(wrapper.text()).not.toContain('Show all conditions');
  });

  it('does not send an implicit preset when the route has no time range', async () => {
    const { wrapper } = await mountPage({});

    expect(getAuditLogsMock).toHaveBeenLastCalledWith({
      page: 1,
      page_size: 10,
      sort: ['created_at:desc'],
    });
    expect(wrapper.text()).toContain('req-1');
  });

  it('syncs interactive filters into route query for reload and sharing', async () => {
    const expectedCreatedFrom = localDateTimeToUtcIso('2026-05-01 10:00:00');
    const expectedCreatedTo = localDateTimeToUtcIso('2026-05-02 18:30:00');
    const { replaceSpy, router, wrapper } = await mountPage();
    getAuditLogsMock.mockClear();
    replaceSpy.mockClear();

    await wrapper.get('[data-testid="audit-route-sync"]').trigger('click');
    await wrapper.get('[data-testid="audit-search"]').trigger('click');
    await flushPromises();

    expect(replaceSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        path: '/audit/logs',
        query: expect.objectContaining({
          actor: 'route-admin',
          created_from: expectedCreatedFrom,
          created_to: expectedCreatedTo,
          result: 'FAILED',
          sort: ['created_at:asc'],
        }),
      }),
    );
    expect(router.currentRoute.value.query).toMatchObject({
      actor: 'route-admin',
      created_from: expectedCreatedFrom,
      created_to: expectedCreatedTo,
      result: 'FAILED',
      sort: ['created_at:asc'],
    });
    expect(getAuditLogsMock).toHaveBeenLastCalledWith(
      expect.objectContaining({
        result: 'FAILED',
        created_from: expectedCreatedFrom,
        created_to: expectedCreatedTo,
        sort: ['created_at:asc'],
      }),
    );
  });

  it('preserves explicit created range over preset-derived display state', async () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-05-31T07:21:04Z'));

    const { wrapper } = await mountPage({
      created_from: '2026-05-01T10:00:00Z',
      created_to: '2026-05-02T18:30:00Z',
    });

    expect(JSON.parse(wrapper.get('[data-testid="audit-filter-model"]').text()).createdRange).toEqual(
      normalizeRouteRangeForPageState(['2026-05-01T10:00:00Z', '2026-05-02T18:30:00Z']),
    );
    expect(JSON.parse(wrapper.get('[data-testid="audit-filter-model"]').text()).createdRange).not.toEqual(
      normalizeRouteRangeForPageState(['2026-05-30T07:21:04.000Z', '2026-05-31T07:21:04.000Z']),
    );
  });

  it('ignores legacy route params and keeps only canonical visible filters', async () => {
    const { router, wrapper } = await mountPage({
      preset: 'last_24h',
      summary: 'failed-operations',
      risk_group: 'auth_failures',
      occurred_from: '2026-05-01T10:00:00Z',
      occurred_to: '2026-05-02T18:30:00Z',
      created_from: '2026-05-03T10:00:00Z',
      created_to: '2026-05-04T18:30:00Z',
      results: 'DENIED',
    });

    expect(JSON.parse(wrapper.get('[data-testid="audit-filter-model"]').text()).createdRange).toEqual(
      normalizeRouteRangeForPageState(['2026-05-03T10:00:00Z', '2026-05-04T18:30:00Z']),
    );
    expect(wrapper.get('[data-testid="audit-filter-model"]').text()).not.toContain('"success":"false"');
    expect(wrapper.get('[data-testid="audit-filter-model"]').text()).not.toContain(
      '"resourceTypes":["auth","session"]',
    );
    expect(router.currentRoute.value.query).toMatchObject({
      preset: 'last_24h',
      created_from: '2026-05-03T10:00:00.000Z',
      created_to: '2026-05-04T18:30:00.000Z',
      results: 'DENIED',
    });
    expect(router.currentRoute.value.query).not.toHaveProperty('summary');
    expect(router.currentRoute.value.query).not.toHaveProperty('risk_group');
    expect(router.currentRoute.value.query).not.toHaveProperty('occurred_from');
    expect(router.currentRoute.value.query).not.toHaveProperty('occurred_to');
    expect(getAuditLogsMock).toHaveBeenLastCalledWith({
      page: 1,
      page_size: 10,
      preset: 'last_24h',
      created_from: '2026-05-03T10:00:00.000Z',
      created_to: '2026-05-04T18:30:00.000Z',
      results: ['DENIED'],
      sort: ['created_at:desc'],
    });
  });

  it('writes back canonical query fields only after interactive changes', async () => {
    const { router, wrapper } = await mountPage({
      preset: 'last_24h',
      summary: 'failed-operations',
      risk_group: 'auth_failures',
      occurred_from: '2026-05-01T10:00:00Z',
      occurred_to: '2026-05-02T18:30:00Z',
      created_from: '2026-05-03T10:00:00Z',
      created_to: '2026-05-04T18:30:00Z',
    });

    await wrapper.get('[data-testid="audit-search"]').trigger('click');
    await flushPromises();

    expect(router.currentRoute.value.query).toMatchObject({
      preset: 'last_24h',
      created_from: '2026-05-03T10:00:00.000Z',
      created_to: '2026-05-04T18:30:00.000Z',
      sort: ['created_at:desc'],
    });
    expect(router.currentRoute.value.query).not.toHaveProperty('summary');
    expect(router.currentRoute.value.query).not.toHaveProperty('risk_group');
    expect(router.currentRoute.value.query).not.toHaveProperty('occurred_from');
    expect(router.currentRoute.value.query).not.toHaveProperty('occurred_to');
  });

  it('does not redirect back to audit logs after the kept-alive page is deactivated', async () => {
    const { replaceSpy, router, wrapper } = await mountKeepAliveHost({
      created_from: '2026-05-30T07:21:04.000Z',
      created_to: '2026-05-31T07:21:04.000Z',
      results: 'DENIED',
    });

    getAuditLogsMock.mockClear();
    replaceSpy.mockClear();

    await router.push({ path: '/users', query: { tab: 'active' } });
    await flushPromises();

    expect(router.currentRoute.value.path).toBe('/users');
    expect(router.currentRoute.value.query).toMatchObject({ tab: 'active' });
    expect(wrapper.get('[data-testid="other-page"]').text()).toBe('other');
    expect(replaceSpy).not.toHaveBeenCalledWith(
      expect.objectContaining({
        path: '/audit/logs',
      }),
    );
    expect(getAuditLogsMock).not.toHaveBeenCalled();
  });

  it('re-applies current route query when the kept-alive audit page is re-activated', async () => {
    const { router, wrapper } = await mountKeepAliveHost({
      created_from: '2026-05-30T07:21:04.000Z',
      created_to: '2026-05-31T07:21:04.000Z',
      results: 'DENIED',
    });

    await router.push({ path: '/users', query: { tab: 'active' } });
    await flushPromises();

    getAuditLogsMock.mockClear();

    await router.push({
      path: '/audit/logs',
      query: {
        resource_type: 'user',
        resource_name: 'Graft Admin',
        resource_id: '1',
      },
    });
    await flushPromises();

    expect(router.currentRoute.value.path).toBe('/audit/logs');
    expect(router.currentRoute.value.query).toMatchObject({
      resource_type: 'user',
      resource_name: 'Graft Admin',
      resource_id: '1',
    });
    expect(wrapper.get('[data-testid="audit-filter-model"]').text()).toContain('"resourceType":"user"');
    expect(wrapper.get('[data-testid="audit-filter-model"]').text()).toContain('"resourceName":"Graft Admin"');
    expect(wrapper.get('[data-testid="audit-filter-model"]').text()).toContain('"resourceId":"1"');
    expect(getAuditLogsMock).toHaveBeenLastCalledWith({
      page: 1,
      page_size: 10,
      resource_type: 'user',
      resource_name: 'Graft Admin',
      resource_id: '1',
      sort: ['created_at:desc'],
    });
  });
});
