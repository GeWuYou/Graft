import { flushPromises, mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';
import { createI18n } from 'vue-i18n';

import AuditPage from './index.vue';

vi.mock('../api/audit', () => ({
  getAuditLogs: vi.fn(async () => ({
    items: [
      {
        id: 1,
        actor_user_id: 1,
        actor_username: 'admin',
        actor_display_name: 'Admin',
        action: 'user.create',
        resource_type: 'user',
        resource_id: '12',
        resource_name: 'Alice',
        success: true,
        request_id: 'req-1',
        ip: '127.0.0.1',
        user_agent: 'vitest',
        message: 'created user',
        metadata: { source: 'test' },
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
  },
  setup(props, { slots }) {
    return () => h('div', [props.title, props.description, slots.default?.(), slots.action?.(), slots.actions?.()]);
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
  emits: ['update:modelValue'],
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
            slots.request_id?.({ row }),
            slots.created_at?.({ row }),
            slots.context?.({ row }),
          ]),
        ),
      );
    };
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
      audit: {
        logList: {
          listTitle: 'Audit Logs',
          hint: 'Hint',
          summary: '{count} logs shown',
          tableHint: 'Table hint',
          refresh: 'Refresh',
          retry: 'Retry',
          clearFilters: 'Clear Filters',
          footerTotal: '{count} audit logs total',
          loadFailed: 'Failed to load audit logs',
          errorTitle: 'Audit logs are temporarily unavailable',
          emptyTitle: 'No audit logs',
          emptyDescription: 'No records',
          readonlyNotice: 'Read only',
          factSourceHint: 'Contract source',
          filters: {
            actionPlaceholder: 'Action',
            resourceTypePlaceholder: 'Resource type',
            resourceNamePlaceholder: 'Resource name',
            requestIdPlaceholder: 'Request ID',
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
          result: {
            success: 'Succeeded',
            failed: 'Failed',
          },
          actor: {
            anonymous: 'Anonymous',
          },
          resource: {
            unknown: 'Unknown',
          },
          context: {
            ip: 'IP',
            userAgent: 'Client',
            message: 'Message',
            metadata: 'Metadata',
            none: 'None',
          },
        },
      },
    },
  },
});

describe('AuditPage', () => {
  it('renders audit list rows from the settled API contract', async () => {
    const wrapper = mount(AuditPage, {
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
          't-button': buttonStub,
          't-date-range-picker': dateRangePickerStub,
          't-empty': passthroughStub,
          't-input': inputStub,
          't-pagination': passthroughStub,
          't-select': selectStub,
          't-table': tableStub,
          't-tag': passthroughStub,
        },
      },
    });

    await flushPromises();

    expect(wrapper.text()).toContain('Audit Logs');
    expect(wrapper.text()).toContain('user.create');
    expect(wrapper.text()).toContain('Admin');
    expect(wrapper.text()).toContain('Alice');
    expect(wrapper.text()).toContain('req-1');
  });
});
