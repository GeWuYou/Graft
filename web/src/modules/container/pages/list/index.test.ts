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
  warning: vi.fn(),
}));

const permissionMocks = vi.hoisted(() => ({
  hasPermission: vi.fn(),
}));

const translations = vi.hoisted(
  (): Record<string, string> => ({
    'container.list.actionFailed': '容器操作失败。',
    'container.list.actionSuccess': '容器操作已提交。',
    'container.list.actions.cancel': '取消',
    'container.list.actions.confirm': '确认',
    'container.list.actions.confirmRestart': '确认重启容器 {name}？',
    'container.list.actions.confirmStart': '确认启动容器 {name}？',
    'container.list.actions.confirmStop': '确认停止容器 {name}？',
    'container.list.actions.copyId': '复制 ID',
    'container.list.actions.detail': '详情',
    'container.list.actions.logs': '日志',
    'container.list.actions.more': '更多',
    'container.list.actions.restart': '重启',
    'container.list.actions.start': '启动',
    'container.list.actions.stop': '停止',
    'container.list.actions.unavailable': '该操作当前不可用。',
    'container.list.clearFilters': '清除筛选',
    'container.list.columnSettings': '列设置',
    'container.list.columns.imageId': '镜像 ID',
    'container.list.columns.labels': '标签',
    'container.list.columns.network': '网络 / IP',
    'container.list.columns.resource': '资源',
    'container.list.columns.runtimeStatus': '运行时 / 状态',
    'container.list.columns.createdAt': '创建时间',
    'container.list.columns.image': '镜像',
    'container.list.columns.name': '容器',
    'container.list.columns.operation': '操作',
    'container.list.columns.ports': '端口',
    'container.list.columns.restartPolicy': '重启策略',
    'container.list.columns.startedAt': '启动时间',
    'container.list.columns.status': '状态',
    'container.list.copyError': '日志复制失败。',
    'container.list.copyIdError': '容器 ID 复制失败。',
    'container.list.copyIdSuccess': '容器 ID 已复制。',
    'container.list.copySuccess': '日志已复制。',
    'container.list.compactDensity': '紧凑密度',
    'container.list.defaultDensity': '默认密度',
    'container.list.description': '查看容器状态。',
    'container.list.detail.command': '命令',
    'container.list.detail.entrypoint': '入口',
    'container.list.detail.identity': '基础信息',
    'container.list.detail.inspectUpdatedAt': '详情更新时间',
    'container.list.detail.loadFailed': '容器详情加载失败。',
    'container.list.detail.metadata': '标签与元数据',
    'container.list.detail.metadataEmpty': '暂无标签或元数据。',
    'container.list.detail.mountEmpty': '暂无挂载。',
    'container.list.detail.mounts': '挂载',
    'container.list.detail.networkPorts': '网络与端口',
    'container.list.detail.networkEmpty': '暂无网络信息。',
    'container.list.detail.networks': '网络',
    'container.list.detail.portEmpty': '暂无端口映射。',
    'container.list.detail.ports': '端口',
    'container.list.detail.rawJson': '原始详情 JSON',
    'container.list.detail.runtime': '运行时',
    'container.list.detail.state': '状态与生命周期',
    'container.list.detail.title': '容器详情',
    'container.list.detail.workingDir': '工作目录',
    'container.list.emptyDescription': '当前容器运行时未返回容器。',
    'container.list.emptyFilteredDescription': '没有符合筛选条件的容器。',
    'container.list.emptyTitle': '暂无容器',
    'container.list.errorCount': '异常 {count}',
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
    'container.list.filters.allHealth': '全部健康状态',
    'container.list.filters.health': '健康状态',
    'container.list.filters.query': '查询',
    'container.list.filters.reset': '重置',
    'container.list.filters.searchPlaceholder': '搜索名称、镜像、ID 或端口',
    'container.list.filters.status': '容器状态',
    'container.list.loadFailed': '容器列表加载失败。',
    'container.list.labelCount': '{count} 个标签',
    'container.list.logs.autoRefresh': '自动刷新',
    'container.list.logs.autoRefreshStatus': '每 {seconds} 秒自动刷新',
    'container.list.logs.copy': '复制',
    'container.list.logs.emptyTitle': '暂无日志',
    'container.list.logs.empty': '暂无日志。',
    'container.list.logs.enabled': '启用',
    'container.list.logs.errorEmpty': '修复错误后可重试加载日志。',
    'container.list.logs.lastLoadedAt': '上次加载：{time}',
    'container.list.logs.loadFailed': '容器日志加载失败。',
    'container.list.logs.notLoaded': '尚未加载日志',
    'container.list.logs.refresh': '刷新日志',
    'container.list.logs.since': '起始时间或时长',
    'container.list.logs.sincePlaceholder': '例如 10m、1h 或 RFC3339',
    'container.list.logs.stderr': 'stderr',
    'container.list.logs.stdout': 'stdout',
    'container.list.logs.tail': '行数',
    'container.list.logs.timestamps': '时间戳',
    'container.list.logs.title': '容器日志',
    'container.list.logs.truncated': '日志已按当前上限截断。',
    'container.list.morePorts': '+{count}',
    'container.list.pagination.empty': '暂无记录',
    'container.list.pagination.summary': '第 {start}-{end} 条 / 共 {total} 条',
    'container.list.refresh': '刷新',
    'container.list.resourceUnavailable': '不可用',
    'container.list.resetColumns': '恢复默认列',
    'container.list.retry': '重试',
    'container.list.runtimeContainers': '{running}/{total} 运行中',
    'container.list.runtimeDisabledHint': '请在系统配置中启用容器运行时访问后重试。',
    'container.list.runtimeLabel': '运行时',
    'container.list.runtimeUnavailable': '运行时不可用',
    'container.list.runningCount': '运行中 {count}',
    'container.list.states.created': '已创建',
    'container.list.states.dead': '异常',
    'container.list.states.exited': '已退出',
    'container.list.states.paused': '已暂停',
    'container.list.states.removing': '移除中',
    'container.list.states.restarting': '重启中',
    'container.list.states.running': '运行中',
    'container.list.states.unknown': '未知',
    'container.list.health.healthy': '健康',
    'container.list.health.unhealthy': '异常',
    'container.list.health.starting': '启动中',
    'container.list.health.none': '无健康检查',
    'container.list.health.unavailable': '健康未知',
    'container.list.stoppedCount': '已停止 {count}',
    'container.list.tableHint': '数据来自当前配置的容器运行时。',
    'container.list.tableSummary': '共 {count} 个容器',
    'container.list.title': '容器管理',
    'ops.container.error.runtimeDisabled': '容器运行时访问未启用',
    'ops.container.error.runtimeUnavailable': '容器运行时连接不可用',
    'ops.container.action.start.completed': '容器启动操作已完成',
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

vi.mock('@/store', () => ({
  usePermissionStore: () => permissionMocks,
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
    window.localStorage.clear();
    vi.spyOn(window, 'confirm').mockReturnValue(true);
    permissionMocks.hasPermission.mockReturnValue(true);
    apiMocks.getContainers.mockImplementation(async (query) => ({
      items: createContainerRows(query?.offset === 20 ? 5 : 20, query?.offset === 20 ? 21 : 1),
      limit: query?.limit ?? 20,
      offset: query?.offset ?? 0,
      runtime: {
        runtime: 'first-adapter',
        status: 'enabled',
        endpoint: 'unix:///var/run/docker.sock',
        containers_running: 1,
        containers_total: 25,
      },
      summary: {
        total: 25,
        running: 1,
        stopped: 24,
        error: 0,
        healthy: 1,
        unhealthy: 0,
        health_unavailable: 24,
      },
      total: 25,
    }));
    apiMocks.getContainer.mockResolvedValue({
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
      runtime: 'first-adapter',
      created_at: '2026-06-14T01:00:00Z',
      started_at: '2026-06-14T01:05:00Z',
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
      resource: { available: false, unavailable_reason: 'stats_not_collected' },
      primary_ip: '172.18.0.2',
      network_summary: 'bridge',
      can_start: false,
      can_stop: true,
      can_restart: true,
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
    apiMocks.runContainerAction.mockResolvedValue({
      action: 'start',
      id: 'container-2',
      message_key: 'ops.container.action.start.completed',
      message: 'started',
      runtime: 'first-adapter',
      state: 'running',
    });
  });

  it('loads and renders container rows with required operation buttons', async () => {
    const wrapper = mountPage();
    await flushPromises();

    expect(apiMocks.getContainers).toHaveBeenCalledTimes(1);
    expect(apiMocks.getContainers).toHaveBeenCalledWith({
      health: undefined,
      keyword: undefined,
      limit: 20,
      offset: 0,
      state: undefined,
    });
    expect(wrapper.text()).toContain('容器管理');
    expect(wrapper.text()).toContain('graft-web');
    expect(wrapper.text()).toContain('graft/web:latest');
    expect(wrapper.text()).toContain('8080->80/tcp');
    expect(wrapper.text()).toContain('+1');
    expect(wrapper.text()).toContain('运行中');
    expect(wrapper.text()).toContain('详情');
    expect(wrapper.text()).toContain('日志');
    expect(wrapper.text()).toContain('启动');
    expect(wrapper.text()).toContain('停止');
    expect(wrapper.text()).toContain('重启');
    expect(wrapper.text()).toContain('第 1-20 条 / 共 25 条');
    expect(wrapper.text()).not.toContain('graft-extra-21');
  });

  it('opens detail and log drawers through module API actions', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('[data-testid="container-action-detail"]').trigger('click');
    await flushPromises();

    expect(apiMocks.getContainer).toHaveBeenCalledWith('container-1');
    expect(wrapper.get('[data-testid="td-drawer-容器详情"]').attributes('data-size')).toBe('960px');
    expect(wrapper.text()).toContain('容器详情');
    expect(wrapper.text()).toContain('状态与生命周期');
    expect(wrapper.text()).toContain('网络与端口');
    expect(wrapper.text()).toContain('标签与元数据');
    expect(wrapper.text()).toContain('npm run serve');
    expect(wrapper.text()).toContain('docker-entrypoint.sh');
    expect(wrapper.text()).toContain('172.18.0.2');
    expect(wrapper.text()).toContain('/app');
    expect(wrapper.text()).toContain('com.docker.compose.project=graft');
    expect(wrapper.text()).toContain('"id": "container-1"');

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

  it('copies container id from the detail drawer context', async () => {
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: { writeText: vi.fn().mockResolvedValue(undefined) },
    });
    const writeText = vi.spyOn(navigator.clipboard, 'writeText').mockResolvedValue(undefined);
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('[data-testid="container-action-detail"]').trigger('click');
    await flushPromises();

    await wrapper
      .findAll('button')
      .find((button) => button.text() === '复制 ID')
      ?.trigger('click');
    await flushPromises();

    expect(writeText).toHaveBeenCalledWith('container-1');
    expect(messageMocks.success).toHaveBeenCalledWith('容器 ID 已复制。');
  });

  it('auto-refreshes logs only after the logs drawer is opened and enabled', async () => {
    vi.useFakeTimers();
    const wrapper = mountPage();
    await flushPromises();

    expect(apiMocks.getContainerLogs).not.toHaveBeenCalled();

    await wrapper.get('[data-testid="container-action-logs"]').trigger('click');
    await flushPromises();

    expect(apiMocks.getContainerLogs).toHaveBeenCalledTimes(1);

    const autoRefreshCheckbox = wrapper.findAll('input[type="checkbox"]').at(3);
    expect(autoRefreshCheckbox).toBeTruthy();
    await autoRefreshCheckbox?.setValue(true);

    vi.advanceTimersByTime(10_000);
    await flushPromises();

    expect(apiMocks.getContainerLogs).toHaveBeenCalledTimes(2);

    vi.useRealTimers();
  });

  it('uses optional column settings without showing started time and restart policy by default', async () => {
    const wrapper = mountPage();
    await flushPromises();

    const table = wrapper.get('[data-testid="container-table"]');
    const columnKeys = JSON.parse(table.attributes('data-column-keys') ?? '[]');

    expect(columnKeys).toEqual([
      'state',
      'name',
      'image',
      'ports',
      'network',
      'resource',
      'runtime_status',
      'created_at',
      'operation',
    ]);
    expect(columnKeys).not.toContain('started_at');
    expect(columnKeys).not.toContain('restart_policy');

    const drawer = wrapper.get('[data-testid="container-column-drawer"]');
    expect(JSON.parse(drawer.attributes('data-default-selected-keys') ?? '[]')).toEqual([
      'state',
      'name',
      'image',
      'ports',
      'network',
      'resource',
      'runtime_status',
      'created_at',
      'operation',
    ]);
    expect(JSON.parse(drawer.attributes('data-disabled-keys') ?? '[]')).toEqual(['state', 'name', 'operation']);
  });

  it('supports server pagination and table density controls', async () => {
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.findAll('[data-testid="container-table-row"]')).toHaveLength(20);

    await wrapper.get('[data-testid="pagination-next"]').trigger('click');
    await flushPromises();

    expect(apiMocks.getContainers).toHaveBeenLastCalledWith({
      health: undefined,
      keyword: undefined,
      limit: 20,
      offset: 20,
      state: undefined,
    });
    expect(wrapper.text()).toContain('第 21-25 条 / 共 25 条');
    expect(wrapper.findAll('[data-testid="container-table-row"]')).toHaveLength(5);
    expect(wrapper.text()).toContain('graft-extra-21');

    expect(wrapper.get('[data-testid="container-table"]').attributes('data-size')).toBe('medium');
    await wrapper.get('[data-testid="table-density"]').trigger('click');
    await flushPromises();
    expect(wrapper.get('[data-testid="container-table"]').attributes('data-size')).toBe('small');
  });

  it('confirms start stop and restart from the more action menu', async () => {
    const wrapper = mountPage();
    await flushPromises();

    const startAction = wrapper
      .findAll('[data-testid="container-action-start"]')
      .find((button) => button.attributes('disabled') === undefined);
    expect(startAction).toBeTruthy();

    await startAction?.trigger('click');
    await flushPromises();

    expect(window.confirm).toHaveBeenCalledWith('确认启动容器 graft-extra-2？');
    expect(apiMocks.runContainerAction).toHaveBeenCalledWith('start', 'container-2');
    expect(messageMocks.success).toHaveBeenCalledWith('容器启动操作已完成');
  });

  it('does not confirm or call the API when server action availability disables the action', async () => {
    const wrapper = mountPage();
    await flushPromises();

    const disabledStartAction = wrapper.get('[data-testid="container-action-start"]');
    expect(disabledStartAction.attributes('disabled')).toBeDefined();
    await disabledStartAction.trigger('click');
    await flushPromises();

    expect(window.confirm).not.toHaveBeenCalled();
    expect(apiMocks.runContainerAction).not.toHaveBeenCalled();
    expect(messageMocks.warning).not.toHaveBeenCalled();
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

function createContainerRows(count: number, startOrdinal = 1) {
  return Array.from({ length: count }, (_, index) => {
    const ordinal = startOrdinal + index;
    return {
      id: `container-${ordinal}`,
      short_id: `container-${ordinal}`,
      name: ordinal === 1 ? 'graft-web' : `graft-extra-${ordinal}`,
      names: [ordinal === 1 ? 'graft-web' : `graft-extra-${ordinal}`],
      image: ordinal === 1 ? 'graft/web:latest' : 'graft/worker:latest',
      image_id: `sha256:${ordinal}`,
      labels: ordinal === 1 ? { 'com.docker.compose.project': 'graft' } : {},
      ports:
        ordinal === 1
          ? [
              { private_port: 80, public_port: 8080, type: 'tcp' as const },
              { private_port: 443, public_port: 8443, type: 'tcp' as const },
              { private_port: 9000, type: 'tcp' as const },
            ]
          : [],
      restart_policy: 'unless-stopped',
      runtime: 'first-adapter',
      state: ordinal === 1 ? ('running' as const) : ('exited' as const),
      health: ordinal === 1 ? ('healthy' as const) : ('unavailable' as const),
      status: ordinal === 1 ? 'Up 10 minutes' : 'Exited',
      created_at: '2026-06-14T01:00:00Z',
      started_at: ordinal === 1 ? '2026-06-14T01:05:00Z' : undefined,
      primary_ip: ordinal === 1 ? '172.18.0.2' : undefined,
      network_summary: ordinal === 1 ? 'bridge' : undefined,
      networks:
        ordinal === 1
          ? [
              {
                name: 'bridge',
                ip_address: '172.18.0.2',
              },
            ]
          : [],
      resource: { available: false, unavailable_reason: 'stats_not_collected' },
      compose_project: ordinal === 1 ? 'graft' : undefined,
      compose_service: ordinal === 1 ? 'web' : undefined,
      can_start: ordinal !== 1,
      can_stop: ordinal === 1,
      can_restart: true,
    };
  });
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
              h('section', [slots.head?.(), slots.toolbar?.(), slots.default?.(), slots.footer?.()]),
        }),
        'advanced-query-column-drawer': defineComponent({
          name: 'AdvancedQueryColumnDrawerStub',
          props: ['columns', 'defaultSelectedKeys', 'disabledKeys', 'resetLabel', 'selectedKeys', 'title', 'visible'],
          setup: (props) => () =>
            h('aside', {
              'data-default-selected-keys': JSON.stringify(props.defaultSelectedKeys ?? []),
              'data-disabled-keys': JSON.stringify(props.disabledKeys ?? []),
              'data-testid': 'container-column-drawer',
            }),
        }),
        'management-table-pagination': defineComponent({
          props: ['summary'],
          setup:
            (props, { slots }) =>
            () =>
              h('footer', [String(props.summary ?? ''), slots.default?.()]),
        }),
        'table-action-menu': defineComponent({
          props: ['actions'],
          emits: ['action'],
          setup:
            (props, { emit }) =>
            () =>
              h(
                'div',
                (props.actions as Array<{ disabled?: boolean; label: string; testId?: string; value: string }>).map(
                  (action) =>
                    h(
                      'button',
                      {
                        disabled: Boolean(action.disabled),
                        'data-testid': action.testId,
                        onClick: () => emit('action', action.value),
                      },
                      translations[action.label] ?? action.label,
                    ),
                ),
              ),
        }),
        'table-view-toolbar': defineComponent({
          props: ['columnSettingsLabel', 'densityLabel', 'refreshLabel'],
          emits: ['column-settings', 'density', 'refresh'],
          setup:
            (props, { emit }) =>
            () =>
              h('div', [
                props.refreshLabel
                  ? h('button', { 'data-testid': 'table-refresh', onClick: () => emit('refresh') }, props.refreshLabel)
                  : null,
                props.columnSettingsLabel
                  ? h(
                      'button',
                      { 'data-testid': 'table-column-settings', onClick: () => emit('column-settings') },
                      props.columnSettingsLabel,
                    )
                  : null,
                props.densityLabel
                  ? h('button', { 'data-testid': 'table-density', onClick: () => emit('density') }, props.densityLabel)
                  : null,
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
        't-collapse': defineComponent({
          setup:
            (_, { slots }) =>
            () =>
              h('section', slots.default?.()),
        }),
        't-collapse-panel': defineComponent({
          props: ['header'],
          setup:
            (props, { slots }) =>
            () =>
              h('section', [h('h3', String(props.header ?? '')), slots.default?.()]),
        }),
        't-drawer': defineComponent({
          props: ['visible', 'header', 'size'],
          setup:
            (props, { slots }) =>
            () =>
              props.visible
                ? h(
                    'aside',
                    {
                      'data-size': String(props.size ?? ''),
                      'data-testid': `td-drawer-${String(props.header ?? '')}`,
                    },
                    [h('h2', String(props.header ?? '')), slots.default?.()],
                  )
                : null,
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
          props: ['modelValue', 'value', 'disabled'],
          emits: ['update:modelValue', 'update:value'],
          setup:
            (props, { emit }) =>
            () =>
              h('input', {
                disabled: Boolean(props.disabled),
                type: 'number',
                value: props.value ?? props.modelValue,
                onInput: (event: Event) => {
                  const value = Number((event.target as HTMLInputElement).value);
                  emit('update:modelValue', value);
                  emit('update:value', value);
                },
              }),
        }),
        't-loading': defineComponent({
          setup:
            (_, { slots }) =>
            () =>
              h('div', slots.default?.()),
        }),
        't-pagination': defineComponent({
          props: ['current', 'pageSize', 'total'],
          emits: ['change', 'update:current', 'update:pageSize'],
          setup:
            (props, { emit }) =>
            () =>
              h(
                'button',
                {
                  'data-testid': 'pagination-next',
                  onClick: () => {
                    const current = Number(props.current ?? 1) + 1;
                    emit('update:current', current);
                    emit('change', { current, pageSize: Number(props.pageSize ?? 20) });
                  },
                },
                `next ${String(props.total ?? 0)}`,
              ),
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
          props: ['columns', 'data', 'size'],
          setup:
            (props, { slots }) =>
            () =>
              h(
                'div',
                {
                  'data-column-keys': JSON.stringify(
                    (props.columns as Array<{ colKey: string }> | undefined)?.map((column) => column.colKey) ?? [],
                  ),
                  'data-size': props.size,
                  'data-testid': 'container-table',
                },
                (props.data as Array<Record<string, unknown>>).length
                  ? (props.data as Array<Record<string, unknown>>).map((row) =>
                      h('div', { 'data-testid': 'container-table-row', key: String(row.id) }, [
                        slots.state?.({ row }),
                        slots.name?.({ row }),
                        slots.image?.({ row }),
                        slots.ports?.({ row }),
                        slots.network?.({ row }),
                        slots.resource?.({ row }),
                        slots.runtime_status?.({ row }),
                        slots.image_id?.({ row }),
                        slots.labels?.({ row }),
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
        't-tooltip': defineComponent({
          setup:
            (_, { slots }) =>
            () =>
              h('span', slots.default?.()),
        }),
      },
    },
  });
}
