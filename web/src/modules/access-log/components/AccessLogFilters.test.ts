import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import { defineComponent, h } from 'vue';
import { createI18n } from 'vue-i18n';

import AccessLogFilters from './AccessLogFilters.vue';

const buttonStub = defineComponent({
  name: 'TButtonStub',
  emits: ['click'],
  setup(_, { attrs, emit, slots }) {
    return () => h('button', { ...attrs, onClick: () => emit('click') }, slots.default?.());
  },
});

const tagStub = defineComponent({
  name: 'TTagStub',
  emits: ['close'],
  setup(_, { slots }) {
    return () => h('div', [h('span', slots.default?.())]);
  },
});

const passthroughStub = defineComponent({
  name: 'PassthroughStub',
  setup(_, { slots }) {
    return () => h('div', [slots.default?.(), slots.content?.()]);
  },
});

const inputStub = defineComponent({
  name: 'TInputStub',
  props: ['modelValue', 'placeholder'],
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    return () =>
      h('input', {
        value: props.modelValue,
        placeholder: props.placeholder,
        onInput: (event: Event) => emit('update:modelValue', (event.target as HTMLInputElement).value),
      });
  },
});

const dateRangeStub = defineComponent({
  name: 'TDateRangePickerStub',
  setup() {
    return () => h('div', 'date-range');
  },
});

const selectStub = defineComponent({
  name: 'TSelectStub',
  props: ['modelValue'],
  emits: ['update:modelValue'],
  setup() {
    return () => h('div', 'select');
  },
});

const i18n = createI18n({
  legacy: false,
  locale: 'zh-CN',
  messages: {
    'zh-CN': {
      accessLog: {
        page: { searchPlaceholder: '搜索请求 ID、Trace ID、路径、用户名' },
        actions: { search: '查询', reset: '重置', addFilter: '添加筛选条件' },
        presets: { label: '快捷筛选' },
        sort: {
          title: '排序',
          tagPrefix: '排序',
          fieldPlaceholder: '排序字段',
          directionPlaceholder: '排序方向',
        },
        filters: {
          occurredRange: '时间范围',
          requestId: '请求 ID',
          traceId: 'Trace ID',
          userId: '用户 ID',
          username: '用户名',
          method: '方法',
          path: '路径',
          statusCode: '状态码',
          durationMin: '最小时长',
          durationMax: '最大时长',
          sortOccurredAt: '发生时间',
          sortDuration: '耗时',
          sortStatusCode: '状态码',
          sortAsc: '升序',
          sortDesc: '降序',
        },
        builder: {
          title: '筛选条件',
          hint: 'hint',
          fields: {
            requestId: '请求 ID',
            traceId: 'Trace ID',
            userId: '用户 ID',
            username: '用户名',
            method: '方法',
            path: '路径',
            statusCode: '状态码',
            durationMinMs: '最小耗时',
            durationMaxMs: '最大耗时',
          },
        },
      },
    },
  },
});

describe('AccessLogFilters', () => {
  it('renders one sorter tag and clears the whole sorter at once', () => {
    const wrapper = mount(AccessLogFilters, {
      props: {
        activePreset: 'all',
        modelValue: {
          keyword: '',
          requestId: 'req-1',
          traceId: '',
          userId: '',
          username: '',
          method: '',
          path: '',
          pathMatch: 'exact',
          route: '',
          statusCode: '',
          durationMinMs: '',
          durationMaxMs: '',
          occurredRange: [],
          sorters: [{ field: 'occurred_at', direction: 'desc' }],
        },
        presets: [],
      },
      global: {
        plugins: [i18n],
        stubs: {
          'management-toolbar': passthroughStub,
          't-button': buttonStub,
          't-tag': tagStub,
          't-input': inputStub,
          't-popup': passthroughStub,
          't-date-range-picker': dateRangeStub,
          't-select': selectStub,
        },
      },
    });

    expect(wrapper.find('input').attributes('placeholder')).toBe('搜索请求 ID、Trace ID、路径、用户名');
    expect(wrapper.text()).toContain('排序：发生时间 ↓');
    expect(wrapper.text()).toContain('请求 ID：req-1');

    const tags = wrapper.findAllComponents(tagStub);
    tags[0]?.vm.$emit('close');

    expect(wrapper.emitted('update:modelValue')?.[0]?.[0]).toMatchObject({
      sorters: [],
    });
  });
});
