// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { readFileSync } from 'node:fs';
import { join } from 'node:path';

import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import ContainerDetailPage from './index.vue';

const sourceText = readFileSync(join(process.cwd(), 'src/modules/container/pages/detail/index.vue'), 'utf8');

const apiMocks = vi.hoisted(() => ({
  getContainer: vi.fn(),
  getContainerLogs: vi.fn(),
}));

const messageMocks = vi.hoisted(() => ({
  error: vi.fn(),
  success: vi.fn(),
}));

const routerMocks = vi.hoisted(() => ({
  back: vi.fn(),
  push: vi.fn(),
  replace: vi.fn(),
}));

const routeState = vi.hoisted(() => ({
  params: { id: 'container-1' },
  query: { tab: 'config' as string | undefined },
}));

const translations = vi.hoisted(
  (): Record<string, string> => ({
    'container.detail.back': '返回',
    'container.detail.config.envName': '变量名',
    'container.detail.config.envPolicy': '策略',
    'container.detail.config.envValue': '值',
    'container.detail.config.environment': '环境变量',
    'container.detail.config.environmentUnavailable': '当前详情契约未返回环境变量。',
    'container.detail.config.hiddenValue': '已隐藏',
    'container.detail.config.maskedValue': '已脱敏',
    'container.detail.config.policy.hidden': '隐藏',
    'container.detail.config.policy.masked': '脱敏',
    'container.detail.config.policy.plain': '明文',
    'container.detail.config.policy.unknown': '未知',
    'container.detail.copy': '复制',
    'container.detail.copyError': '内容复制失败。',
    'container.detail.copySuccess': '内容已复制。',
    'container.detail.description': '查看容器运行时详情、资源、日志、配置、网络和挂载信息。',
    'container.detail.empty': '暂无容器详情。',
    'container.detail.health.restartCount': '重启次数',
    'container.detail.health.status': '健康状态',
    'container.detail.inspectUpdatedAt': '详情更新时间',
    'container.detail.logs.empty': '暂无日志。',
    'container.detail.logs.followTail': '跟随尾部',
    'container.detail.logs.refresh': '刷新日志',
    'container.detail.logs.searchPlaceholder': '搜索日志内容',
    'container.detail.logs.truncated': '日志已按当前上限截断。',
    'container.detail.logs.wrap': '自动换行',
    'container.detail.missingId': '缺少容器标识。',
    'container.detail.network.gateway': '网关',
    'container.detail.network.ipAddress': 'IP 地址',
    'container.detail.network.macAddress': 'MAC 地址',
    'container.detail.network.name': '网络',
    'container.detail.network.ports': '端口映射',
    'container.detail.network.primaryIp': '主 IP',
    'container.detail.network.summary': '网络摘要',
    'container.detail.operation': '操作',
    'container.detail.raw.description': '敏感字段已脱敏，仅用于只读排查。',
    'container.detail.raw.empty': '暂无原始 JSON。',
    'container.detail.raw.error': '原始 JSON 无法格式化。',
    'container.detail.raw.root': 'container',
    'container.detail.raw.source': '源码视图',
    'container.detail.raw.title': '原始 JSON',
    'container.detail.raw.tree': '树形视图',
    'container.detail.refresh': '刷新',
    'container.detail.resources.available': '已采集',
    'container.detail.resources.collectedAt': '采集时间',
    'container.detail.resources.cpu': 'CPU',
    'container.detail.resources.currentSnapshot': '当前快照',
    'container.detail.resources.memory': '内存',
    'container.detail.resources.memoryLimit': '内存上限',
    'container.detail.resources.memoryPercent': '内存百分比',
    'container.detail.resources.memoryUsage': '内存使用',
    'container.detail.resources.status': '采集状态',
    'container.detail.storage.access': '访问',
    'container.detail.storage.destination': '挂载点',
    'container.detail.storage.mode': '模式',
    'container.detail.storage.source': '来源',
    'container.detail.storage.type': '类型',
    'container.detail.summary.identity': '容器',
    'container.detail.summary.network': '网络',
    'container.detail.summary.resources': '资源',
    'container.detail.tabs.config': '配置',
    'container.detail.tabs.health': '健康',
    'container.detail.tabs.logs': '日志',
    'container.detail.tabs.network': '网络',
    'container.detail.tabs.overview': '概览',
    'container.detail.tabs.raw': '原始 JSON',
    'container.detail.tabs.resources': '资源',
    'container.detail.tabs.storage': '挂载',
    'container.detail.title': '容器详情',
    'container.list.detail.command': '命令',
    'container.list.detail.entrypoint': '入口',
    'container.list.detail.inspectUpdatedAt': '详情更新时间',
    'container.list.detail.mountEmpty': '暂无挂载。',
    'container.list.detail.networkEmpty': '暂无网络信息。',
    'container.list.detail.workingDir': '工作目录',
    'container.list.eyebrow': '运维管理',
    'container.list.fields.createdAt': '创建时间',
    'container.list.fields.id': 'ID',
    'container.list.fields.image': '镜像',
    'container.list.fields.imageId': '镜像 ID',
    'container.list.fields.name': '名称',
    'container.list.fields.restartPolicy': '重启策略',
    'container.list.fields.startedAt': '启动时间',
    'container.list.fields.state': '状态码',
    'container.list.fields.status': '状态',
    'container.list.health.healthy': '健康',
    'container.list.health.unavailable': '健康未知',
    'container.list.logs.loadFailed': '容器日志加载失败。',
    'container.list.retry': '重试',
    'container.list.states.running': '运行中',
  }),
);

vi.mock('../../api/container', () => ({
  getContainer: apiMocks.getContainer,
  getContainerLogs: apiMocks.getContainerLogs,
}));

vi.mock('tdesign-vue-next/es/message', () => ({
  MessagePlugin: messageMocks,
}));

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    locale: 'zh-CN',
    t: (key: string, params?: Record<string, unknown>) =>
      (translations[key] ?? key).replace(/\{(\w+)\}/g, (_, name) => String(params?.[name] ?? `{${name}}`)),
  }),
}));

vi.mock('vue-router', () => ({
  useRoute: () => routeState,
  useRouter: () => routerMocks,
}));

vi.mock('@/shared/observability', async () => {
  const actual = await vi.importActual<typeof import('@/shared/observability')>('@/shared/observability');
  return {
    ...actual,
    formatLocaleDateTime: (value?: string | null) => value || '-',
  };
});

describe('container detail page', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    routeState.params.id = 'container-1';
    routeState.query.tab = 'config';
    Object.defineProperty(window, 'history', {
      configurable: true,
      value: { length: 1 },
    });
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: { writeText: vi.fn().mockResolvedValue(undefined) },
    });
    apiMocks.getContainer.mockResolvedValue(createContainerDetail());
    apiMocks.getContainerLogs.mockResolvedValue({
      id: 'container-1',
      lines: ['server started'],
      runtime: 'docker',
      stderr: true,
      stdout: true,
      tail: 200,
      timestamps: false,
      truncated: true,
    });
  });

  it('loads detail from the route id and renders dense tab content', async () => {
    const wrapper = mountPage();
    await flushPromises();

    expect(apiMocks.getContainer).toHaveBeenCalledWith('container-1');
    expect(wrapper.text()).toContain('容器详情 - graft-web');
    expect(wrapper.text()).toContain('graft/web:latest');
    expect(wrapper.text()).toContain('172.18.0.2');
    expect(wrapper.text()).toContain('21.8%');
    expect(wrapper.text()).toContain('31.25 GiB / 31.25 GiB');
    expect(wrapper.text()).toContain('8080:80/tcp');
    expect(wrapper.text()).toContain('环境变量');
    expect(wrapper.text()).toContain('APP_MODE');
    expect(wrapper.text()).toContain('production');
    expect(wrapper.text()).toContain('API_TOKEN');
    expect(wrapper.text()).toContain('已脱敏');
    expect(wrapper.text()).toContain('SECRET_KEY');
    expect(wrapper.text()).toContain('已隐藏');
    expect(wrapper.text()).toContain('敏感字段已脱敏，仅用于只读排查。');
    expect(wrapper.text()).toContain('container');
  });

  it('loads logs when the logs tab is selected and syncs the route query', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('[data-testid="tab-logs"]').trigger('click');
    await flushPromises();

    expect(routerMocks.replace).toHaveBeenCalledWith({
      params: { id: 'container-1' },
      query: { tab: 'logs' },
    });
    expect(apiMocks.getContainerLogs).toHaveBeenCalledWith('container-1', {
      tail: 200,
      since: undefined,
      stderr: true,
      stdout: true,
      timestamps: false,
    });
    expect(wrapper.text()).toContain('server started');
    expect(wrapper.text()).toContain('日志已按当前上限截断。');
  });

  it('copies only available environment values', async () => {
    const writeText = vi.spyOn(navigator.clipboard, 'writeText').mockResolvedValue(undefined);
    const wrapper = mountPage();
    await flushPromises();

    const copyButtons = wrapper.findAll('[data-testid="env-copy"]');
    expect(copyButtons).toHaveLength(1);

    await copyButtons[0].trigger('click');
    await flushPromises();

    expect(writeText).toHaveBeenCalledWith('production');
    expect(messageMocks.success).toHaveBeenCalledWith('内容已复制。');
  });

  it('falls back to the list route when there is no browser history', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('[data-testid="detail-back"]').trigger('click');
    await flushPromises();

    expect(routerMocks.push).toHaveBeenCalledWith({ name: 'ContainerList' });
  });

  it('uses shared log and JSON viewers instead of raw pre blocks', () => {
    expect(sourceText).toContain('<log-viewer');
    expect(sourceText).toContain('<json-viewer');
    expect(sourceText).not.toContain('container-detail-code');
  });
});

function createContainerDetail() {
  return {
    id: 'container-1',
    short_id: 'container-1',
    name: 'graft-web',
    names: ['graft-web'],
    image: 'graft/web:latest',
    image_id: 'sha256:1',
    labels: { 'com.docker.compose.project': 'graft' },
    status: 'Up 10 minutes',
    state: 'running',
    health: 'healthy',
    command: ['npm', 'run', 'serve'],
    entrypoint: ['docker-entrypoint.sh'],
    runtime: 'docker',
    created_at: '2026-06-14T01:00:00Z',
    started_at: '2026-06-14T01:05:00Z',
    inspect_updated_at: '2026-06-14T01:08:00Z',
    ports: [{ private_port: 80, public_port: 8080, type: 'tcp' }],
    mounts: [
      {
        type: 'bind',
        source: '/srv/graft',
        destination: '/app',
        mode: 'rw',
        read_only: false,
      },
    ],
    networks: [
      {
        name: 'bridge',
        ip_address: '172.18.0.2',
        gateway: '172.18.0.1',
      },
    ],
    environment_policy: 'masked',
    environment: [
      {
        key: 'APP_MODE',
        masked: false,
        sensitive: false,
        source: 'config',
        value: 'production',
      },
      {
        key: 'API_TOKEN',
        masked: true,
        sensitive: true,
        source: 'config',
      },
      {
        key: 'SECRET_KEY',
        policy: 'hidden',
        masked: false,
        sensitive: true,
        source: 'config',
      },
    ],
    resource: {
      available: true,
      stats_available: true,
      cpu_percent: 21.8,
      memory_limit_bytes: 33557250099.2,
      memory_percent: 100,
      memory_usage_bytes: 33557250099.2,
    },
    primary_ip: '172.18.0.2',
    network_summary: 'bridge',
    restart_count: 2,
    restart_policy: 'unless-stopped',
    can_start: false,
    can_stop: true,
    can_restart: true,
    can_remove: true,
    runtime_info: {
      runtime: 'docker',
      status: 'enabled',
      endpoint: 'unix:///var/run/docker.sock',
    },
  };
}

function mountPage() {
  return mount(ContainerDetailPage, {
    global: {
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
              h(
                'button',
                {
                  ...attrs,
                  disabled: Boolean(props.disabled),
                  'data-testid':
                    attrs['data-testid'] ??
                    (slots
                      .default?.()
                      .map((node) => String(node.children ?? ''))
                      .join('') === '返回'
                      ? 'detail-back'
                      : undefined),
                  onClick: () => emit('click'),
                },
                slots.default?.(),
              ),
        }),
        't-card': defineComponent({
          props: ['title'],
          setup:
            (props, { slots }) =>
            () =>
              h('section', [h('h2', String(props.title ?? '')), slots.default?.()]),
        }),
        't-descriptions': defineComponent({
          setup:
            (_, { slots }) =>
            () =>
              h('section', slots.default?.()),
        }),
        't-descriptions-item': defineComponent({
          props: ['label'],
          setup:
            (props, { slots }) =>
            () =>
              h('div', [h('strong', String(props.label ?? '')), slots.default?.()]),
        }),
        't-empty': defineComponent({
          props: ['description'],
          setup: (props) => () => h('div', String(props.description ?? '')),
        }),
        't-progress': defineComponent({
          props: ['percentage'],
          setup: (props) => () => h('span', `${String(props.percentage)}%`),
        }),
        't-statistic': defineComponent({
          props: ['value', 'unit'],
          setup: (props) => () => h('strong', `${String(props.value ?? '')}${String(props.unit ?? '')}`),
        }),
        't-input': defineComponent({
          setup: () => () => h('input'),
        }),
        't-loading': defineComponent({
          setup:
            (_, { slots }) =>
            () =>
              h('div', slots.default?.()),
        }),
        't-space': defineComponent({
          setup:
            (_, { slots }) =>
            () =>
              h('div', slots.default?.()),
        }),
        't-select': defineComponent({
          inheritAttrs: false,
          emits: ['change', 'update:value'],
          setup:
            (_, { emit }) =>
            () =>
              h(
                'select',
                {
                  onChange: (event: Event) => {
                    const value = Number((event.target as HTMLSelectElement).value);
                    emit('update:value', value);
                    emit('change', value);
                  },
                },
                [h('option', { value: 200 }, '200')],
              ),
        }),
        't-tab-panel': defineComponent({
          props: ['label', 'value'],
          setup:
            (props, { slots }) =>
            () =>
              h('section', [h('h3', String(props.label ?? props.value ?? '')), slots.default?.()]),
        }),
        't-switch': defineComponent({
          setup: () => () => h('button', 'switch'),
        }),
        't-tabs': defineComponent({
          props: ['value'],
          emits: ['change', 'update:value'],
          setup:
            (_props, { emit, slots }) =>
            () =>
              h('div', [
                h(
                  'button',
                  {
                    'data-testid': 'tab-logs',
                    onClick: () => {
                      emit('update:value', 'logs');
                      emit('change', 'logs');
                    },
                  },
                  'logs',
                ),
                slots.default?.(),
              ]),
        }),
        't-table': defineComponent({
          props: ['columns', 'data'],
          setup:
            (props, { slots }) =>
            () =>
              h('div', [
                ...(props.columns as Array<{ colKey: string; title: string }>).map((column) =>
                  h('strong', column.title),
                ),
                ...(props.data as Array<Record<string, unknown>>).map((row) =>
                  h('div', { key: String(row.name ?? row.key ?? row.destination) }, [
                    Object.values(row).map((value) => h('span', String(value ?? ''))),
                    slots.value?.({ row }),
                    slots.policy?.({ row }),
                    slots.operation?.({ row }),
                  ]),
                ),
              ]),
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
