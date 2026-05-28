import { flushPromises, mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';
import { createI18n } from 'vue-i18n';

import AuditLogsPage from './index.vue';

vi.mock('../../api/audit', () => ({
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
        request_id: 'req-1',
        ip: '127.0.0.1',
        user_agent: 'vitest',
        message: 'role removed',
        metadata: {
          source: 'test',
          trace_id: 'trace-1',
          session_id: 'sess-1',
          plugin: 'rbac',
          endpoint: '/api/rbac/roles/12',
          role: 'ops-admin',
          permission: 'rbac.role.delete',
        },
        created_at: '2026-05-27T08:00:00Z',
      },
    ],
    total: 1,
    page: 1,
    page_size: 10,
  })),
}));

vi.mock('@/modules/shared/localized-api-error', () => ({
  resolveLocalizedErrorMessage: () => 'load failed',
}));

vi.mock('@/utils/logger', () => ({
  createLogger: () => ({
    error: vi.fn(),
  }),
}));

const passthroughStub = defineComponent({
  name: 'PassthroughStub',
  props: {
    title: {
      type: String,
      default: '',
    },
    description: {
      type: String,
      default: '',
    },
    summary: {
      type: String,
      default: '',
    },
  },
  setup(props, { slots }) {
    return () =>
      h('div', [
        props.title,
        props.description,
        props.summary,
        slots.default?.(),
        slots.action?.(),
        slots.actions?.(),
        slots.filters?.(),
        slots.head?.(),
        slots.footer?.(),
      ]);
  },
});

const buttonStub = defineComponent({
  name: 'TButtonStub',
  emits: ['click'],
  setup(_, { emit, slots, attrs }) {
    return () => h('button', { ...attrs, onClick: (event: MouseEvent) => emit('click', event) }, slots.default?.());
  },
});

const inputStub = defineComponent({
  name: 'TInputStub',
  props: {
    modelValue: {
      type: String,
      default: '',
    },
  },
  emits: ['update:modelValue'],
  setup(props, { emit, attrs }) {
    return () =>
      h('input', {
        ...attrs,
        value: props.modelValue,
        onInput: (event: Event) => emit('update:modelValue', (event.target as HTMLInputElement).value),
      });
  },
});

const selectStub = defineComponent({
  name: 'TSelectStub',
  props: {
    modelValue: {
      type: String,
      default: '',
    },
    options: {
      type: Array,
      default: () => [],
    },
  },
  emits: ['update:modelValue'],
  setup(props, { emit, attrs }) {
    return () =>
      h(
        'select',
        {
          ...attrs,
          value: props.modelValue,
          onChange: (event: Event) => emit('update:modelValue', (event.target as HTMLSelectElement).value),
        },
        (props.options as Array<{ label: string; value: string }>).map((option) =>
          h('option', { value: option.value }, option.label),
        ),
      );
  },
});

const dateRangePickerStub = defineComponent({
  name: 'TDateRangePickerStub',
  props: {
    modelValue: {
      type: Array,
      default: () => [],
    },
  },
  setup(_, { slots }) {
    return () => h('div', { 'data-testid': 'audit-date-range-picker' }, slots.default?.());
  },
});

const tableStub = defineComponent({
  name: 'TTableStub',
  props: {
    data: {
      type: Array,
      default: () => [],
    },
  },
  setup(props, { slots }) {
    return () => {
      if (props.data.length === 0) {
        return h('div', slots.empty?.());
      }

      return h(
        'div',
        (props.data as Array<Record<string, unknown>>).map((row, index) =>
          h('div', { 'data-testid': `audit-row-${index}` }, [
            slots.action?.({ row }),
            slots.actor?.({ row }),
            slots.resource?.({ row }),
            slots.result?.({ row }),
            slots.created_at?.({ row }),
            slots.operation?.({ row }),
            slots.expandedRow?.({ row }),
          ]),
        ),
      );
    };
  },
});

const drawerStub = defineComponent({
  name: 'TDrawerStub',
  props: {
    visible: {
      type: Boolean,
      default: false,
    },
  },
  setup(props, { slots }) {
    return () => (props.visible ? h('section', { 'data-testid': 'audit-drawer' }, slots.default?.()) : null);
  },
});

const paginationStub = defineComponent({
  name: 'TPaginationStub',
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

const tableActionMenuStub = defineComponent({
  name: 'TableActionMenuStub',
  props: {
    actions: {
      type: Array,
      default: () => [],
    },
  },
  emits: ['action'],
  setup(props, { emit }) {
    return () =>
      h('div', [
        h(
          'button',
          {
            'data-testid': (props.actions[0] as { testId?: string } | undefined)?.testId ?? 'action-detail',
            onClick: () => emit('action', 'detail'),
          },
          'detail',
        ),
        h(
          'button',
          {
            'data-testid': (props.actions[1] as { testId?: string } | undefined)?.testId ?? 'action-same-request',
            onClick: () => emit('action', 'same-request'),
          },
          'same-request',
        ),
      ]);
  },
});

const spaceStub = defineComponent({
  name: 'TSpaceStub',
  setup(_, { slots }) {
    return () => h('div', slots.default?.());
  },
});

const i18n = createI18n({
  legacy: false,
  locale: 'en-US',
  messages: {
    'en-US': {
      menu: {
        audit: {
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
        logList: {
          listTitle: 'Audit Logs',
          hint: 'Hint',
          summary: '{count} logs shown',
          tableHint: 'Table hint',
          refresh: 'Refresh',
          detail: 'Details',
          more: 'More',
          detailTitle: 'Audit Investigation Detail',
          retry: 'Retry',
          clearFilters: 'Clear Filters',
          footerTotal: '{count} audit logs total',
          loadFailed: 'Failed to load audit logs',
          errorTitle: 'Audit logs are temporarily unavailable',
          emptyTitle: 'No audit logs',
          emptyDescription: 'No records',
          readonlyNotice: 'Read only',
          factSourceHint: 'Contract source',
          correlationTitle: 'Correlation Search',
          correlationSubtitle: 'Correlation subtitle',
          presets: {
            all: 'All Events',
            failedAuth: 'Failed Authentication',
            rbacChanges: 'RBAC Changes',
            permissionDenied: 'Permission Denied',
            pluginOps: 'Plugin Operations',
            sensitiveOps: 'Sensitive Ops',
          },
          filters: {
            actionPlaceholder: 'Action',
            resourceTypePlaceholder: 'Resource type',
            resourceNamePlaceholder: 'Resource name',
            requestIdPlaceholder: 'Request ID',
            actorPlaceholder: 'Actor',
            resourcePlaceholder: 'Resource',
            sessionPlaceholder: 'Session',
            successPlaceholder: 'Result',
            successAll: 'All',
            successTrue: 'Succeeded',
            successFalse: 'Failed',
            createdRangePlaceholder: 'Date range',
          },
          columns: {
            action: 'Action',
            actor: 'Actor',
            resource: 'Resource',
            result: 'Result',
            requestId: 'Request ID',
            createdAt: 'Created At',
            context: 'Context',
          },
          quickActions: {
            sameRequest: 'Same Request',
            sameActor: 'Same Actor',
          },
          density: {
            comfortable: 'Comfortable Density',
            compact: 'Compact Density',
            switchCompact: 'Switch to Compact',
            switchComfortable: 'Switch to Comfortable',
          },
          result: {
            success: 'Succeeded',
            failed: 'Failed',
          },
          risk: {
            failed: 'High Risk',
            sensitive: 'Sensitive',
            normal: 'Routine',
          },
          riskSignals: {
            authAnomaly: 'Auth anomaly',
            privilegeSensitive: 'Privilege sensitive',
            requestTraceable: 'Request traceable',
          },
          actor: {
            anonymous: 'Anonymous',
          },
          resource: {
            unknown: 'Unknown',
          },
          investigationSignals: {
            requestChain: { title: 'Traceable request coverage', description: 'desc', action: 'Pivot request chain' },
            rbacRisk: { title: 'RBAC-sensitive slice', description: 'desc', action: 'Open RBAC change preset' },
            failedFlow: { title: 'Failed operation slice', description: 'desc', action: 'Open failed-auth preset' },
          },
          expanded: {
            correlationTitle: 'Correlation snapshot',
            correlationSummary: '{actor} touched {resource} in request {requestId}.',
            tags: {
              request: 'Request',
              actor: 'Actor',
              resource: 'Resource',
              noRequest: 'No request id',
            },
          },
          timeline: {
            actorTitle: 'Actor timeline',
            requestTitle: 'Request timeline',
            resourceTitle: 'Resource timeline',
          },
          detailSections: {
            basic: 'Basic Info',
            request: 'Request Info',
            context: 'Context Snapshot',
            correlation: 'Correlation Chain',
            timeline: 'Timeline View',
            risk: 'Risk Analysis',
            metadata: 'Metadata',
          },
          detailFields: {
            requestId: 'Request ID',
            traceId: 'Trace ID',
            sessionId: 'Session ID',
            ip: 'IP',
            userAgent: 'User-Agent',
            message: 'Message',
            plugin: 'Plugin',
            endpoint: 'Endpoint',
            relatedRole: 'Related Role',
            relatedPermission: 'Related Permission',
            beforeSnapshot: 'Before Snapshot',
            afterSnapshot: 'After Snapshot',
          },
          detailHero: '{actor} acted on {resource}. Request chain: {requestId}.',
          correlationItems: {
            sameRequest: {
              title: 'Same request chain',
              description: 'Filter around request {requestId}.',
              empty: 'No request ID',
            },
            sameActor: {
              title: 'Same actor',
              description: 'Review nearby actions by {actor}.',
            },
            sameResource: {
              title: 'Same resource',
              description: 'Review nearby actions for {resource}.',
            },
          },
          copyMetadata: 'Copy Metadata',
          copyMetadataSuccess: 'Metadata copied',
          copyMetadataFailed: 'Failed to copy metadata',
        },
      },
    },
  },
});

describe('AuditLogsPage', () => {
  it('renders investigation workflow surfaces and rich detail information', async () => {
    const wrapper = mount(AuditLogsPage, {
      global: {
        plugins: [i18n],
        directives: {
          permission: {
            mounted() {},
          },
        },
        stubs: {
          'management-empty-state': passthroughStub,
          'management-page-content': passthroughStub,
          'management-page-header': passthroughStub,
          'management-table-card': passthroughStub,
          'management-table-pagination': passthroughStub,
          'management-toolbar': passthroughStub,
          'table-action-menu': tableActionMenuStub,
          't-button': buttonStub,
          't-date-range-picker': dateRangePickerStub,
          't-empty': passthroughStub,
          't-input': inputStub,
          't-drawer': drawerStub,
          't-pagination': paginationStub,
          't-select': selectStub,
          't-space': spaceStub,
          't-table': tableStub,
          't-tag': tagStub,
        },
      },
    });

    await flushPromises();

    expect(wrapper.text()).toContain('Correlation Search');
    expect(wrapper.text()).toContain('Failed Authentication');
    expect(wrapper.text()).toContain('Traceable request coverage');
    expect(wrapper.text()).toContain('RBAC-sensitive slice');
    expect(wrapper.text()).toContain('role.delete');
    expect(wrapper.text()).toContain('High Risk');
    expect(wrapper.text()).toContain('Correlation snapshot');

    await wrapper.get('[data-testid="audit-detail"]').trigger('click');
    await flushPromises();

    expect(wrapper.text()).toContain('Trace ID');
    expect(wrapper.text()).toContain('trace-1');
    expect(wrapper.text()).toContain('Context Snapshot');
    expect(wrapper.text()).toContain('Correlation Chain');
    expect(wrapper.text()).toContain('Risk Analysis');
    expect(wrapper.text()).toContain('Plugin');

    await wrapper.get('[data-testid="audit-same-request"]').trigger('click');
    await flushPromises();
    expect((wrapper.get('input[placeholder="Request ID"]').element as HTMLInputElement).value).toBe('req-1');
  });
});
