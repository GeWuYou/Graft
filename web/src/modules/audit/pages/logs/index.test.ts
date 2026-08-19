import { flushPromises, mount } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, KeepAlive, resolveComponent } from 'vue';
import { createI18n } from 'vue-i18n';
import { createMemoryHistory, createRouter } from 'vue-router';

import { localDateTimeToUtcIso, normalizeRouteRangeForPageState } from '@/shared/observability';
import { clearQueryCache } from '@/shared/query';
import { getPermissionStore } from '@/store/modules/permission';

import type { AuditLogListResponse, AuditVisibilityPolicyResponse } from '../../types/audit';
import AuditLogsPage from './index.vue';

const {
  deleteAuditSavedViewMock,
  deleteAuditVisibilityOverrideMock,
  getAuditLogDetailMock,
  getAuditLogsMock,
  getAuditSavedViewsMock,
  getAuditVisibilityPolicyMock,
  postAuditSavedViewMock,
  putAuditSavedViewMock,
  updateAuditVisibilityDefaultMock,
  upsertAuditVisibilityOverrideMock,
  upsertAuditVisibilityOverridesBatchMock,
} = vi.hoisted(() => ({
  deleteAuditSavedViewMock: vi.fn(),
  deleteAuditVisibilityOverrideMock: vi.fn(),
  getAuditLogDetailMock: vi.fn(),
  getAuditLogsMock: vi.fn(),
  getAuditSavedViewsMock: vi.fn(),
  getAuditVisibilityPolicyMock: vi.fn(),
  postAuditSavedViewMock: vi.fn(),
  putAuditSavedViewMock: vi.fn(),
  updateAuditVisibilityDefaultMock: vi.fn(),
  upsertAuditVisibilityOverrideMock: vi.fn(),
  upsertAuditVisibilityOverridesBatchMock: vi.fn(),
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

function createVisibilityPolicy(overrides: Partial<AuditVisibilityPolicyResponse> = {}): AuditVisibilityPolicyResponse {
  return {
    default: {
      key: 'global',
      strategy: 'visible',
      updated_at: '2026-05-27T08:00:00Z',
    },
    overrides: [],
    catalog: [
      {
        source: 'REQUEST',
        action_key: 'POST /api/auth/login',
        display_name: 'Login',
        description: 'Login request',
        category: 'auth',
        default_strategy: 'visible',
        effective_strategy: 'visible',
        overridden: false,
      },
      {
        source: 'REQUEST',
        action_key: 'POST /api/auth/logout',
        display_name: 'Logout',
        description: 'Logout request',
        category: 'auth',
        default_strategy: 'visible',
        effective_strategy: 'visible',
        overridden: false,
      },
    ],
    ...overrides,
  };
}

vi.mock('../../api/audit', () => ({
  deleteAuditSavedView: deleteAuditSavedViewMock,
  deleteAuditVisibilityOverride: deleteAuditVisibilityOverrideMock,
  getAuditLogDetail: getAuditLogDetailMock,
  getAuditLogs: getAuditLogsMock,
  getAuditSavedViews: getAuditSavedViewsMock,
  getAuditVisibilityPolicy: getAuditVisibilityPolicyMock,
  postAuditSavedView: postAuditSavedViewMock,
  putAuditSavedView: putAuditSavedViewMock,
  updateAuditVisibilityDefault: updateAuditVisibilityDefaultMock,
  upsertAuditVisibilityOverride: upsertAuditVisibilityOverrideMock,
  upsertAuditVisibilityOverridesBatch: upsertAuditVisibilityOverridesBatchMock,
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
    props: ['presets', 'activePreset', 'modelValue', 'canManageVisibility'],
    emits: ['search', 'reset', 'apply-preset', 'update:modelValue'],
    setup(props, { emit, slots }) {
      return () =>
        h('div', [
          h('span', { 'data-testid': 'audit-filter-model' }, JSON.stringify(props.modelValue)),
          h('span', { 'data-testid': 'audit-filter-can-manage-visibility' }, String(props.canManageVisibility)),
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
          slots['saved-query-views']?.(),
        ]);
    },
  }),
}));

vi.mock('../../components/AuditTable.vue', () => ({
  default: defineComponent({
    name: 'AuditTableStub',
    props: ['rows', 'summary', 'footerSummary', 'selectedRowKeys'],
    emits: [
      'detail',
      'select-change',
      'update:current',
      'update:pageSize',
      'page-change',
      'view-access-log',
      'view-app-log',
      'view-security-event',
    ],
    setup(props, { emit, slots }) {
      return () =>
        h('div', [
          props.summary,
          props.footerSummary,
          h('span', JSON.stringify(props.rows)),
          h('span', { 'data-testid': 'audit-selected-row-keys' }, JSON.stringify(props.selectedRowKeys ?? [])),
          h(
            'button',
            {
              'data-testid': 'audit-select-current-row',
              onClick: () => emit('select-change', [props.rows?.[0]?.id, props.rows?.[0]?.id, 99]),
            },
            'select-current-row',
          ),
          h('button', { 'data-testid': 'audit-page-change', onClick: () => emit('page-change') }, 'page-change'),
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
          slots.batch?.(),
        ]);
    },
  }),
}));

const managementBatchBarStub = defineComponent({
  name: 'ManagementBatchBarStub',
  emits: ['select-current-page'],
  setup(_, { emit }) {
    return () =>
      h(
        'button',
        { 'data-testid': 'audit-select-current-page', onClick: () => emit('select-current-page') },
        'select-current-page',
      );
  },
});

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
  props: ['disabled', 'loading'],
  emits: ['click'],
  setup(props, { emit, slots, attrs }) {
    return () =>
      h(
        'button',
        {
          ...attrs,
          disabled: props.disabled || props.loading,
          'data-loading': String(Boolean(props.loading)),
          onClick: () => emit('click'),
        },
        slots.default?.(),
      );
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
  props: ['visible', 'header', 'footer'],
  setup(props, { slots }) {
    return () => h('div', { 'data-testid': 'policy-drawer', 'data-footer': String(props.footer) }, slots.default?.());
  },
});

const dialogStub = defineComponent({
  name: 'TDialogStub',
  props: ['visible', 'header', 'body', 'confirmLoading'],
  emits: ['update:visible', 'cancel', 'close', 'confirm'],
  setup(props, { emit }) {
    return () =>
      h('div', { 'data-testid': 'ignore-default-dialog', 'data-visible': String(props.visible) }, [
        h('span', props.header),
        h('span', props.body),
        h('button', { 'data-testid': 'ignore-default-cancel', onClick: () => emit('cancel') }, 'cancel'),
        h(
          'button',
          {
            'data-testid': 'ignore-default-confirm',
            disabled: props.confirmLoading,
            onClick: () => emit('confirm'),
          },
          'confirm',
        ),
      ]);
  },
});

const tagStub = defineComponent({
  name: 'TTagStub',
  setup(_, { slots }) {
    return () => h('span', slots.default?.());
  },
});

const savedQueryViewControlStub = defineComponent({
  name: 'SavedQueryViewControlStub',
  props: ['controller'],
  setup(props) {
    return () =>
      h('div', [
        h('span', { 'data-testid': 'audit-saved-view-selected' }, String(props.controller?.selectedId.value ?? '')),
        h(
          'button',
          {
            'data-testid': 'audit-saved-view-apply',
            onClick: () => void props.controller?.select(7),
          },
          'apply-saved-view',
        ),
      ]);
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
  'audit.logList.batch.actions': 'Batch actions',
  'audit.logList.batch.clear': 'Clear selection',
  'audit.logList.batch.invertCurrentPage': 'Invert current page',
  'audit.logList.batch.selectCurrentPage': 'Select current page',
  'audit.logList.batch.selected': '{count} selected',
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
  'audit.logList.policy.manage': 'Manage Recording Policy',
  'audit.logList.policy.drawerTitle': 'Audit Recording Policy',
  'audit.logList.policy.defaultStrategy': 'Default Policy for New Events',
  'audit.logList.policy.defaultHint': 'Future events only',
  'audit.logList.policy.saveDefault': 'Save',
  'audit.logList.policy.saveAllOverrides': 'Save All',
  'audit.logList.policy.saveSuccess': 'Audit visibility default updated',
  'audit.logList.policy.saveFailed': 'Failed to update audit visibility default',
  'audit.logList.policy.overrideTitle': 'Per-event overrides',
  'audit.logList.policy.overrideHint': 'Override hint',
  'audit.logList.policy.saveOverride': 'Save',
  'audit.logList.policy.saveOverrideSuccess': 'Audit visibility override updated',
  'audit.logList.policy.saveOverrideFailed': 'Failed to update audit visibility override',
  'audit.logList.policy.saveAllSuccess': 'All overrides updated',
  'audit.logList.policy.saveAllFailed': 'Failed to update all overrides',
  'audit.logList.policy.resetOverride': 'Reset Override',
  'audit.logList.policy.resetOverrideSuccess': 'Audit visibility override removed',
  'audit.logList.policy.resetOverrideFailed': 'Failed to remove audit visibility override',
  'audit.logList.policy.overriddenTag': 'Overridden',
  'audit.logList.policy.unsavedTag': 'Unsaved',
  'audit.logList.policy.descriptionFallback': 'No description',
  'audit.logList.policy.ignoreConfirmTitle': 'Ignore future audit events?',
  'audit.logList.policy.ignoreConfirmBody': 'Ignored events cannot be recovered.',
  'audit.logList.policy.ignoreConfirmCancel': 'Cancel',
  'audit.logList.policy.ignoreConfirmAction': 'Ignore and Drop',
  'audit.logList.policy.catalog.request.post_api_auth_refresh.displayName': 'Refresh Session Token',
  'audit.logList.policy.catalog.request.post_api_auth_refresh.description': 'Frontend-owned refresh token description',
  'audit.logList.policy.catalog.request.post_api_auth_login.displayName': 'Login',
  'audit.logList.policy.catalog.request.post_api_auth_login.description': 'Login request',
  'audit.logList.policy.catalog.request.post_api_auth_logout.displayName': 'Logout',
  'audit.logList.policy.catalog.request.post_api_auth_logout.description': 'Logout request',
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
    clearQueryCache();
    deleteAuditSavedViewMock.mockReset();
    deleteAuditVisibilityOverrideMock.mockReset();
    getAuditLogsMock.mockReset();
    getAuditLogDetailMock.mockReset();
    getAuditSavedViewsMock.mockReset();
    getAuditVisibilityPolicyMock.mockReset();
    updateAuditVisibilityDefaultMock.mockReset();
    upsertAuditVisibilityOverrideMock.mockReset();
    upsertAuditVisibilityOverridesBatchMock.mockReset();
    postAuditSavedViewMock.mockReset();
    putAuditSavedViewMock.mockReset();
    getAuditLogsMock.mockResolvedValue(createAuditLogsResponse());
    getAuditLogDetailMock.mockImplementation(async (id: number) => ({
      ...createAuditLogsResponse().items[0],
      id,
      metadata: {
        detail: true,
      },
    }));
    getAuditSavedViewsMock.mockResolvedValue([]);
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
    upsertAuditVisibilityOverridesBatchMock.mockResolvedValue({ items: [] });
    getPermissionStore().setBootstrapSnapshot(null);
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
    canManagePolicy = false,
  ) {
    getPermissionStore().setBootstrapSnapshot(canManagePolicy ? ({ permissions: ['audit.manage'] } as never) : null);
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/security/audit', component: AuditLogsPage },
        { path: '/observability/access-logs', component: passthroughStub },
        { path: '/observability/application-logs', component: passthroughStub },
        { path: '/observability/overview', component: passthroughStub },
      ],
    });

    await router.push({
      path: '/security/audit',
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
          ManagementBatchBar: managementBatchBarStub,
          't-button': buttonStub,
          't-checkbox': checkboxStub,
          't-checkbox-group': checkboxGroupStub,
          't-drawer': drawerStub,
          't-dialog': dialogStub,
          't-space': passthroughStub,
          't-select': selectStub,
          't-tag': tagStub,
          SavedQueryViewControl: savedQueryViewControlStub,
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
        { path: '/security/audit', name: 'AuditLogList', component: AuditLogsPage },
        { path: '/security/users', name: 'UsersIndex', component: OtherPage },
      ],
    });

    await router.push({
      path: '/security/audit',
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
          't-dialog': dialogStub,
          't-space': passthroughStub,
          't-select': selectStub,
          't-tag': tagStub,
          SavedQueryViewControl: savedQueryViewControlStub,
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

  it('restores a saved query through the route and clears its selection on reset', async () => {
    getAuditSavedViewsMock.mockResolvedValue([
      {
        id: 7,
        name: 'Denied sessions',
        page_size: 50,
        query_state: {
          visibility_scope: 'hidden_only',
          actor: 'alice',
          action_prefixes: ['rbac.'],
          created_from: '2026-05-01T10:00:00.000Z',
          created_to: '2026-05-02T18:30:00.000Z',
          results: ['DENIED', 'FAILED'],
          session_id: 'session-7',
          success: false,
          sort: ['created_at:asc'],
        },
        visible_columns: ['action', 'session_id', 'result', 'created_at'],
      },
    ]);
    const { router, wrapper } = await mountPage(undefined, true);

    await wrapper.get('[data-testid="audit-saved-view-apply"]').trigger('click');
    await flushPromises();

    expect(router.currentRoute.value.query).toMatchObject({
      actor: 'alice',
      visibility_scope: 'hidden_only',
      action_prefixes: 'rbac.',
      created_from: '2026-05-01T10:00:00.000Z',
      created_to: '2026-05-02T18:30:00.000Z',
      results: 'DENIED,FAILED',
      session: 'session-7',
      success: 'false',
      sort: ['created_at:asc'],
    });
    expect(router.currentRoute.value.query).not.toHaveProperty('scope');
    expect(getAuditLogsMock).toHaveBeenLastCalledWith({
      page: 1,
      page_size: 50,
      visibility_scope: 'hidden_only',
      actor: 'alice',
      action_prefixes: ['rbac.'],
      created_from: '2026-05-01T10:00:00.000Z',
      created_to: '2026-05-02T18:30:00.000Z',
      results: ['DENIED', 'FAILED'],
      session_id: 'session-7',
      success: false,
      sort: ['created_at:asc'],
    });
    expect(wrapper.get('[data-testid="audit-saved-view-selected"]').text()).toBe('7');

    await wrapper.get('[data-testid="audit-reset"]').trigger('click');
    await flushPromises();

    expect(wrapper.get('[data-testid="audit-saved-view-selected"]').text()).toBe('');
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

  it('keeps cross-page selection unique when selecting the next current page', async () => {
    const { wrapper } = await mountPage();

    await wrapper.get('[data-testid="audit-select-current-row"]').trigger('click');
    expect(wrapper.get('[data-testid="audit-selected-row-keys"]').text()).toBe('[1]');

    getAuditLogsMock.mockResolvedValueOnce(
      createAuditLogsResponse({
        items: [{ ...createAuditLogsResponse().items[0], id: 2 }],
        page: 2,
      }),
    );
    await wrapper.get('[data-testid="audit-page-change"]').trigger('click');
    await flushPromises();
    await wrapper.get('[data-testid="audit-select-current-page"]').trigger('click');

    expect(wrapper.get('[data-testid="audit-selected-row-keys"]').text()).toBe('[1,2]');
  });

  it('prefers frontend locale mapping over backend policy catalog display text when a source-action key exists', async () => {
    const { wrapper } = await mountPage();

    await (wrapper.vm as unknown as { loadPolicySnapshot: () => Promise<void> }).loadPolicySnapshot();
    await flushPromises();

    expect(wrapper.text()).toContain('Refresh Session Token');
    expect(wrapper.text()).toContain('Frontend-owned refresh token description');
    expect(wrapper.text()).not.toContain('Refresh token request');
  });

  it('keeps other override and default drafts dirty when a single override is saved', async () => {
    getAuditVisibilityPolicyMock.mockResolvedValue(createVisibilityPolicy());
    const { wrapper } = await mountPage({}, true);
    const page = wrapper.vm as unknown as {
      handleOverrideDraftChange: (source: string, actionKey: string, value: string) => void;
      handlePolicyDefaultChange: (value: string) => void;
      isOverrideDirty: (source: string, actionKey: string) => boolean;
      loadPolicySnapshot: () => Promise<void>;
      overrideDrafts: Record<string, Record<string, string>>;
      policyDefaultDirty: boolean;
      policyDefaultStrategy: string;
      resetPolicyOverride: (source: 'REQUEST', actionKey: string) => Promise<void>;
      savePolicyOverride: (source: 'REQUEST', actionKey: string) => Promise<void>;
    };

    await page.loadPolicySnapshot();
    page.handlePolicyDefaultChange('hidden');
    page.handleOverrideDraftChange('REQUEST', 'POST /api/auth/login', 'hidden');
    page.handleOverrideDraftChange('REQUEST', 'POST /api/auth/logout', 'ignore');
    await page.savePolicyOverride('REQUEST', 'POST /api/auth/login');

    expect(upsertAuditVisibilityOverrideMock).toHaveBeenCalledWith(
      expect.objectContaining({ action_key: 'POST /api/auth/login', strategy: 'hidden' }),
    );
    expect(page.isOverrideDirty('REQUEST', 'POST /api/auth/login')).toBe(false);
    expect(page.isOverrideDirty('REQUEST', 'POST /api/auth/logout')).toBe(true);
    expect(page.overrideDrafts.REQUEST['POST /api/auth/logout']).toBe('ignore');
    expect(page.policyDefaultStrategy).toBe('hidden');
    expect(page.policyDefaultDirty).toBe(true);

    await page.resetPolicyOverride('REQUEST', 'POST /api/auth/logout');
    expect(deleteAuditVisibilityOverrideMock).not.toHaveBeenCalled();
    expect(page.isOverrideDirty('REQUEST', 'POST /api/auth/logout')).toBe(false);
    expect(page.overrideDrafts.REQUEST['POST /api/auth/logout']).toBe('visible');
  });

  it('preserves dirty drafts while refreshing untouched policy state after the drawer is reopened', async () => {
    const initialPolicy = createVisibilityPolicy();
    const refreshedPolicy = createVisibilityPolicy({
      default: {
        key: 'global',
        strategy: 'hidden',
        updated_at: '2026-05-27T09:00:00Z',
      },
      catalog: createVisibilityPolicy().catalog.map((item) => ({
        ...item,
        default_strategy: 'hidden',
        effective_strategy: 'hidden',
      })),
    });
    getAuditVisibilityPolicyMock.mockResolvedValueOnce(initialPolicy).mockResolvedValue(refreshedPolicy);
    const { wrapper } = await mountPage({}, true);
    const page = wrapper.vm as unknown as {
      handleOverrideDraftChange: (source: string, actionKey: string, value: string) => void;
      isOverrideDirty: (source: string, actionKey: string) => boolean;
      openPolicyDrawer: () => Promise<void>;
      overrideDrafts: Record<string, Record<string, string>>;
      policyDefaultStrategy: string;
      policyDrawerVisible: boolean;
    };

    await page.openPolicyDrawer();
    page.handleOverrideDraftChange('REQUEST', 'POST /api/auth/logout', 'ignore');
    page.policyDrawerVisible = false;
    await page.openPolicyDrawer();

    expect(page.policyDrawerVisible).toBe(true);
    expect(page.isOverrideDirty('REQUEST', 'POST /api/auth/logout')).toBe(true);
    expect(page.overrideDrafts.REQUEST['POST /api/auth/logout']).toBe('ignore');
    expect(page.overrideDrafts.REQUEST['POST /api/auth/login']).toBe('hidden');
    expect(page.policyDefaultStrategy).toBe('hidden');
    expect(getAuditVisibilityPolicyMock).toHaveBeenCalledTimes(2);
  });

  it('preserves unrelated drafts when saving the default or resetting a persisted override', async () => {
    const persistedPolicy = createVisibilityPolicy({
      overrides: [
        {
          id: 1,
          source: 'REQUEST',
          action_key: 'POST /api/auth/login',
          strategy: 'hidden',
          description: 'Login request',
          created_at: '2026-05-27T08:00:00Z',
          updated_at: '2026-05-27T08:00:00Z',
        },
      ],
      catalog: createVisibilityPolicy().catalog.map((item) =>
        item.action_key === 'POST /api/auth/login' ? { ...item, effective_strategy: 'hidden', overridden: true } : item,
      ),
    });
    getAuditVisibilityPolicyMock
      .mockResolvedValueOnce(persistedPolicy)
      .mockResolvedValueOnce({
        ...persistedPolicy,
        default: { ...persistedPolicy.default, strategy: 'hidden' },
      })
      .mockResolvedValue(createVisibilityPolicy());
    const { wrapper } = await mountPage({}, true);
    const page = wrapper.vm as unknown as {
      handleOverrideDraftChange: (source: string, actionKey: string, value: string) => void;
      handlePolicyDefaultChange: (value: string) => void;
      isOverrideDirty: (source: string, actionKey: string) => boolean;
      loadPolicySnapshot: () => Promise<void>;
      overrideDrafts: Record<string, Record<string, string>>;
      requestSavePolicyDefault: () => void;
      resetPolicyOverride: (source: 'REQUEST', actionKey: string) => Promise<void>;
    };

    await page.loadPolicySnapshot();
    page.handleOverrideDraftChange('REQUEST', 'POST /api/auth/logout', 'ignore');
    page.handlePolicyDefaultChange('hidden');
    page.requestSavePolicyDefault();
    await flushPromises();

    expect(page.isOverrideDirty('REQUEST', 'POST /api/auth/logout')).toBe(true);
    expect(page.overrideDrafts.REQUEST['POST /api/auth/logout']).toBe('ignore');

    await page.resetPolicyOverride('REQUEST', 'POST /api/auth/login');
    expect(deleteAuditVisibilityOverrideMock).toHaveBeenCalledWith('REQUEST', 'POST /api/auth/login');
    expect(page.isOverrideDirty('REQUEST', 'POST /api/auth/logout')).toBe(true);
    expect(page.overrideDrafts.REQUEST['POST /api/auth/logout']).toBe('ignore');
  });

  it('saves exactly dirty overrides atomically and retains all drafts when the batch fails', async () => {
    getAuditVisibilityPolicyMock.mockResolvedValue(createVisibilityPolicy());
    const { wrapper } = await mountPage({}, true);
    const page = wrapper.vm as unknown as {
      handleOverrideDraftChange: (source: string, actionKey: string, value: string) => void;
      isOverrideDirty: (source: string, actionKey: string) => boolean;
      loadPolicySnapshot: () => Promise<void>;
      saveAllPolicyOverrides: () => Promise<void>;
    };

    await page.loadPolicySnapshot();
    const saveAllButton = () => wrapper.findAll('button').find((button) => button.text() === 'Save All');
    expect(saveAllButton()?.attributes()).toHaveProperty('disabled');
    page.handleOverrideDraftChange('REQUEST', 'POST /api/auth/login', 'hidden');
    page.handleOverrideDraftChange('REQUEST', 'POST /api/auth/logout', 'ignore');
    await flushPromises();
    expect(wrapper.text()).toContain('Unsaved');
    expect(saveAllButton()?.attributes()).not.toHaveProperty('disabled');
    upsertAuditVisibilityOverridesBatchMock.mockRejectedValueOnce(new Error('batch failed'));
    await page.saveAllPolicyOverrides();

    expect(page.isOverrideDirty('REQUEST', 'POST /api/auth/login')).toBe(true);
    expect(page.isOverrideDirty('REQUEST', 'POST /api/auth/logout')).toBe(true);

    let resolveBatch!: (value: { items: never[] }) => void;
    upsertAuditVisibilityOverridesBatchMock.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          resolveBatch = resolve;
        }),
    );
    const savePromise = page.saveAllPolicyOverrides();
    const duplicateSavePromise = page.saveAllPolicyOverrides();
    await flushPromises();
    expect(upsertAuditVisibilityOverridesBatchMock).toHaveBeenCalledTimes(2);
    expect(saveAllButton()?.attributes('data-loading')).toBe('true');
    resolveBatch({ items: [] });
    await Promise.all([savePromise, duplicateSavePromise]);
    expect(upsertAuditVisibilityOverridesBatchMock).toHaveBeenLastCalledWith({
      items: [
        {
          source: 'REQUEST',
          action_key: 'POST /api/auth/login',
          strategy: 'hidden',
          description: 'Login request',
        },
        {
          source: 'REQUEST',
          action_key: 'POST /api/auth/logout',
          strategy: 'ignore',
          description: 'Logout request',
        },
      ],
    });
    expect(page.isOverrideDirty('REQUEST', 'POST /api/auth/login')).toBe(false);
    expect(page.isOverrideDirty('REQUEST', 'POST /api/auth/logout')).toBe(false);
  });

  it('confirms the dangerous ignore default and removes the unused drawer footer', async () => {
    getAuditVisibilityPolicyMock.mockResolvedValueOnce(createVisibilityPolicy()).mockResolvedValue(
      createVisibilityPolicy({
        default: { key: 'global', strategy: 'ignore', updated_at: '2026-05-27T08:00:00Z' },
      }),
    );
    const { wrapper } = await mountPage({}, true);
    const page = wrapper.vm as unknown as {
      handlePolicyDefaultChange: (value: string) => void;
      loadPolicySnapshot: () => Promise<void>;
      requestSavePolicyDefault: () => void;
    };

    await page.loadPolicySnapshot();
    expect(wrapper.findAllComponents(selectStub)[0]?.props('options')).toContainEqual({
      label: 'Ignore and drop',
      value: 'ignore',
    });
    page.handlePolicyDefaultChange('ignore');
    page.requestSavePolicyDefault();
    await flushPromises();

    expect(wrapper.get('[data-testid="policy-drawer"]').attributes('data-footer')).toBe('false');
    expect(wrapper.get('[data-testid="ignore-default-dialog"]').attributes('data-visible')).toBe('true');
    expect(updateAuditVisibilityDefaultMock).not.toHaveBeenCalled();

    await wrapper.get('[data-testid="ignore-default-cancel"]').trigger('click');
    expect(wrapper.get('[data-testid="ignore-default-dialog"]').attributes('data-visible')).toBe('false');
    expect(updateAuditVisibilityDefaultMock).not.toHaveBeenCalled();

    page.requestSavePolicyDefault();
    await wrapper.get('[data-testid="ignore-default-confirm"]').trigger('click');
    await flushPromises();
    expect(updateAuditVisibilityDefaultMock).toHaveBeenCalledWith({ strategy: 'ignore' });
  });

  it('normalizes elevated visibility ranges for non-managers and keeps them for audit managers', async () => {
    const nonManager = await mountPage({ visibility_scope: 'hidden_only' });
    expect(nonManager.wrapper.get('[data-testid="audit-filter-can-manage-visibility"]').text()).toBe('false');
    expect(JSON.parse(nonManager.wrapper.get('[data-testid="audit-filter-model"]').text()).visibilityScope).toBe(
      'default',
    );
    expect(getAuditLogsMock).toHaveBeenLastCalledWith(expect.objectContaining({ visibility_scope: 'default' }));
    nonManager.wrapper.unmount();

    getAuditLogsMock.mockClear();
    const manager = await mountPage({ visibility_scope: 'hidden_only' }, true);
    expect(manager.wrapper.get('[data-testid="audit-filter-can-manage-visibility"]').text()).toBe('true');
    expect(JSON.parse(manager.wrapper.get('[data-testid="audit-filter-model"]').text()).visibilityScope).toBe(
      'hidden_only',
    );
    expect(getAuditLogsMock).toHaveBeenLastCalledWith(expect.objectContaining({ visibility_scope: 'hidden_only' }));
    expect(
      (
        manager.wrapper.vm as unknown as {
          currentAuditSavedViewQueryState: () => { visibility_scope?: string };
        }
      ).currentAuditSavedViewQueryState().visibility_scope,
    ).toBe('hidden_only');

    await manager.wrapper.get('[data-testid="audit-reset"]').trigger('click');
    await flushPromises();
    expect(JSON.parse(manager.wrapper.get('[data-testid="audit-filter-model"]').text()).visibilityScope).toBe(
      'default',
    );
    expect(
      (manager.wrapper.vm as unknown as { buildQuery: () => { visibility_scope?: string } }).buildQuery()
        .visibility_scope,
    ).toBe('default');
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
        path: '/security/audit',
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
        path: '/security/audit',
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

    await router.push({ path: '/security/users', query: { tab: 'active' } });
    await flushPromises();

    expect(router.currentRoute.value.path).toBe('/security/users');
    expect(router.currentRoute.value.query).toMatchObject({ tab: 'active' });
    expect(wrapper.get('[data-testid="other-page"]').text()).toBe('other');
    expect(replaceSpy).not.toHaveBeenCalledWith(
      expect.objectContaining({
        path: '/security/audit',
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

    await router.push({ path: '/security/users', query: { tab: 'active' } });
    await flushPromises();

    getAuditLogsMock.mockClear();

    await router.push({
      path: '/security/audit',
      query: {
        resource_type: 'user',
        resource_name: 'Graft',
        resource_id: '1',
      },
    });
    await flushPromises();

    expect(router.currentRoute.value.path).toBe('/security/audit');
    expect(router.currentRoute.value.query).toMatchObject({
      resource_type: 'user',
      resource_name: 'Graft',
      resource_id: '1',
    });
    expect(wrapper.get('[data-testid="audit-filter-model"]').text()).toContain('"resourceType":"user"');
    expect(wrapper.get('[data-testid="audit-filter-model"]').text()).toContain('"resourceName":"Graft"');
    expect(wrapper.get('[data-testid="audit-filter-model"]').text()).toContain('"resourceId":"1"');
    expect(getAuditLogsMock).toHaveBeenLastCalledWith({
      page: 1,
      page_size: 10,
      visibility_scope: 'default',
      resource_type: 'user',
      resource_name: 'Graft',
      resource_id: '1',
      sort: ['created_at:desc'],
    });
  });

  it('routes audit table row actions to related logs and opens fetched detail', async () => {
    const { router, wrapper } = await mountPage();

    await wrapper.get('[data-testid="audit-view-access-log"]').trigger('click');
    await flushPromises();

    expect(router.currentRoute.value.path).toBe('/observability/access-logs');
    expect(router.currentRoute.value.query).toMatchObject({ request_id: 'req-1' });

    await router.push('/security/audit');
    await flushPromises();

    await wrapper.get('[data-testid="audit-view-app-log"]').trigger('click');
    await flushPromises();

    expect(router.currentRoute.value.path).toBe('/observability/application-logs');
    expect(router.currentRoute.value.query).toMatchObject({ request_id: 'req-1' });

    await router.push('/security/audit');
    await flushPromises();

    await wrapper.get('[data-testid="audit-view-security-event"]').trigger('click');
    await flushPromises();

    expect(router.currentRoute.value.path).toBe('/security/audit');
    expect(router.currentRoute.value.query).toMatchObject({ audit_log_id: '1' });

    await wrapper.get('[data-testid="audit-detail"]').trigger('click');
    await flushPromises();

    expect(getAuditLogDetailMock).toHaveBeenCalledWith(1);
    expect(wrapper.text()).toContain('context');
    expect(wrapper.text()).toContain('req-1');
  });
});
