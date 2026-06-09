// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import { defineComponent, h } from 'vue';
import { createI18n } from 'vue-i18n';

import AccessLogFilters from './AccessLogFilters.vue';

const logFilterBuilderStub = defineComponent({
  name: 'AdvancedQueryFilterBuilderStub',
  props: [
    'keyword',
    'keywordPlaceholder',
    'tags',
    'timeFields',
    'fields',
    'sorters',
    'sortAddDisabled',
    'sortFieldOptionsByIndex',
    'sortMoveUpDisabled',
    'sortMoveDownDisabled',
    'removeSorterLabel',
    'moveUpLabel',
    'moveDownLabel',
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
        h('span', { 'data-testid': 'sort-move-up-disabled' }, JSON.stringify(props.sortMoveUpDisabled)),
        h('span', { 'data-testid': 'sort-move-down-disabled' }, JSON.stringify(props.sortMoveDownDisabled)),
        h('button', { 'data-testid': 'close-sorter', onClick: () => emit('close-tag', 'sorter:0') }),
      ]);
  },
});

const i18n = createI18n({
  legacy: false,
  locale: 'zh-CN',
  messages: {
    'zh-CN': {
      accessLog: {
        page: { searchPlaceholder: '搜索请求 ID、路径、用户名' },
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
          requestId: '请求 ID',
          userId: '用户 ID',
          username: '用户名',
          method: '方法',
          path: '路径',
          statusCode: '状态码',
          durationMin: '最小时长',
          durationMax: '最大时长',
          startedRange: '请求开始时间',
          occurredRange: '请求结束时间',
          sortStartedAt: '请求开始时间',
          sortOccurredAt: '发生时间',
          sortDuration: '耗时',
          sortStatusCode: '状态码',
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
            requestId: '请求 ID',
            userId: '用户 ID',
            username: '用户名',
            method: '方法',
            path: '路径',
            statusCode: '状态码',
            durationMinMs: '最小耗时',
            durationMaxMs: '最大耗时',
            timeRange: '时间范围',
            sorterBuilder: '排序方式',
          },
        },
      },
    },
  },
});

describe('AccessLogFilters', () => {
  it('passes unified tags and time fields to the shared builder', () => {
    const wrapper = mount(AccessLogFilters, {
      props: {
        activePreset: 'all',
        modelValue: {
          keyword: '',
          startedRange: ['2026-05-31 10:00:00', '2026-05-31 11:00:00'],
          occurredRange: ['2026-05-31 10:30:00', '2026-05-31 11:30:00'],
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
          AdvancedQueryFilterBuilder: logFilterBuilderStub,
        },
      },
    });

    expect(wrapper.get('[data-testid="keyword-placeholder"]').text()).toBe('搜索请求 ID、路径、用户名');

    const tags = JSON.parse(wrapper.get('[data-testid="tags"]').text());
    expect(tags.map((tag: { label: string }) => tag.label)).toContain('排序 1: 发生时间 ↓');
    expect(tags.map((tag: { label: string }) => tag.label)).toContain('请求 ID：req-1');
    expect(tags.map((tag: { label: string }) => tag.label)).toContain(
      '请求开始时间：2026-05-31 10:00:00 ~ 2026-05-31 11:00:00',
    );
    expect(tags.map((tag: { label: string }) => tag.label)).toContain(
      '请求结束时间：2026-05-31 10:30:00 ~ 2026-05-31 11:30:00',
    );

    const timeFields = JSON.parse(wrapper.get('[data-testid="time-fields"]').text());
    expect(timeFields).toHaveLength(2);
    expect(timeFields[0].label).toBe('请求开始时间');
    expect(timeFields[1].label).toBe('请求结束时间');
  });

  it('normalizes duplicate sorters and disables add when all fields are used', () => {
    const wrapper = mount(AccessLogFilters, {
      props: {
        activePreset: 'all',
        modelValue: {
          keyword: '',
          startedRange: [],
          occurredRange: [],
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
          sorters: [
            { field: 'started_at', direction: 'desc' },
            { field: 'started_at', direction: 'asc' },
            { field: 'occurred_at', direction: 'asc' },
            { field: 'duration_ms', direction: 'desc' },
            { field: 'status_code', direction: 'asc' },
          ],
        },
        presets: [],
      },
      global: {
        plugins: [i18n],
        stubs: {
          AdvancedQueryFilterBuilder: logFilterBuilderStub,
        },
      },
    });

    expect(JSON.parse(wrapper.get('[data-testid="sorters"]').text())).toHaveLength(4);
    expect(wrapper.get('[data-testid="sort-add-disabled"]').text()).toBe('true');
    expect(JSON.parse(wrapper.get('[data-testid="sort-field-options"]').text())[0]).toEqual([
      { label: '请求开始时间', value: 'started_at' },
    ]);
    expect(JSON.parse(wrapper.get('[data-testid="sort-move-up-disabled"]').text())).toEqual([
      true,
      false,
      false,
      false,
    ]);
    expect(JSON.parse(wrapper.get('[data-testid="sort-move-down-disabled"]').text())).toEqual([
      false,
      false,
      false,
      true,
    ]);
  });

  it('clears one sorter when the shared builder closes sorter tag', async () => {
    const wrapper = mount(AccessLogFilters, {
      props: {
        activePreset: 'all',
        modelValue: {
          keyword: '',
          startedRange: [],
          occurredRange: [],
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
          sorters: [{ field: 'occurred_at', direction: 'desc' }],
        },
        presets: [],
      },
      global: {
        plugins: [i18n],
        stubs: {
          AdvancedQueryFilterBuilder: logFilterBuilderStub,
        },
      },
    });

    await wrapper.get('[data-testid="close-sorter"]').trigger('click');

    expect(wrapper.emitted('update:modelValue')?.[0]?.[0]).toMatchObject({
      sorters: [],
    });
  });
});
