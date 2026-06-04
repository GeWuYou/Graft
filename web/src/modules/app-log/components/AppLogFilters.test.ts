import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';
import { createI18n } from 'vue-i18n';

import type { AppLogFilterState } from '../types/app-log';

const logFilterBuilderStub = defineComponent({
  name: 'AdvancedQueryFilterBuilderStub',
  props: [
    'keywordPlaceholder',
    'tags',
    'timeFields',
    'fields',
    'sorters',
    'sortAddDisabled',
    'sortFieldOptionsByIndex',
  ],
  emits: ['close-tag'],
  setup(props, { emit }) {
    return () =>
      h('div', [
        h('span', { 'data-testid': 'keyword-placeholder' }, props.keywordPlaceholder),
        h('span', { 'data-testid': 'tags' }, JSON.stringify(props.tags)),
        h('span', { 'data-testid': 'time-fields' }, JSON.stringify(props.timeFields)),
        h('span', { 'data-testid': 'fields' }, JSON.stringify(props.fields)),
        h('span', { 'data-testid': 'sorters' }, JSON.stringify(props.sorters)),
        h('span', { 'data-testid': 'sort-add-disabled' }, String(props.sortAddDisabled)),
        h('span', { 'data-testid': 'sort-field-options' }, JSON.stringify(props.sortFieldOptionsByIndex)),
        h('button', { 'data-testid': 'close-sorter', onClick: () => emit('close-tag', 'sorter:0') }),
      ]);
  },
});

vi.mock('@/shared/components/query-list/AdvancedQueryFilterBuilder.vue', () => ({
  default: logFilterBuilderStub,
}));

const { default: AppLogFilters } = await import('./AppLogFilters.vue');

const i18n = createI18n({
  legacy: false,
  locale: 'zh-CN',
  messages: {
    'zh-CN': {
      appLog: {
        page: { searchPlaceholder: '搜索组件、操作、消息或错误' },
        actions: {
          search: '查询',
          reset: '重置',
          addFilter: '添加筛选条件',
          addSorter: '添加排序项',
          removeSorter: '移除排序项',
          moveSorterUp: '上移',
          moveSorterDown: '下移',
        },
        presets: { label: '快捷筛选' },
        filters: {
          occurredRange: '发生时间',
          severity: '级别',
          component: '组件',
          operation: '操作',
          requestId: '请求 ID',
          traceId: 'Trace ID',
          message: '消息',
          error: '错误',
          allSeverity: '全部级别',
          sortOccurredAt: '发生时间',
          sortSeverity: '级别',
          sortComponent: '组件',
          sortAsc: '升序',
          sortDesc: '降序',
        },
        sort: {
          tagPrefix: '排序',
          fieldPlaceholder: '排序字段',
          directionPlaceholder: '排序方向',
        },
        builder: {
          title: '筛选条件',
          hint: 'hint',
          groups: {
            filters: '筛选条件',
          },
          fields: {
            timeRange: '时间范围',
            sorterBuilder: '排序方式',
            severity: '级别',
            component: '组件',
            operation: '操作',
            requestId: '请求 ID',
            traceId: 'Trace ID',
            message: '消息',
            error: '错误',
          },
        },
      },
    },
  },
});

const baseFilters: AppLogFilterState = {
  keyword: '',
  occurredRange: [],
  severity: '',
  component: '',
  operation: '',
  requestId: '',
  traceId: '',
  message: '',
  error: '',
  sorters: [{ field: 'occurred_at', direction: 'desc' }],
};

describe('AppLogFilters', () => {
  it('passes app-log tags and time fields to the shared builder', () => {
    const wrapper = mount(AppLogFilters, {
      props: {
        activePreset: 'all',
        modelValue: {
          ...baseFilters,
          occurredRange: ['2026-06-04 10:00:00', '2026-06-04 11:00:00'],
          severity: 'error',
          component: 'modules.auth.route',
        },
        presets: [],
      },
      global: {
        plugins: [i18n],
      },
    });

    expect(wrapper.get('[data-testid="keyword-placeholder"]').text()).toBe('搜索组件、操作、消息或错误');

    const tags = JSON.parse(wrapper.get('[data-testid="tags"]').text());
    expect(tags.map((tag: { label: string }) => tag.label)).toContain('排序 1: 发生时间 ↓');
    expect(tags.map((tag: { label: string }) => tag.label)).toContain(
      '发生时间: 2026-06-04 10:00:00 ~ 2026-06-04 11:00:00',
    );
    expect(tags.map((tag: { label: string }) => tag.label)).toContain('级别: ERROR');
    expect(tags.map((tag: { label: string }) => tag.label)).toContain('组件: modules.auth.route');

    const timeFields = JSON.parse(wrapper.get('[data-testid="time-fields"]').text());
    expect(timeFields).toHaveLength(1);
    expect(timeFields[0].label).toBe('发生时间');
  });

  it('normalizes duplicate sorters and emits sorter removal', async () => {
    const wrapper = mount(AppLogFilters, {
      props: {
        activePreset: 'all',
        modelValue: {
          ...baseFilters,
          sorters: [
            { field: 'occurred_at', direction: 'desc' },
            { field: 'occurred_at', direction: 'asc' },
            { field: 'severity', direction: 'asc' },
            { field: 'component', direction: 'desc' },
          ],
        },
        presets: [],
      },
      global: {
        plugins: [i18n],
      },
    });

    expect(JSON.parse(wrapper.get('[data-testid="sorters"]').text())).toHaveLength(3);
    expect(wrapper.get('[data-testid="sort-add-disabled"]').text()).toBe('true');
    expect(JSON.parse(wrapper.get('[data-testid="sort-field-options"]').text())[0]).toEqual([
      { label: '发生时间', value: 'occurred_at' },
    ]);

    await wrapper.get('[data-testid="close-sorter"]').trigger('click');
    expect(wrapper.emitted('update:modelValue')?.[0]?.[0]).toMatchObject({
      sorters: [
        { field: 'severity', direction: 'asc' },
        { field: 'component', direction: 'desc' },
      ],
    });
  });
});
