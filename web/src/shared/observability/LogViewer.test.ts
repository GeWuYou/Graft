// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import LogViewer from './LogViewer.vue';

vi.mock('tdesign-icons-vue-next', () => ({
  BrowseIcon: defineComponent({ setup: () => () => h('span', 'detail-icon') }),
  CopyIcon: defineComponent({ setup: () => () => h('span', 'copy-icon') }),
}));

vi.mock('tdesign-vue-next/es/message', () => ({
  MessagePlugin: {
    error: vi.fn(),
    success: vi.fn(),
  },
}));

const labels = {
  allLevelsLabel: '全部',
  collapseDetailLabel: '收起详情',
  copyErrorLabel: '复制失败',
  copyJsonLabel: '复制 JSON',
  copyLabel: '复制全部',
  copyLineLabel: '复制本行',
  copySuccessLabel: '复制成功',
  downloadLabel: '下载',
  emptyLabel: '暂无日志',
  levelFilterLabel: '级别',
  matchCountLabel: '{count} 个匹配',
  metadataLabel: 'Metadata',
  messageLabel: '完整消息',
  rawLabel: '原始日志',
  copyMessageLabel: '复制消息',
  refreshLabel: '刷新日志',
  refreshScrollLabel: '刷新后滚到底部',
  retryLabel: '重试',
  searchPlaceholder: '搜索日志内容',
  truncatedLabel: '日志已截断',
  viewDetailLabel: '查看详情',
  wrapLabel: '自动换行',
};

describe('LogViewer', () => {
  it('enables wrapping by default and keeps horizontal scrolling inside the log viewport', () => {
    const wrapper = mount(LogViewer, {
      props: {
        ...labels,
        lines: createLines(2),
      },
      global: { stubs: tdesignStubs },
    });

    expect(wrapper.find('.log-viewer__viewport--wrap').exists()).toBe(true);
    expect(wrapper.find('.log-viewer__viewport').classes()).toContain('log-viewer__viewport--wrap');
  });

  it('limits metadata tags and folds repeated low-signal fields out of the main row', () => {
    const wrapper = mount(LogViewer, {
      props: {
        ...labels,
        lines: [
          '2026-06-17T06:31:42.585+0800 INFO middleware/logger.go:61 http request completed {"service":"sub2api","env":"production","component":"http","request_id":"abc","duration":"12ms"}',
        ],
      },
      global: { stubs: tdesignStubs },
    });

    const contentColumn = wrapper.find('.log-viewer__content');
    const metadataTags = wrapper.find('.log-viewer__metadata-tags');

    expect(contentColumn.text()).toContain('request_id=abc');
    expect(contentColumn.text()).toContain('component=http');
    expect(contentColumn.text()).toContain('+2');
    expect(contentColumn.text()).not.toContain('service=sub2api');
    expect(contentColumn.text()).not.toContain('env=production');
    expect(contentColumn.text()).not.toContain('{"service":"sub2api"');
    expect(metadataTags.element.parentElement).toBe(contentColumn.element);
  });

  it('renders each log row as fixed metadata columns plus one flexible content column', () => {
    const wrapper = mount(LogViewer, {
      props: {
        ...labels,
        lines: [
          '2026-06-17T06:31:42.585+0800 INFO service/deep/path/pricing_service.go:461 loaded {"request_id":"abc"}',
        ],
      },
      global: { stubs: tdesignStubs },
    });

    const line = wrapper.find('.log-viewer__line');
    expect(line.find(':scope > .log-viewer__line-number').exists()).toBe(true);
    expect(line.find(':scope > .log-viewer__timestamp-cell').exists()).toBe(true);
    expect(line.find(':scope > .log-viewer__level-cell').exists()).toBe(true);
    expect(line.find(':scope > .log-viewer__source-cell').exists()).toBe(true);
    expect(line.find(':scope > .log-viewer__content').exists()).toBe(true);
    expect(line.find(':scope > .log-viewer__row-actions').exists()).toBe(true);
    expect(line.find('.log-viewer__content .log-viewer__metadata-tags').exists()).toBe(true);
    expect(wrapper.find('.log-viewer__line-main').exists()).toBe(false);
  });

  it('shows short source text while keeping the full source in tooltip content', () => {
    const wrapper = mount(LogViewer, {
      props: {
        ...labels,
        lines: [
          '2026-06-17T06:31:42.585+0800 INFO service/deep/path/pricing_service.go:461 loaded {"request_id":"abc"}',
        ],
      },
      global: { stubs: tdesignStubs },
    });

    expect(wrapper.find('.log-viewer__source').text()).toBe('pricing_service.go:461');
    expect(wrapper.find('[data-tooltip="service/deep/path/pricing_service.go:461"]').exists()).toBe(true);
  });

  it('shows search highlight and match count without filtering order', async () => {
    const wrapper = mount(LogViewer, {
      props: {
        ...labels,
        lines: createLines(3),
      },
      global: { stubs: tdesignStubs },
    });

    await wrapper.find('input[type="search"]').setValue('request');

    expect(wrapper.text()).toContain('6 个匹配');
    expect(wrapper.findAll('.log-viewer__token--keyword')).toHaveLength(3);
    expect(wrapper.findAll('.log-viewer__line-number').map((node) => node.text())).toEqual(['1', '2', '3']);
  });

  it('expands a line to show formatted JSON metadata and raw log text', async () => {
    const wrapper = mount(LogViewer, {
      props: {
        ...labels,
        lines: [
          '2026-06-17T06:31:42.585+0800 ERROR middleware/logger.go:61 http request failed {"service":"sub2api","status":500}',
        ],
      },
      global: { stubs: tdesignStubs },
    });

    await wrapper.find('.log-viewer__icon-action').trigger('click');

    expect(wrapper.find('.log-viewer__message-full').text()).toContain('http request failed');
    expect(wrapper.find('.log-viewer__json').text()).toContain('"status": 500');
    expect(wrapper.find('.log-viewer__raw').text()).toContain('http request failed');
    expect(wrapper.find('.log-viewer__line--danger').exists()).toBe(true);
  });

  it('keeps row actions visually weak until hover or focus reveals them', () => {
    const wrapper = mount(LogViewer, {
      props: {
        ...labels,
        lines: createLines(1),
      },
      global: { stubs: tdesignStubs },
    });

    expect(wrapper.find('.log-viewer__row-actions').exists()).toBe(true);
    expect(wrapper.text()).not.toContain('查看详情');
    expect(wrapper.findAll('.log-viewer__icon-action')).toHaveLength(2);
  });
});

const tdesignStubs = {
  't-alert': defineComponent({
    props: ['title'],
    setup:
      (props, { slots }) =>
      () =>
        h('div', [String(props.title ?? ''), slots.default?.(), slots.operation?.()]),
  }),
  't-button': defineComponent({
    props: ['disabled'],
    emits: ['click'],
    setup:
      (props, { attrs, emit, slots }) =>
      () =>
        h('button', { ...attrs, disabled: Boolean(props.disabled), onClick: () => emit('click') }, [
          slots.icon?.(),
          slots.default?.(),
        ]),
  }),
  't-empty': defineComponent({
    props: ['description'],
    setup: (props) => () => h('div', String(props.description ?? '')),
  }),
  't-input': defineComponent({
    props: ['value'],
    emits: ['update:value'],
    setup:
      (props, { attrs, emit }) =>
      () =>
        h('input', {
          ...attrs,
          type: attrs.type ?? 'text',
          value: props.value,
          onInput: (event: Event) => emit('update:value', (event.target as HTMLInputElement).value),
        }),
  }),
  't-select': defineComponent({
    props: ['modelValue', 'options', 'value'],
    emits: ['change', 'update:value'],
    setup:
      (props, { emit }) =>
      () =>
        h(
          'select',
          {
            value: props.value,
            onChange: (event: Event) => {
              const rawValue = (event.target as HTMLSelectElement).value;
              const value = Number.isNaN(Number(rawValue)) ? rawValue : Number(rawValue);
              emit('update:value', value);
              emit('change', value);
            },
          },
          (props.options as Array<{ label: string; value: string | number }>).map((option) =>
            h('option', { value: option.value }, option.label),
          ),
        ),
  }),
  't-skeleton': defineComponent({
    setup: () => () => h('div', 'loading'),
  }),
  't-switch': defineComponent({
    props: ['value'],
    emits: ['update:value'],
    setup:
      (props, { emit }) =>
      () =>
        h('button', { onClick: () => emit('update:value', !props.value) }, String(Boolean(props.value))),
  }),
  't-tag': defineComponent({
    setup:
      (_, { slots }) =>
      () =>
        h('span', slots.default?.()),
  }),
  't-tooltip': defineComponent({
    props: ['content'],
    setup:
      (props, { slots }) =>
      () =>
        h('span', { 'data-tooltip': props.content }, slots.default?.()),
  }),
};

function createLines(count: number) {
  return Array.from(
    { length: count },
    (_, index) =>
      `2026-06-17T06:31:4${index}.585+0800 INFO middleware/logger.go:61 http request completed {"request_id":"${index}"}`,
  );
}
