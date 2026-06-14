// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import ContainerListPage from './index.vue';

const apiMocks = vi.hoisted(() => ({
  getContainer: vi.fn(),
  getContainerLogs: vi.fn(),
  getContainers: vi.fn(),
  runContainerAction: vi.fn(),
}));

const messageMocks = vi.hoisted(() => ({
  error: vi.fn(),
  success: vi.fn(),
}));

const translations = vi.hoisted(
  (): Record<string, string> => ({
    'container.list.actionFailed': '容器操作失败。',
    'container.list.actionSuccess': '容器操作已提交。',
    'container.list.actions.cancel': '取消',
    'container.list.actions.confirm': '确认',
    'container.list.actions.confirmRestart': '确认重启该容器？',
    'container.list.actions.confirmStart': '确认启动该容器？',
    'container.list.actions.confirmStop': '确认停止该容器？',
    'container.list.actions.detail': '详情',
    'container.list.actions.logs': '日志',
    'container.list.actions.restart': '重启',
    'container.list.actions.start': '启动',
    'container.list.actions.stop': '停止',
    'container.list.clearFilters': '清除筛选',
    'container.list.columns.createdAt': '创建时间',
    'container.list.columns.image': '镜像',
    'container.list.columns.name': '容器',
    'container.list.columns.operation': '操作',
    'container.list.columns.ports': '端口',
    'container.list.columns.restartPolicy': '重启策略',
    'container.list.columns.startedAt': '启动时间',
    'container.list.columns.status': '状态',
    'container.list.copyError': '日志复制失败。',
    'container.list.copySuccess': '日志已复制。',
    'container.list.description': '查看容器状态。',
    'container.list.detail.command': '命令',
    'container.list.detail.entrypoint': '入口',
    'container.list.detail.identity': '基础信息',
    'container.list.detail.inspectUpdatedAt': '详情更新时间',
    'container.list.detail.loadFailed': '容器详情加载失败。',
    'container.list.detail.mountEmpty': '暂无挂载。',
    'container.list.detail.mounts': '挂载',
    'container.list.detail.networkEmpty': '暂无网络信息。',
    'container.list.detail.networks': '网络',
    'container.list.detail.runtime': '运行时',
    'container.list.detail.title': '容器详情',
    'container.list.detail.workingDir': '工作目录',
    'container.list.emptyDescription': '当前容器运行时未返回容器。',
    'container.list.emptyFilteredDescription': '没有符合筛选条件的容器。',
    'container.list.emptyTitle': '暂无容器',
    'container.list.eyebrow': '运维管理',
    'container.list.fields.apiVersion': 'API 版本',
    'container.list.fields.architecture': '架构',
    'container.list.fields.createdAt': '创建时间',
    'container.list.fields.endpoint': 'Endpoint',
    'container.list.fields.id': 'ID',
    'container.list.fields.image': '镜像',
    'container.list.fields.name': '名称',
    'container.list.fields.operatingSystem': '操作系统',
    'container.list.fields.restartPolicy': '重启策略',
    'container.list.fields.runtime': '运行时',
    'container.list.fields.serverVersion': '服务端版本',
    'container.list.fields.startedAt': '启动时间',
    'container.list.fields.state': '状态码',
    'container.list.fields.status': '状态',
    'container.list.filters.allStatuses': '全部状态',
    'container.list.filters.query': '查询',
    'container.list.filters.reset': '重置',
    'container.list.filters.searchPlaceholder': '搜索名称、镜像、ID 或端口',
    'container.list.filters.status': '容器状态',
    'container.list.loadFailed': '容器列表加载失败。',
    'container.list.logs.copy': '复制',
    'container.list.logs.empty': '暂无日志。',
    'container.list.logs.loadFailed': '容器日志加载失败。',
    'container.list.logs.refresh': '刷新日志',
    'container.list.logs.since': '起始时间或时长',
    'container.list.logs.sincePlaceholder': '例如 10m、1h 或 RFC3339',
    'container.list.logs.stderr': 'stderr',
    'container.list.logs.stdout': 'stdout',
    'container.list.logs.tail': '行数',
    'container.list.logs.timestamps': '时间戳',
    'container.list.logs.title': '容器日志',
    'container.list.logs.truncated': '日志已按当前上限截断。',
    'container.list.refresh': '刷新',
    'container.list.retry': '重试',
    'container.list.runtimeContainers': '{running}/{total} 运行中',
    'container.list.runtimeDisabledHint': '请在系统配置中启用容器运行时访问后重试。',
    'container.list.runtimeLabel': '运行时',
    'container.list.runtimeUnavailable': '运行时不可用',
    'container.list.states.created': '已创建',
    'container.list.states.dead': '异常',
    'container.list.states.exited': '已退出',
    'container.list.states.paused': '已暂停',
    'container.list.states.removing': '移除中',
    'container.list.states.restarting': '重启中',
    'container.list.states.running': '运行中',
    'container.list.states.unknown': '未知',
    'container.list.tableHint': '数据来自当前配置的容器运行时。',
    'container.list.tableSummary': '共 {count} 个容器',
    'container.list.title': '容器管理',
    'ops.container.error.runtimeDisabled': '容器运行时访问未启用',
    'ops.container.error.runtimeUnavailable': '容器运行时连接不可用',
  }),
);

vi.mock('../../api/container', () => ({
  getContainer: apiMocks.getContainer,
  getContainerLogs: apiMocks.getContainerLogs,
  getContainers: apiMocks.getContainers,
  runContainerAction: apiMocks.runContainerAction,
}));

vi.mock('tdesign-vue-next', () => ({
  MessagePlugin: messageMocks,
}));

vi.mock('tdesign-icons-vue-next', () => ({
  RefreshIcon: defineComponent({ name: 'RefreshIcon', setup: () => () => h('span') }),
  SearchIcon: defineComponent({ name: 'SearchIcon', setup: () => () => h('span') }),
}));

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    locale: 'zh-CN',
    t: (key: string, params?: Record<string, unknown>) =>
      (translations[key] ?? key).replace(/\{(\w+)\}/g, (_, name) => String(params?.[name] ?? `{${name}}`)),
  }),
}));

vi.mock('@/shared/observability', async () => {
  const actual = await vi.importActual<typeof import('@/shared/observability')>('@/shared/observability');
  return {
    ...actual,
    formatLocaleDateTime: (value?: string | null) => value || '-',
  };
});

describe('container list page', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    apiMocks.getContainers.mockResolvedValue({
      items: [
        {
          id: 'container-1',
          names: ['graft-web'],
          image: 'graft/web:latest',
          status: 'Up 10 minutes',
          state: 'running',
          runtime: 'first-adapter',
          created_at: '2026-06-14T01:00:00Z',
          started_at: '2026-06-14T01:05:00Z',
          ports: [{ private_port: 80, public_port: 8080, type: 'tcp' }],
          restart_policy: 'unless-stopped',
        },
      ],
      runtime: {
        runtime: 'first-adapter',
        status: 'enabled',
        endpoint: 'unix:///var/run/docker.sock',
        containers_running: 1,
        containers_total: 1,
      },
    });
    apiMocks.getContainer.mockResolvedValue({
      id: 'container-1',
      names: ['graft-web'],
      image: 'graft/web:latest',
      status: 'Up 10 minutes',
      state: 'running',
      command: ['npm', 'run', 'serve'],
      entrypoint: ['docker-entrypoint.sh'],
      runtime: 'first-adapter',
      created_at: '2026-06-14T01:00:00Z',
      started_at: '2026-06-14T01:05:00Z',
      ports: [],
      mounts: [],
      networks: [],
      runtime_info: {
        runtime: 'first-adapter',
        status: 'enabled',
        endpoint: 'unix:///var/run/docker.sock',
      },
    });
    apiMocks.getContainerLogs.mockResolvedValue({
      id: 'container-1',
      runtime: 'first-adapter',
      tail: 200,
      truncated: false,
      stdout: true,
      stderr: true,
      timestamps: false,
      lines: ['server started'],
    });
  });

  it('loads and renders container rows with required operation buttons', async () => {
    const wrapper = mountPage();
    await flushPromises();

    expect(apiMocks.getContainers).toHaveBeenCalledTimes(1);
    expect(wrapper.text()).toContain('容器管理');
    expect(wrapper.text()).toContain('graft-web');
    expect(wrapper.text()).toContain('graft/web:latest');
    expect(wrapper.text()).toContain('8080->80/tcp');
    expect(wrapper.text()).toContain('运行中');
    expect(wrapper.text()).toContain('详情');
    expect(wrapper.text()).toContain('日志');
    expect(wrapper.text()).toContain('启动');
    expect(wrapper.text()).toContain('停止');
    expect(wrapper.text()).toContain('重启');
  });

  it('opens detail and log drawers through module API actions', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('[data-testid="container-action-detail"]').trigger('click');
    await flushPromises();

    expect(apiMocks.getContainer).toHaveBeenCalledWith('container-1');
    expect(wrapper.text()).toContain('容器详情');
    expect(wrapper.text()).toContain('npm run serve');
    expect(wrapper.text()).toContain('docker-entrypoint.sh');

    await wrapper.get('[data-testid="container-action-logs"]').trigger('click');
    await flushPromises();

    expect(apiMocks.getContainerLogs).toHaveBeenCalledWith('container-1', {
      tail: 200,
      since: undefined,
      stderr: true,
      stdout: true,
      timestamps: false,
    });
    expect(wrapper.text()).toContain('server started');
  });

  it('renders runtime disabled as an access configuration error with system config hint', async () => {
    apiMocks.getContainers.mockRejectedValue(
      apiError('ops.container.error.runtimeDisabled', 'Container runtime is disabled'),
    );

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain('容器运行时访问未启用');
    expect(wrapper.text()).toContain('请在系统配置中启用容器运行时访问后重试。');
    expect(wrapper.text()).not.toContain('容器模块');
    expect(wrapper.text()).not.toContain('module is disabled');
  });

  it('renders runtime connection failures without implying the module is disabled', async () => {
    apiMocks.getContainers.mockRejectedValue(
      apiError('ops.container.error.runtimeUnavailable', 'Container runtime is unavailable'),
    );

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain('容器运行时连接不可用');
    expect(wrapper.text()).not.toContain('请在系统配置中启用容器运行时访问后重试。');
    expect(wrapper.text()).not.toContain('容器模块');
    expect(wrapper.text()).not.toContain('module is disabled');
  });
});

function apiError(messageKey: string, message: string) {
  return {
    code: 'COMMON_INTERNAL_ERROR',
    isApiRequestError: true,
    message,
    messageKey,
    status: 500,
  };
}

function mountPage() {
  return mount(ContainerListPage, {
    global: {
      directives: {
        permission: () => undefined,
      },
      stubs: {
        'management-page-header': defineComponent({
          props: ['title', 'description'],
          setup:
            (props, { slots }) =>
            () =>
              h('header', [
                h('h1', props.title as string),
                h('p', props.description as string),
                slots.meta?.(),
                slots.actions?.(),
              ]),
        }),
        'management-toolbar': defineComponent({
          setup:
            (_, { slots }) =>
            () =>
              h('section', [slots.filters?.()]),
        }),
        'management-table-card': defineComponent({
          setup:
            (_, { slots }) =>
            () =>
              h('section', [slots.head?.(), slots.toolbar?.(), slots.default?.()]),
        }),
        'table-view-toolbar': defineComponent({
          props: ['refreshLabel'],
          emits: ['refresh'],
          setup:
            (props, { emit }) =>
            () =>
              h('button', { onClick: () => emit('refresh') }, String(props.refreshLabel ?? '')),
        }),
        't-alert': defineComponent({
          props: ['title'],
          setup:
            (props, { slots }) =>
            () =>
              h('div', [String(props.title ?? ''), slots.default?.(), slots.operation?.()]),
        }),
        't-button': defineComponent({
          props: ['loading', 'disabled'],
          emits: ['click'],
          setup:
            (props, { attrs, emit, slots }) =>
            () =>
              h(
                'button',
                {
                  ...attrs,
                  disabled: Boolean(props.disabled),
                  onClick: () => emit('click'),
                },
                slots.default?.(),
              ),
        }),
        't-checkbox': defineComponent({
          props: ['modelValue'],
          emits: ['update:modelValue'],
          setup:
            (props, { emit, slots }) =>
            () =>
              h('label', [
                h('input', {
                  checked: Boolean(props.modelValue),
                  type: 'checkbox',
                  onInput: (event: Event) => emit('update:modelValue', (event.target as HTMLInputElement).checked),
                }),
                slots.default?.(),
              ]),
        }),
        't-descriptions': defineComponent({
          props: ['title'],
          setup:
            (props, { slots }) =>
            () =>
              h('section', [h('h2', String(props.title ?? '')), slots.default?.()]),
        }),
        't-descriptions-item': defineComponent({
          props: ['label'],
          setup:
            (props, { slots }) =>
            () =>
              h('div', [h('strong', String(props.label ?? '')), slots.default?.()]),
        }),
        't-drawer': defineComponent({
          props: ['visible', 'header'],
          setup:
            (props, { slots }) =>
            () =>
              props.visible ? h('aside', [h('h2', String(props.header ?? '')), slots.default?.()]) : null,
        }),
        't-empty': defineComponent({
          props: ['title', 'description'],
          setup:
            (props, { slots }) =>
            () =>
              h('div', [String(props.title ?? ''), String(props.description ?? ''), slots.action?.()]),
        }),
        't-form': defineComponent({
          setup:
            (_, { slots }) =>
            () =>
              h('form', slots.default?.()),
        }),
        't-form-item': defineComponent({
          setup:
            (_, { slots }) =>
            () =>
              h('div', slots.default?.()),
        }),
        't-input': defineComponent({
          props: ['modelValue'],
          emits: ['update:modelValue', 'enter'],
          setup:
            (props, { emit }) =>
            () =>
              h('input', {
                value: props.modelValue,
                onInput: (event: Event) => emit('update:modelValue', (event.target as HTMLInputElement).value),
                onKeydown: (event: KeyboardEvent) => {
                  if (event.key === 'Enter') emit('enter');
                },
              }),
        }),
        't-input-number': defineComponent({
          props: ['modelValue'],
          emits: ['update:modelValue'],
          setup:
            (props, { emit }) =>
            () =>
              h('input', {
                type: 'number',
                value: props.modelValue,
                onInput: (event: Event) => emit('update:modelValue', Number((event.target as HTMLInputElement).value)),
              }),
        }),
        't-loading': defineComponent({
          setup:
            (_, { slots }) =>
            () =>
              h('div', slots.default?.()),
        }),
        't-option': defineComponent({
          props: ['label', 'value'],
          setup: (props) => () => h('option', { value: props.value }, String(props.label ?? '')),
        }),
        't-popconfirm': defineComponent({
          emits: ['confirm'],
          setup:
            (_, { emit, slots }) =>
            () =>
              h('span', { onClick: () => emit('confirm') }, slots.default?.()),
        }),
        't-select': defineComponent({
          props: ['modelValue'],
          emits: ['update:modelValue'],
          setup:
            (props, { emit, slots }) =>
            () =>
              h(
                'select',
                {
                  value: props.modelValue,
                  onInput: (event: Event) => emit('update:modelValue', (event.target as HTMLSelectElement).value),
                },
                slots.default?.(),
              ),
        }),
        't-space': defineComponent({
          setup:
            (_, { slots }) =>
            () =>
              h('div', slots.default?.()),
        }),
        't-table': defineComponent({
          props: ['data'],
          setup:
            (props, { slots }) =>
            () =>
              h(
                'div',
                (props.data as Array<Record<string, unknown>>).length
                  ? (props.data as Array<Record<string, unknown>>).map((row) =>
                      h('div', { key: String(row.id) }, [
                        slots.state?.({ row }),
                        slots.name?.({ row }),
                        slots.image?.({ row }),
                        slots.ports?.({ row }),
                        slots.created_at?.({ row }),
                        slots.started_at?.({ row }),
                        slots.restart_policy?.({ row }),
                        slots.operation?.({ row }),
                      ]),
                    )
                  : slots.empty?.(),
              ),
        }),
        't-tag': defineComponent({
          setup:
            (_, { slots }) =>
            () =>
              h('span', slots.default?.()),
        }),
      },
    },
  });
}
