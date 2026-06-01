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
  props: ['modelValue', 'placeholder'],
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    return () =>
      h('button', {
        type: 'button',
        'data-placeholder': Array.isArray(props.placeholder) ? props.placeholder.join('|') : props.placeholder,
        onClick: () => emit('update:modelValue', ['2026-05-31 10:00:00', '2026-05-31 11:00:00']),
      });
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
        time: {
          last24h: '最近 24 小时',
          last7d: '最近 7 天',
          last30d: '最近 30 天',
          custom: '自定义时间范围',
        },
        page: { searchPlaceholder: '搜索请求 ID、路径、用户名' },
        actions: { search: '查询', reset: '重置', addFilter: '添加筛选条件' },
        presets: { label: '快捷筛选' },
        sort: {
          title: '排序',
          tagPrefix: '排序',
          fieldPlaceholder: '排序字段',
          directionPlaceholder: '排序方向',
        },
        filters: {
          startedRange: '请求时间',
          requestId: '请求 ID',
          userId: '用户 ID',
          username: '用户名',
          method: '方法',
          path: '路径',
          statusCode: '状态码',
          durationMin: '最小时长',
          durationMax: '最大时长',
          sortStartedAt: '请求开始时间',
          sortOccurredAt: '发生时间',
          sortDuration: '耗时',
          sortStatusCode: '状态码',
          sortAsc: '升序',
          sortDesc: '降序',
        },
        builder: {
          title: '筛选条件',
          hint: 'hint',
          groups: {
            time: '时间范围',
            filters: '筛选条件',
            sort: '排序方式',
          },
          summary: {
            customTime: '{range}',
            sortBy: '按{field}{direction}',
          },
          fields: {
            requestId: '请求 ID',
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
          startedRange: [],
          requestId: 'req-1',
          userId: '',
          username: '',
          method: '',
          path: '',
          pathMatch: 'exact',
          route: '',
          statusCode: '',
          durationMinMs: '',
          durationMaxMs: '',
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

    expect(wrapper.find('input').attributes('placeholder')).toBe('搜索请求 ID、路径、用户名');
    expect(wrapper.text()).toContain('排序：发生时间 ↓');
    expect(wrapper.text()).toContain('请求 ID：req-1');

    const tags = wrapper.findAllComponents(tagStub);
    tags[0]?.vm.$emit('close');

    expect(wrapper.emitted('update:modelValue')?.[0]?.[0]).toMatchObject({
      sorters: [],
    });
  });

  it('renders unified request time summary and started range tag', () => {
    const wrapper = mount(AccessLogFilters, {
      props: {
        activePreset: 'all',
        modelValue: {
          keyword: '',
          startedRange: ['2026-05-31 10:00:00', '2026-05-31 11:00:00'],
          requestId: '',
          userId: '',
          username: '',
          method: '',
          path: '',
          pathMatch: 'exact',
          route: '',
          statusCode: '',
          durationMinMs: '',
          durationMaxMs: '',
          sorters: [],
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

    expect(wrapper.text()).toContain('2026-05-31 10:00:00 ~ 2026-05-31 11:00:00');
    expect(wrapper.text()).toContain('时间范围');
  });
});
