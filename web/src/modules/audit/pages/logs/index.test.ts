import { flushPromises, mount } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, KeepAlive, resolveComponent } from 'vue';
import { createI18n } from 'vue-i18n';
import { createMemoryHistory, createRouter } from 'vue-router';

import { localDateTimeToUtcIso, normalizeRouteRangeForPageState } from '@/shared/observability';

import type { AuditLogListResponse } from '../../types/audit';
import AuditLogsPage from './index.vue';

const {
  deleteAuditVisibilityOverrideMock,
  getAuditLogDetailMock,
  getAuditLogsMock,
  getAuditVisibilityPolicyMock,
  updateAuditVisibilityDefaultMock,
  upsertAuditVisibilityOverrideMock,
} = vi.hoisted(() => ({
  deleteAuditVisibilityOverrideMock: vi.fn(),
  getAuditLogDetailMock: vi.fn(),
  getAuditLogsMock: vi.fn(),
  getAuditVisibilityPolicyMock: vi.fn(),
  updateAuditVisibilityDefaultMock: vi.fn(),
  upsertAuditVisibilityOverrideMock: vi.fn(),
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
        session_id: 'sess-1',
        ip: '127.0.0.1',
        user_agent: 'vitest',
        request_method: 'POST',
        request_path: '/api/roles/12/delete',
        status_code: 403,
        message: 'role removed',
        metadata: {
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
  deleteAuditVisibilityOverride: deleteAuditVisibilityOverrideMock,
  getAuditLogDetail: getAuditLogDetailMock,
  getAuditLogs: getAuditLogsMock,
  getAuditVisibilityPolicy: getAuditVisibilityPolicyMock,
  updateAuditVisibilityDefault: updateAuditVisibilityDefaultMock,
  upsertAuditVisibilityOverride: upsertAuditVisibilityOverrideMock,
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
              'data-testid': 'audit-container-preset',
              onClick: () => emit('apply-preset', 'container-dangerous-ops'),
            },
            'container-preset',
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
    emits: [
      'detail',
      'update:current',
      'update:pageSize',
      'page-change',
      'view-access-log',
      'view-app-log',
      'view-security-event',
    ],
    setup(props, { emit }) {
      return () =>
        h('div', [
          props.summary,
          props.footerSummary,
          h('span', JSON.stringify(props.rows)),
          h('button', { 'data-testid': 'audit-detail', onClick: () => emit('detail', props.rows?.[0]) }, 'detail'),
          h(
            'button',
            { 'data-testid': 'audit-view-access-log', onClick: () => emit('view-access-log', props.rows?.[0]) },
            'access-log',
          ),
          h(
            'button',
            { 'data-testid': 'audit-view-app-log', onClick: () => emit('view-app-log', props.rows?.[0]) },
            'app-log',
          ),
          h(
            'button',
            {
              'data-testid': 'audit-view-security-event',
              onClick: () => emit('view-security-event', props.rows?.[0]),
            },
            'security-event',
          ),
        ]);
    },
  }),
}));

vi.mock('../../components/AuditDetailDrawer.vue', () => ({
  default: defineComponent({
    name: 'AuditDetailDrawerStub',
    props: ['initialTab', 'visible', 'record', 'monitorOrigin'],
    setup(props) {
      return () =>
        h('div', [
          String(props.visible),
          props.initialTab,
          props.record?.request_id,
          JSON.stringify(props.monitorOrigin),
        ]);
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

const selectStub = defineComponent({
  name: 'TSelectStub',
  props: ['modelValue', 'options'],
  emits: ['update:modelValue', 'change'],
  setup(props, { emit, slots }) {
    return () =>
      h('div', [
        h('span', { 'data-testid': 't-select-model' }, String(props.modelValue ?? '')),
        h(
          'button',
          {
            'data-testid': 't-select-next',
            onClick: () => {
              const options = Array.isArray(props.options) ? props.options : [];
              const currentIndex = options.findIndex((item) => item?.value === props.modelValue);
              const next = options[(currentIndex + 1 + options.length) % options.length];
              if (!next) {
                return;
              }
              emit('update:modelValue', next.value);
              emit('change', next.value);
            },
          },
          'next',
        ),
        slots.default?.(),
      ]);
  },
});

const auditMessages: Record<string, string> = {
  'audit.common.unknownActor': 'Anonymous',
  'audit.common.unknownResource': 'Unknown resource',
  'audit.common.source.REQUEST': 'Audit Event',
  'audit.common.source.SECURITY_EVENT': 'Security Event',
  'audit.common.source.DOMAIN_EVENT': 'Domain Audit',
  'audit.common.source.UNKNOWN': 'Unknown',
  'audit.common.result.SUCCESS': 'Success',
  'audit.common.result.FAILED': 'Business Failed',
  'audit.common.result.DENIED': 'Denied',
  'audit.common.result.ERROR': 'System Error',
  'audit.common.targetType.permission': 'Permission',
  'audit.common.targetType.role': 'Role',
  'audit.common.targetType.user': 'User',
  'audit.logList.detailTitle': 'Audit Detail',
  'audit.logList.sort.createdAt': 'Created At',
  'audit.logList.columns.action': 'Action',
  'audit.logList.columns.resource': 'Resource',
  'audit.logList.columns.correlation': 'Correlation',
  'audit.logList.columns.sessionId': 'Session ID',
  'audit.logList.columns.ip': 'IP',
  'audit.logList.columns.result': 'Result',
  'audit.logList.columns.risk': 'Risk',
  'audit.logList.presets.all': 'All',
  'audit.logList.presets.securityEvents': 'Security Events',
  'audit.logList.presets.failedOperations': 'Failed Operations',
  'audit.logList.presets.rbacChanges': 'RBAC Changes',
  'audit.logList.presets.permissionDenied': 'Permission Denials',
  'audit.logList.presets.sensitiveOps': 'Sensitive Ops',
  'audit.logList.presets.authFailed': 'Auth Failed',
  'audit.logList.presets.highRisk': 'High Risk',
  'audit.logList.presets.containerDangerousOps': 'Container Dangerous Ops',
  'audit.logList.footerTotal': 'Total {count}',
  'audit.logList.businessCategory.sensitiveOperations': 'Sensitive Operations',
  'audit.logList.builder.fields.businessCategory': 'Business Category',
  'audit.logList.scope.drilldownTag': 'Drilldown: {name}',
  'audit.logList.scope.convertAction': 'Convert to normal filters',
  'audit.logList.scope.exitAction': 'Exit drilldown',
  'audit.logList.reasonFallback': 'No additional reason',
  'audit.logList.drawer.messageFallback': 'No additional message',
  'audit.logList.drawer.sections.basic': 'Event Summary',
  'audit.logList.drawer.sections.request': 'Request Context',
  'audit.logList.drawer.sections.security': 'Security Event Context',
  'audit.logList.drawer.sections.correlation': 'Related Context',
  'audit.logList.drawer.sections.risk': 'Risk',
  'audit.logList.drawer.sections.context': 'Audit Context',
  'audit.logList.drawer.sections.metadata': 'Metadata',
  'audit.logList.drawer.sections.rawJson': 'Raw JSON',
  'audit.logList.drawer.fields.target': 'Audit Target',
  'audit.logList.drawer.fields.source': 'Source',
  'audit.logList.drawer.fields.result': 'Result',
  'audit.logList.drawer.fields.reason': 'Reason',
  'audit.logList.drawer.fields.requestId': 'Request ID',
  'audit.logList.drawer.fields.sessionId': 'Session ID',
  'audit.logList.drawer.fields.ip': 'IP',
  'audit.logList.drawer.fields.userAgent': 'User-Agent',
  'audit.logList.drawer.fields.method': 'Method',
  'audit.logList.drawer.fields.path': 'Path',
  'audit.logList.drawer.fields.status': 'Status',
  'audit.logList.drawer.fields.eventType': 'Event Type',
  'audit.logList.drawer.fields.permission': 'Permission',
  'audit.logList.drawer.fields.securityTarget': 'Security Target',
  'audit.logList.drawer.actions.copyRequestId': 'Copy',
  'audit.logList.drawer.actions.copyRequestIdSuccess': 'Copied',
  'audit.logList.drawer.actions.copyRequestIdFail': 'Copy failed',
  'audit.logList.drawer.actions.expandJson': 'Expand JSON',
  'audit.logList.drawer.actions.collapseJson': 'Collapse JSON',
  'audit.logList.drawer.actions.copyJson': 'Copy JSON',
  'audit.logList.drawer.actions.copyJsonSuccess': 'JSON copied',
  'audit.logList.drawer.actions.copyJsonFail': 'JSON copy failed',
  'audit.logList.drawer.actions.expandMetadata': 'Expand metadata',
  'audit.logList.drawer.actions.collapseMetadata': 'Collapse metadata',
  'audit.logList.drawer.actions.copyMetadata': 'Copy JSON',
  'audit.logList.drawer.actions.copyMetadataSuccess': 'Metadata copied',
  'audit.logList.drawer.actions.copyMetadataFail': 'Metadata copy failed',
  'audit.logList.drawer.actions.backToMonitor': 'Back to monitor',
  'audit.logList.drawer.actions.viewRelatedRequest': 'View Related Request',
  'audit.logList.drawer.actions.viewAccessLogRequest': 'View Access Log',
  'audit.logList.drawer.actions.openRelatedEvents': 'Open related events',
  'audit.logList.drawer.related.sameRequest': 'Same Request',
  'audit.logList.drawer.related.sameActor': 'Same Actor',
  'audit.logList.drawer.related.sameResource': 'Same Resource',
  'audit.logList.drawer.related.empty': 'Empty',
  'audit.logList.drawer.risk.failedOperation': 'Failed operation',
  'audit.logList.drawer.risk.sensitiveOperation': 'Sensitive write',
  'audit.logList.drawer.risk.requestTrace': 'Request trace',
  'audit.logList.drawer.risk.securityEvent': 'Security Event',
  'audit.logList.drawer.contextEmpty': 'No context',
  'audit.logList.drawer.metadataEmpty': 'No metadata',
  'audit.logList.drawer.rawJsonEmpty': 'No raw JSON',
  'audit.logList.columns.actor': 'Actor',
  'audit.logList.columns.createdAt': 'Created At',
  'audit.logList.title': 'Audit Logs',
  'audit.logList.description': 'Review audit logs',
  'audit.logList.errorTitle': 'Audit Logs',
  'audit.logList.refresh': 'Refresh',
  'audit.logList.retry': 'Retry',
  'audit.logList.actions.backToMonitor': 'Back to monitor',
  'audit.logList.columnSettings': 'Columns',
  'audit.logList.columnViews.label': 'View Presets',
  'audit.logList.columnViews.resetDefault': 'Restore Default Columns',
  'audit.logList.columnViews.default': 'Default View',
  'audit.logList.columnViews.troubleshooting': 'Troubleshooting View',
  'audit.logList.columnViews.technical': 'Technical View',
  'audit.logList.policy.manage': 'Manage Visibility',
  'audit.logList.policy.drawerTitle': 'Audit Visibility Policy',
  'audit.logList.policy.defaultStrategy': 'Global default strategy',
  'audit.logList.policy.visibilityScope': 'Current list visibility scope',
  'audit.logList.policy.saveDefault': 'Save Default',
  'audit.logList.policy.saveSuccess': 'Audit visibility default updated',
  'audit.logList.policy.saveFailed': 'Failed to update audit visibility default',
  'audit.logList.policy.overrideTitle': 'Per-event overrides',
  'audit.logList.policy.overrideHint': 'Override hint',
  'audit.logList.policy.saveOverride': 'Save Override',
  'audit.logList.policy.saveOverrideSuccess': 'Audit visibility override updated',
  'audit.logList.policy.saveOverrideFailed': 'Failed to update audit visibility override',
  'audit.logList.policy.resetOverride': 'Reset Override',
  'audit.logList.policy.resetOverrideSuccess': 'Audit visibility override removed',
  'audit.logList.policy.resetOverrideFailed': 'Failed to remove audit visibility override',
  'audit.logList.policy.overriddenTag': 'Overridden',
  'audit.logList.policy.descriptionFallback': 'No description',
  'audit.logList.policy.defaultState': 'Default: {value}',
  'audit.logList.policy.effectiveState': 'Effective: {value}',
  'audit.logList.policy.scope.default': 'Default visible only',
  'audit.logList.policy.scope.all': 'Show all persisted',
  'audit.logList.policy.scope.hiddenOnly': 'Hidden only',
  'audit.logList.policy.strategy.visible': 'Visible',
  'audit.logList.policy.strategy.hidden': 'Hidden',
  'audit.logList.policy.strategy.ignore': 'Ignore and drop',
  'menu.audit.title': 'Security Audit',
};

const i18n = createI18n({
  legacy: false,
  locale: 'en-US',
  messages: {
    'en-US': auditMessages,
  },
});

describe('AuditLogsPage', () => {
  beforeEach(() => {
    deleteAuditVisibilityOverrideMock.mockReset();
    getAuditLogsMock.mockReset();
    getAuditLogDetailMock.mockReset();
    getAuditVisibilityPolicyMock.mockReset();
    updateAuditVisibilityDefaultMock.mockReset();
    upsertAuditVisibilityOverrideMock.mockReset();
    getAuditLogsMock.mockResolvedValue(createAuditLogsResponse());
    getAuditLogDetailMock.mockImplementation(async (id: number) => ({
      ...createAuditLogsResponse().items[0],
      id,
      metadata: {
        detail: true,
      },
    }));
    getAuditVisibilityPolicyMock.mockResolvedValue({
      default: {
        key: 'global',
        strategy: 'visible',
        updated_at: '2026-05-27T08:00:00Z',
      },
      overrides: [
        {
          id: 1,
          source: 'REQUEST',
          action_key: 'POST /api/auth/refresh',
          strategy: 'hidden',
          description: 'Refresh token request',
          created_at: '2026-05-27T08:00:00Z',
          updated_at: '2026-05-27T08:00:00Z',
        },
      ],
      catalog: [
        {
          source: 'REQUEST',
          action_key: 'POST /api/auth/refresh',
          display_name: 'Refresh token',
          description: 'Refresh token request',
          category: 'auth',
          default_strategy: 'visible',
          effective_strategy: 'hidden',
          overridden: true,
        },
      ],
    });
    updateAuditVisibilityDefaultMock.mockResolvedValue({
      key: 'global',
      strategy: 'hidden',
      updated_at: '2026-05-27T08:00:00Z',
    });
    upsertAuditVisibilityOverrideMock.mockResolvedValue({
      id: 1,
      source: 'REQUEST',
      action_key: 'POST /api/auth/refresh',
      strategy: 'ignore',
      description: 'Refresh token request',
      created_at: '2026-05-27T08:00:00Z',
      updated_at: '2026-05-27T08:00:00Z',
    });
    deleteAuditVisibilityOverrideMock.mockResolvedValue({});
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
      routes: [
        { path: '/audit/logs', component: AuditLogsPage },
        { path: '/logs/access', component: passthroughStub },
        { path: '/logs/app', component: passthroughStub },
        { path: '/audit/overview', component: passthroughStub },
      ],
    });

    await router.push({
      path: '/audit/logs',
      query: initialQuery,
    });
    await router.isReady();

    const replaceSpy = vi.spyOn(router, 'replace');
    const wrapper = mount(AuditLogsPage, {
      global: {
        plugins: [router, i18n],
        stubs: {
          'management-empty-state': passthroughStub,
          'management-page-content': passthroughStub,
          'management-page-header': passthroughStub,
          't-button': buttonStub,
          't-checkbox': checkboxStub,
          't-checkbox-group': checkboxGroupStub,
          't-drawer': drawerStub,
          't-space': passthroughStub,
          't-select': selectStub,
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
        plugins: [router, i18n],
        stubs: {
          'management-empty-state': passthroughStub,
          'management-page-content': passthroughStub,
          'management-page-header': passthroughStub,
          't-button': buttonStub,
          't-checkbox': checkboxStub,
          't-checkbox-group': checkboxGroupStub,
          't-drawer': drawerStub,
          't-space': passthroughStub,
          't-select': selectStub,
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
      visibility_scope: 'default',
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
    expect(wrapper.text()).toContain('false');

    await wrapper.get('[data-testid="audit-detail"]').trigger('click');
    await flushPromises();
    expect(wrapper.text()).toContain('true');
    expect(wrapper.text()).toContain('req-1');
  });

  it('opens a detail drawer directly when audit_log_id is present in the route query', async () => {
    getAuditLogsMock.mockResolvedValueOnce(createAuditLogsResponse());
    getAuditLogDetailMock.mockResolvedValueOnce({
      ...createAuditLogsResponse().items[0],
      id: 1,
    });

    const { wrapper } = await mountPage({
      audit_log_id: '1',
    });

    await flushPromises();
    expect(getAuditLogDetailMock).toHaveBeenCalledWith(1);
    expect(wrapper.text()).toContain('1');
  });

  it('opens a detail drawer from audit_log_id even when the current page rows do not include that record', async () => {
    getAuditLogsMock.mockResolvedValueOnce(
      createAuditLogsResponse({
        items: [
          {
            ...createAuditLogsResponse().items[0],
            id: 99,
          },
        ],
      }),
    );
    getAuditLogDetailMock.mockResolvedValueOnce({
      ...createAuditLogsResponse().items[0],
      id: 1,
      request_id: 'req-deeplink',
    });

    const { wrapper } = await mountPage({
      audit_log_id: '1',
    });

    await flushPromises();
    expect(getAuditLogDetailMock).toHaveBeenCalledWith(1);
    expect(wrapper.text()).toContain('req-deeplink');
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
      visibility_scope: 'default',
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
      visibility_scope: 'default',
      preset: 'last_24h',
      scope: 'sensitive_operations',
      sort: ['created_at:desc'],
    });
    expect(wrapper.text()).toContain('Sensitive Operations');
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

  it('maps the container dangerous-op quick preset to canonical container action filters', async () => {
    const { router, wrapper } = await mountPage();
    getAuditLogsMock.mockClear();

    await wrapper.get('[data-testid="audit-container-preset"]').trigger('click');
    await flushPromises();

    expect(router.currentRoute.value.query).toMatchObject({
      preset: 'last_24h',
      action_prefix: 'ops.container.action.',
      business_category: 'high_risk_operations',
      resource_types: 'container,container_batch',
      risk_levels: 'HIGH',
    });
    expect(router.currentRoute.value.query).not.toHaveProperty('scope');
    expect(getAuditLogsMock).toHaveBeenLastCalledWith(
      expect.objectContaining({
        preset: 'last_24h',
        action_prefix: 'ops.container.action.',
        business_category: 'high_risk_operations',
        resource_types: ['container', 'container_batch'],
        risk_levels: ['HIGH'],
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
      visibility_scope: 'default',
      sort: ['created_at:desc'],
    });
    expect(wrapper.text()).toContain('false');
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
      visibility_scope: 'default',
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
      visibility_scope: 'default',
      resource_type: 'user',
      resource_name: 'Graft Admin',
      resource_id: '1',
      sort: ['created_at:desc'],
    });
  });

  it('routes audit table row actions to related logs and opens fetched detail', async () => {
    const { router, wrapper } = await mountPage();

    await wrapper.get('[data-testid="audit-view-access-log"]').trigger('click');
    await flushPromises();

    expect(router.currentRoute.value.path).toBe('/logs/access');
    expect(router.currentRoute.value.query).toMatchObject({ request_id: 'req-1' });

    await router.push('/audit/logs');
    await flushPromises();

    await wrapper.get('[data-testid="audit-view-app-log"]').trigger('click');
    await flushPromises();

    expect(router.currentRoute.value.path).toBe('/logs/app');
    expect(router.currentRoute.value.query).toMatchObject({ request_id: 'req-1' });

    await router.push('/audit/logs');
    await flushPromises();

    await wrapper.get('[data-testid="audit-view-security-event"]').trigger('click');
    await flushPromises();

    expect(router.currentRoute.value.path).toBe('/audit/logs');
    expect(router.currentRoute.value.query).toMatchObject({ audit_log_id: '1' });

    await wrapper.get('[data-testid="audit-detail"]').trigger('click');
    await flushPromises();

    expect(getAuditLogDetailMock).toHaveBeenCalledWith(1);
    expect(wrapper.text()).toContain('context');
    expect(wrapper.text()).toContain('req-1');
  });
});
