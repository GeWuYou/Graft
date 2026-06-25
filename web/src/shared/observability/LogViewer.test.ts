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
  basicInfoLabel: '基础信息',
  collapseDetailLabel: '收起详情',
  copyErrorLabel: '复制失败',
  copyJsonLabel: '复制 JSON',
  copyLabel: '复制全部',
  copyLineLabel: '复制本行',
  copySuccessLabel: '复制成功',
  downloadLabel: '下载',
  emptyLabel: '暂无日志',
  detailTitleLabel: '日志详情',
  importantFieldsLabel: '关键字段',
  levelLabel: '级别',
  levelFilterLabel: '级别',
  matchCountLabel: '{count} 个匹配',
  metadataLabel: 'Metadata',
  messageLabel: '完整消息',
  rawLabel: '原始日志',
  copyMessageLabel: '复制消息',
  refreshLabel: '刷新日志',
  refreshScrollLabel: '跟随底部',
  refreshScrollTooltipLabel: '刷新日志后自动滚动到底部',
  retryLabel: '重试',
  searchPlaceholder: '搜索日志内容',
  sourceLabel: '来源',
  timeLabel: '时间',
  truncatedLabel: '日志已截断',
  viewDetailLabel: '查看详情',
  wrapLabel: '自动换行',
  viewerMode: false,
  viewerStorageKey: 'graft.test.log-viewer.height',
  fullscreenLabel: '全屏',
  exitFullscreenLabel: '退出全屏',
  resizeHandleLabel: '调整阅读器高度',
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

  it('keeps the log toolbar split into balanced action and filter groups', () => {
    const wrapper = mount(LogViewer, {
      props: {
        ...labels,
        lines: createLines(2),
      },
      global: { stubs: tdesignStubs },
    });

    expect(wrapper.find('.log-viewer__toolbar-left').text()).toContain('刷新日志');
    expect(wrapper.find('.log-viewer__toolbar-left').text()).toContain('复制全部');
    expect(wrapper.find('.log-viewer__toolbar-left').text()).toContain('下载');
    expect(wrapper.find('.log-viewer__toolbar-right').text()).toContain('自动换行');
    expect(wrapper.find('.log-viewer__toolbar-right').text()).toContain('跟随底部');
    expect(wrapper.find('[data-tooltip="刷新日志后自动滚动到底部"]').exists()).toBe(true);
  });

  it('offers every parsed log level in the level filter', () => {
    const wrapper = mount(LogViewer, {
      props: {
        ...labels,
        lines: createLines(1),
      },
      global: { stubs: tdesignStubs },
    });

    const filterOptions = wrapper
      .findAll('select')[1]
      .findAll('option')
      .map((option) => option.text());

    expect(filterOptions).toEqual(['级别: 全部', 'FATAL', 'ERROR', 'WARN', 'INFO', 'DEBUG', 'TRACE', 'LOG', 'UNKNOWN']);
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
    expect(contentColumn.text()).toContain('duration=12ms');
    expect(contentColumn.text()).toContain('+3');
    expect(contentColumn.text()).not.toContain('component=http');
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

  it('opens a drawer with formatted JSON metadata and raw log text instead of expanding the stream', async () => {
    const wrapper = mount(LogViewer, {
      props: {
        ...labels,
        lines: [
          '2026-06-17T06:31:42.585+0800 ERROR middleware/logger.go:61 http request failed {"service":"sub2api","request_id":"abc","path":"/v1/responses","method":"POST","user_id":1,"group_id":6,"model":"gpt-5.5","status_code":500}',
        ],
      },
      global: { stubs: tdesignStubs },
    });

    await wrapper.find('.log-viewer__icon-action').trigger('click');

    expect(wrapper.find('.log-viewer__line-detail').exists()).toBe(false);
    expect(wrapper.text()).toContain('日志详情');
    expect(wrapper.find('.log-viewer__summary').text()).toContain('http request failed');
    expect(wrapper.find('.log-viewer__summary-title').text()).toContain('ERROR');
    expect(wrapper.find('.log-viewer__field-chips').text()).toContain('request_id=abc');
    expect(wrapper.find('.log-viewer__field-chips').text()).toContain('path=/v1/responses');
    expect(wrapper.find('.log-viewer__field-chips').text()).toContain('model=gpt-5.5');
    expect(wrapper.find('.log-viewer__basic').text()).toContain('级别');
    expect(wrapper.find('.log-viewer__level-value').text()).toBe('ERROR');
    expect(wrapper.find('.log-viewer__detail-drawer').text()).toContain('http request failed');
    expect(wrapper.findAll('.log-viewer__code-block')[0].text()).toContain('"status_code": 500');
    expect(wrapper.findAll('.log-viewer__code-block')[1].text()).toContain('http request failed');
    expect(wrapper.find('.log-viewer__basic-info').exists()).toBe(false);
    expect(wrapper.find('.log-viewer__line--danger').exists()).toBe(true);
    expect(wrapper.find('.log-viewer__line--active').exists()).toBe(true);
    expect(wrapper.find('.log-viewer__drawer-actions').exists()).toBe(false);
    expect(wrapper.text()).not.toContain('复制消息');
  });

  it('renders logfmt details from parsed message and hides missing placeholders', async () => {
    const wrapper = mount(LogViewer, {
      props: {
        ...labels,
        lines: ['time=2026-06-16T22:27:57.106Z level=INFO msg="server run start"'],
      },
      global: { stubs: tdesignStubs },
    });

    expect(wrapper.find('.log-viewer__message').text()).toBe('server run start');
    expect(wrapper.find('.log-viewer__metadata-tags').text()).toContain('time=2026-06-16T22:27:57.106Z');
    expect(wrapper.find('.log-viewer__metadata-tags').text()).not.toContain('msg=server run start');

    await wrapper.find('.log-viewer__icon-action').trigger('click');

    expect(wrapper.find('.log-viewer__summary-title').text()).toContain('INFO');
    expect(wrapper.find('.log-viewer__summary-title').text()).toContain('server run start');
    expect(wrapper.find('.log-viewer__summary-meta').text()).toBe('2026-06-16T22:27:57.106Z');
    expect(wrapper.find('.log-viewer__field-chips').text()).toContain('time=2026-06-16T22:27:57.106Z');
    expect(wrapper.find('.log-viewer__field-chips').text()).toContain('level=INFO');
    expect(wrapper.find('.log-viewer__field-chips').text()).toContain('msg=server run start');
    expect(wrapper.find('.log-viewer__basic').text()).toContain('时间');
    expect(wrapper.find('.log-viewer__basic').text()).toContain('级别');
    expect(wrapper.find('.log-viewer__basic').text()).toContain('完整消息');
    expect(wrapper.find('.log-viewer__basic').text()).not.toContain('来源');
    expect(wrapper.text()).not.toContain('- · -');
    expect(wrapper.findAll('.log-viewer__code-block')[0].text()).toContain('"msg": "server run start"');
  });

  it('renders plain text details without empty metadata or field sections', async () => {
    const wrapper = mount(LogViewer, {
      props: {
        ...labels,
        lines: ['GitHub MCP Server running on stdio'],
      },
      global: { stubs: tdesignStubs },
    });

    expect(wrapper.find('.log-viewer__message').text()).toBe('GitHub MCP Server running on stdio');
    expect(wrapper.find('.log-viewer__metadata-tags').exists()).toBe(false);

    await wrapper.find('.log-viewer__icon-action').trigger('click');

    expect(wrapper.find('.log-viewer__summary-title').text()).toContain('LOG');
    expect(wrapper.find('.log-viewer__summary-title').text()).toContain('GitHub MCP Server running on stdio');
    expect(wrapper.find('.log-viewer__summary-meta').exists()).toBe(false);
    expect(wrapper.find('.log-viewer__field-chips').exists()).toBe(false);
    expect(wrapper.text()).not.toContain('Metadata');
    expect(wrapper.text()).not.toContain('{}');
    expect(wrapper.find('.log-viewer__basic').text()).toContain('完整消息');
    expect(wrapper.find('.log-viewer__basic').text()).not.toContain('时间');
    expect(wrapper.find('.log-viewer__basic').text()).not.toContain('来源');
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

  it('can render inside the shared viewer frame mode', () => {
    const wrapper = mount(LogViewer, {
      props: {
        ...labels,
        viewerMode: true,
        lines: createLines(1),
      },
      global: { stubs: tdesignStubs },
    });

    expect(wrapper.find('.log-viewer--framed').exists()).toBe(true);
    expect(wrapper.find('.log-viewer__body').exists()).toBe(true);
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
  'content-viewer-frame': defineComponent({
    setup(_, { slots }) {
      return () => h('section', { class: 'content-viewer-frame-stub' }, [slots.toolbar?.(), slots.default?.()]);
    },
  }),
  't-drawer': defineComponent({
    props: ['header', 'visible'],
    emits: ['close', 'update:visible'],
    setup:
      (props, { slots }) =>
      () =>
        props.visible ? h('aside', [h('h2', String(props.header ?? '')), slots.default?.()]) : null,
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
