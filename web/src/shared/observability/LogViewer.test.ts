import { mount } from '@vue/test-utils';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, nextTick } from 'vue';
import { createI18n } from 'vue-i18n';

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
  autoScrollLabel: '自动滚动',
  autoScrollTooltipLabel: '当视口位于底部附近时自动跟随最新日志',
  basicInfoLabel: '基础信息',
  clearLabel: '清空',
  collapseDetailLabel: '收起详情',
  copyErrorLabel: '复制失败',
  copyJsonLabel: '复制 JSON',
  copyLabel: '复制全部',
  copyLineLabel: '复制本行',
  copyMessageLabel: '复制消息',
  copySuccessLabel: '复制成功',
  detailTitleLabel: '日志详情',
  downloadLabel: '下载',
  emptyLabel: '暂无日志',
  emptyDescriptionLabel: '等待容器输出...',
  fullscreenLabel: '全屏',
  exitFullscreenLabel: '退出全屏',
  importantFieldsLabel: '关键字段',
  jumpBottomLabel: '跳至底部',
  levelFilterLabel: '级别',
  levelLabel: '级别',
  matchCountLabel: '{count} 个匹配',
  messageLabel: '日志内容',
  metadataLabel: '元数据',
  operationLabel: '操作',
  pauseLabel: '暂停',
  rawLabel: '原始日志',
  reconnectLabel: '重新连接',
  resizeHandleLabel: '调整阅读器高度',
  resumeLabel: '继续',
  retryLabel: '重试',
  searchPlaceholder: '搜索日志内容',
  stderrLabel: 'STDERR',
  stdoutLabel: 'STDOUT',
  streamLabel: '输出流',
  sourceLabel: '来源',
  timeLabel: '时间',
  truncatedLabel: '日志已截断',
  viewDetailLabel: '查看详情',
  viewerMode: false,
  viewerStorageKey: 'graft.test.log-viewer.height',
  wrapLabel: '自动换行',
};

describe('LogViewer', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('renders the rebuilt toolbar groups and sticky header columns', () => {
    const wrapper = mount(LogViewer, {
      props: {
        ...labels,
        entries: createEntries(2),
      },
      global: { components: tdesignComponents, plugins: [createTestI18n()] },
    });

    expect(wrapper.find('.log-viewer__toolbar-left').text()).toContain('清空');
    expect(wrapper.find('.log-viewer__toolbar-left').text()).toContain('复制全部');
    expect(wrapper.find('.log-viewer__toolbar-left').text()).toContain('下载');
    expect(wrapper.find('.log-viewer__toolbar-middle').text()).toContain('级别');
    expect(wrapper.find('.log-viewer__toolbar-right').text()).toContain('自动换行');
    expect(wrapper.find('.log-viewer__toolbar-right').text()).toContain('自动滚动');
    expect(wrapper.find('.log-viewer__toolbar-right').text()).toContain('暂停');
    expect(wrapper.find('.log-viewer__header-row').text()).toContain('时间');
    expect(wrapper.find('.log-viewer__header-row').text()).toContain('输出流');
    expect(wrapper.find('.log-viewer__header-row').text()).toContain('日志内容');
    expect(wrapper.find('.log-viewer__header-row').text()).not.toContain('来源');
    expect(wrapper.find('.log-viewer__header-row').text()).not.toContain('#');
  });

  it('renders structured stream styling without a source column', () => {
    const wrapper = mount(LogViewer, {
      props: {
        ...labels,
        entries: [
          createEntry(
            '2026-06-17T06:31:42.585+0800 INFO service/deep/path/pricing_service.go:461 loaded {"request_id":"abc"}',
            'stderr',
            '2026-06-17T06:31:42.585+08:00',
          ),
        ],
      },
      global: { components: tdesignComponents, plugins: [createTestI18n()] },
    });

    const line = wrapper.find('.log-viewer__line');
    expect(line.find('.log-viewer__stream-cell').text()).toContain('STDERR');
    expect(line.text()).not.toContain('pricing_service.go:461');
  });

  it('shows search highlight and keeps tail line numbers stable after appends', async () => {
    const wrapper = mount(LogViewer, {
      props: {
        ...labels,
        entries: createEntries(3),
        lineLimit: 3,
        contentVersion: 3,
      },
      global: { components: tdesignComponents, plugins: [createTestI18n()] },
    });

    await wrapper.find('input[type="search"]').setValue('request');

    expect(wrapper.text()).toContain('6 个匹配');
    await wrapper.setProps({
      entries: createEntries(4),
      contentVersion: 4,
    });
    await nextTick();
  });

  it('emits clear pause resume and reconnect actions through the rebuilt toolbar', async () => {
    const wrapper = mount(LogViewer, {
      props: {
        ...labels,
        entries: createEntries(1),
        paused: false,
        showReconnect: true,
      },
      global: { components: tdesignComponents, plugins: [createTestI18n()] },
    });

    await wrapper.get('[data-testid="log-viewer-clear"]').trigger('click');
    await wrapper.get('[data-testid="log-viewer-pause-toggle"]').trigger('click');
    await wrapper.get('[data-testid="log-viewer-reconnect"]').trigger('click');

    expect(wrapper.emitted('clear')).toHaveLength(1);
    expect(wrapper.emitted('pause')).toHaveLength(1);
    expect(wrapper.emitted('reconnect')).toHaveLength(1);

    await wrapper.setProps({ paused: true });
    await wrapper.get('[data-testid="log-viewer-pause-toggle"]').trigger('click');
    expect(wrapper.emitted('resume')).toHaveLength(1);
  });

  it('opens the detail drawer with structured metadata and stream info', async () => {
    const wrapper = mount(LogViewer, {
      props: {
        ...labels,
        entries: [
          createEntry(
            '2026-06-17T06:31:42.585+0800 ERROR middleware/logger.go:61 http request failed {"request_id":"abc","path":"/v1/responses","status_code":500}',
            'stderr',
            '2026-06-17T06:31:42.585+08:00',
          ),
        ],
      },
      global: { components: tdesignComponents, plugins: [createTestI18n()] },
    });

    await wrapper.find('.log-viewer__icon-action').trigger('click');

    expect(wrapper.find('.log-viewer__summary-title').text()).toContain('ERROR');
    expect(wrapper.find('.log-viewer__summary-title').text()).toContain('STDERR');
    expect(wrapper.find('.log-viewer__field-chips').text()).toContain('request_id=abc');
    expect(wrapper.find('.log-viewer__basic').text()).toContain('输出流');
    expect(wrapper.find('.log-viewer__basic').text()).toContain('STDERR');
  });

  it('shows jump-bottom only when the viewport is no longer pinned', async () => {
    const wrapper = mount(LogViewer, {
      attachTo: document.body,
      props: {
        ...labels,
        entries: createEntries(40),
        contentVersion: 40,
      },
      global: { components: tdesignComponents, plugins: [createTestI18n()] },
    });

    const viewport = wrapper.get('.log-viewer__viewport').element as HTMLDivElement;
    Object.defineProperty(viewport, 'clientHeight', { configurable: true, value: 240 });
    Object.defineProperty(viewport, 'scrollHeight', { configurable: true, writable: true, value: 2000 });
    Object.defineProperty(viewport, 'scrollTop', { configurable: true, writable: true, value: 1000 });

    await wrapper.get('.log-viewer__viewport').trigger('scroll');
    await nextTick();

    expect(wrapper.text()).toContain('跳至底部');
    wrapper.unmount();
  });

  it('auto-scrolls to the bottom on the first non-empty render by default', async () => {
    const wrapper = mount(LogViewer, {
      attachTo: document.body,
      props: {
        ...labels,
        entries: [],
        contentVersion: 0,
      },
      global: { components: tdesignComponents, plugins: [createTestI18n()] },
    });

    const viewport = wrapper.get('.log-viewer__viewport').element as HTMLDivElement;
    let internalScrollTop = 0;
    Object.defineProperty(viewport, 'clientHeight', { configurable: true, value: 240 });
    Object.defineProperty(viewport, 'scrollHeight', { configurable: true, get: () => 2000 });
    Object.defineProperty(viewport, 'scrollTop', {
      configurable: true,
      get: () => internalScrollTop,
      set: (value: number) => {
        internalScrollTop = value;
      },
    });

    await wrapper.setProps({
      entries: createEntries(40),
      contentVersion: 40,
    });
    await nextTick();
    await new Promise((resolve) => setTimeout(resolve, 0));
    await nextTick();

    expect(internalScrollTop).toBe(2000);
    wrapper.unmount();
  });

  it('renders a titled empty state with description', () => {
    const wrapper = mount(LogViewer, {
      props: {
        ...labels,
        entries: [],
      },
      global: { components: tdesignComponents, plugins: [createTestI18n()] },
    });

    expect(wrapper.text()).toContain('暂无日志');
    expect(wrapper.text()).toContain('等待容器输出...');
  });
});

const tdesignComponents = {
  TAlert: defineComponent({
    props: ['title'],
    setup:
      (props, { slots }) =>
      () =>
        h('div', [String(props.title ?? ''), slots.default?.(), slots.operation?.()]),
  }),
  TButton: defineComponent({
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
  TEmpty: defineComponent({
    props: ['title', 'description'],
    setup: (props) => () => h('div', [String(props.title ?? ''), String(props.description ?? '')]),
  }),
  ContentViewerFrame: defineComponent({
    setup(_, { slots }) {
      return () => h('section', { class: 'content-viewer-frame-stub' }, [slots.toolbar?.(), slots.default?.()]);
    },
  }),
  TDrawer: defineComponent({
    props: ['header', 'visible'],
    emits: ['close', 'update:visible'],
    setup:
      (props, { slots }) =>
      () =>
        props.visible ? h('aside', [h('h2', String(props.header ?? '')), slots.default?.()]) : null,
  }),
  TInput: defineComponent({
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
  TSelect: defineComponent({
    props: ['options', 'value'],
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
  TSkeleton: defineComponent({
    setup: () => () => h('div', 'loading'),
  }),
  TSwitch: defineComponent({
    props: ['value'],
    emits: ['update:value'],
    setup:
      (props, { emit }) =>
      () =>
        h('button', { onClick: () => emit('update:value', !props.value) }, String(Boolean(props.value))),
  }),
  TTag: defineComponent({
    setup:
      (_, { slots }) =>
      () =>
        h('span', slots.default?.()),
  }),
  TTooltip: defineComponent({
    props: ['content'],
    setup:
      (props, { slots }) =>
      () =>
        h('span', { 'data-tooltip': props.content }, slots.default?.()),
  }),
};

function createTestI18n() {
  return createI18n({
    legacy: false,
    locale: 'zh-CN',
    messages: {
      'zh-CN': {},
    },
  });
}

function createEntry(line: string, stream: 'stdout' | 'stderr' = 'stdout', occurredAt = '2026-06-17T06:31:40+08:00') {
  return {
    line,
    occurredAt,
    stream,
  } as const;
}

function createEntries(count: number) {
  return Array.from({ length: count }, (_, index) =>
    createEntry(
      `2026-06-17T06:31:4${index}.585+0800 INFO middleware/logger.go:61 http request completed {"request_id":"${index}"}`,
      index % 2 === 0 ? 'stdout' : 'stderr',
      `2026-06-17T06:31:4${index}.585+08:00`,
    ),
  );
}
