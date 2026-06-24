import { flushPromises, mount } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, ref } from 'vue';

import { LOCALE } from '@/contracts/i18n/locales';

import ContainerListPage from './index.vue';

const apiMocks = vi.hoisted(() => ({
  batchContainerActions: vi.fn(),
  getContainer: vi.fn(),
  getContainerLogs: vi.fn(),
  getContainers: vi.fn(),
  removeContainer: vi.fn(),
  restartContainer: vi.fn(),
  startContainer: vi.fn(),
  stopContainer: vi.fn(),
}));

const dialogMocks = vi.hoisted(() => ({
  alert: vi.fn(),
  confirm: vi.fn(),
  instances: [] as Array<{ hide: ReturnType<typeof vi.fn>; setConfirmLoading: ReturnType<typeof vi.fn> }>,
}));

const messageMocks = vi.hoisted(() => ({
  error: vi.fn(),
  success: vi.fn(),
  warning: vi.fn(),
}));

const notifyMocks = vi.hoisted(() => ({
  warning: vi.fn(),
}));

const routerMocks = vi.hoisted(() => ({
  push: vi.fn(),
  resolve: vi.fn((target: { name?: string; params?: { id?: string }; query?: { tab?: string } }) => {
    const path = `/ops/containers/${target.params?.id ?? ''}`;
    const query = target.query ?? {};
    return {
      fullPath: query.tab ? `${path}?tab=${query.tab}` : path,
      meta: { keepAlive: false, pageKind: 'detail' },
      name: target.name,
      params: target.params ?? {},
      path,
      query,
    };
  }),
}));

const tabsRouterStoreMock = vi.hoisted(() => ({
  activeTabKey: '',
  appendTabRouterList: vi.fn(),
  tabRouters: [] as Array<{ path: string; fullPath?: string; tabKey?: string }>,
  setActiveTabKey: vi.fn((tabKey: string) => {
    tabsRouterStoreMock.activeTabKey = tabKey;
  }),
}));

const translations = vi.hoisted(
  (): Record<string, string> => ({
    'container.list.actionFailed': '容器操作失败。',
    'container.list.actionSuccess': '容器操作已提交。',
    'container.list.actions.cancel': '取消',
    'container.list.actions.confirm': '确认',
    'container.list.actions.confirmRestart': '确认重启容器 {name}？',
    'container.list.actions.confirmRestartTitle': '确认重启容器',
    'container.list.actions.confirmRemove': '确认删除容器 {name}？删除后不可恢复。',
    'container.list.actions.confirmRemoveRunning': '容器 {name} 正在运行。默认删除会被拒绝，如需强制删除必须显式勾选。',
    'container.list.actions.confirmRemoveTitle': '确认删除容器',
    'container.list.actions.confirmStart': '确认启动容器 {name}？',
    'container.list.actions.confirmStartTitle': '确认启动容器',
    'container.list.actions.confirmStop': '确认停止容器 {name}？',
    'container.list.actions.confirmStopTitle': '确认停止容器',
    'container.list.actions.copyId': '复制 ID',
    'container.list.actions.dangerousDisabled': '高危操作已禁用或当前状态不允许。',
    'container.list.actions.detail': '详情',
    'container.list.actions.forceRemove': '强制删除运行中容器',
    'container.list.actions.inspect': '检查',
    'container.list.actions.logs': '日志',
    'container.list.actions.more': '更多',
    'container.list.actions.remove': '删除',
    'container.list.actions.restart': '重启',
    'container.list.actions.start': '启动',
    'container.list.actions.stop': '停止',
    'container.list.actions.unavailable': '该操作当前不可用。',
    'container.list.actions.viewEnvironment': '查看环境变量',
    'container.list.actions.viewMounts': '查看挂载',
    'container.list.actions.viewNetworks': '查看网络',
    'container.detail.title': '容器详情',
    'container.list.actionModeEnabled': '操作已启用',
    'container.list.batch.cancelSelection': '取消选择',
    'container.list.batch.confirmRemove': '确认删除选中的 {count} 个容器？删除后不可恢复。',
    'container.list.batch.confirmRemoveRunning':
      '其中 {count} 个容器正在运行。默认删除会被拒绝，如需强制删除必须显式勾选。',
    'container.list.batch.confirmRemoveTitle': '确认批量删除',
    'container.list.batch.confirmRestart': '确认重启选中的 {count} 个容器？',
    'container.list.batch.confirmRestartTitle': '确认批量重启',
    'container.list.batch.confirmScope':
      '当前已选择 {selectedCount} 个容器，本次将处理 {actionableCount} 个可操作容器，跳过 {skippedCount} 个。',
    'container.list.batch.confirmStart': '确认启动选中的 {count} 个容器？',
    'container.list.batch.confirmStartTitle': '确认批量启动',
    'container.list.batch.confirmStop': '确认停止选中的 {count} 个容器？',
    'container.list.batch.confirmStopTitle': '确认批量停止',
    'container.list.batch.failed': '批量操作失败。',
    'container.list.batch.failureDetailTitle': '查看失败明细',
    'container.list.batch.noFailureDetail': '暂无失败明细。',
    'container.list.batch.noSelection': '请先选择容器。',
    'container.list.batch.partialTitle': '批量操作部分成功',
    'container.list.batch.remove': '批量删除',
    'container.list.batch.removeHint': '删除选中的 {count} 个容器。',
    'container.list.batch.restart': '批量重启',
    'container.list.batch.restartHint': '重启选中的 {count} 个容器。',
    'container.list.batch.selected': '已选择 {count} 个容器',
    'container.list.batch.skipInapplicable': '部分容器因当前状态或权限不适用，将被跳过。',
    'container.list.batch.skipSourceRestricted': '其中 {count} 个容器因来源策略不允许参与批量高危操作而被跳过。',
    'container.list.batch.start': '批量启动',
    'container.list.batch.startHint': '启动选中的 {count} 个容器。',
    'container.list.batch.stop': '批量停止',
    'container.list.batch.stopHint': '停止选中的 {count} 个容器。',
    'container.list.batch.success': '批量操作已完成，成功 {count} 个。',
    'container.list.clearFilters': '清除筛选',
    'container.list.columnSettings': '列设置',
    'container.list.columns.cpu': 'CPU',
    'container.list.columns.imageId': '镜像 ID',
    'container.list.columns.labels': '标签',
    'container.list.columns.memory': '内存',
    'container.list.columns.network': '网络 / IP',
    'container.list.columns.resource': '资源',
    'container.list.columns.runtimeStatus': '运行时 / 状态',
    'container.list.columns.createdAt': '创建时间',
    'container.list.columns.image': '镜像',
    'container.list.columns.name': '容器',
    'container.list.columns.operation': '操作',
    'container.list.columns.ports': '端口',
    'container.list.columns.restartPolicy': '重启策略',
    'container.list.columns.selection': '选择',
    'container.list.columns.source': '来源',
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
    'container.list.detail.environment': '环境变量',
    'container.list.detail.environmentUnavailable': '当前容器无法查看环境变量。',
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
    'container.list.filters.allOrchestrators': '全部来源',
    'container.list.filters.allHealth': '全部健康状态',
    'container.list.filters.health': '健康状态',
    'container.list.filters.orchestrator': '来源',
    'container.list.filters.sourceScopeKind': '来源范围',
    'container.list.filters.allSourceScopeKinds': '全部范围',
    'container.list.filters.sourceScopePlaceholder': '输入{kind}名称精确筛选',
    'container.list.filters.sourceScopePlaceholderDisabled': '先选择来源范围',
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
    'container.list.autoRefreshInterval5Seconds': '每 5 秒',
    'container.list.resourceUnavailable': '不可用',
    'container.list.unhealthyCount': '不健康 {count}',
    'container.list.readOnlyMode': '只读模式',
    'container.list.resetColumns': '恢复默认列',
    'container.list.retry': '重试',
    'container.list.runtimeContainers': '{running}/{total} 运行中',
    'container.list.runtimeDisabledHint': '请在系统配置中启用容器运行时访问后重试。',
    'container.list.runtimeLabel': '运行时',
    'container.list.runtimeUnavailable': '运行时不可用',
    'container.list.runningCount': '运行中 {count}',
    'container.list.stats.unavailable': 'N/A',
    'container.list.stats.unavailableReasonFallback': '资源统计暂不可用',
    'container.list.stats.cpuTooltip': 'CPU 使用率：{percent}',
    'container.list.stats.memoryTooltip': '内存：{usage} / {limit}，{percent}',
    'container.list.states.created': '已创建',
    'container.list.states.dead': '异常',
    'container.list.states.exited': '已退出',
    'container.list.states.paused': '已暂停',
    'container.list.states.removing': '移除中',
    'container.list.states.restarting': '重启中',
    'container.list.states.running': '运行中',
    'container.list.states.unknown': '未知',
    'container.list.orchestrators.compose': 'Compose',
    'container.list.orchestrators.kubernetes': 'Kubernetes',
    'container.list.orchestrators.standalone': '独立容器',
    'container.list.orchestrators.swarm': 'Swarm',
    'container.list.orchestrators.unknown': '未知来源',
    'container.list.sourceKinds.compose_project': '项目',
    'container.list.sourceKinds.compose_service': '服务',
    'container.list.sourceKinds.swarm_stack': 'Stack',
    'container.list.sourceKinds.swarm_task': '任务',
    'container.list.sourceKinds.kubernetes_namespace': '命名空间',
    'container.list.sourceKinds.kubernetes_pod': 'Pod',
    'container.list.sourceUnknownSummary': '未提供来源摘要',
    'container.list.actions.sourceRisk': '该容器来自 {source}，执行高危操作前请确认上层编排状态。',
    'container.list.health.healthy': '健康',
    'container.list.health.unhealthy': '异常',
    'container.list.health.starting': '启动中',
    'container.list.health.none': '无健康检查',
    'container.list.health.unavailable': '健康未知',
    'container.list.stoppedCount': '已停止 {count}',
    'container.list.tableHint': '数据来自当前配置的容器运行时。',
    'container.list.tableSummary': '共 {count} 个容器',
    'container.list.title': '容器管理',
    'container.list.totalCount': '总数 {count}',
    'ops.container.error.runtimeDisabled': '容器运行时访问未启用',
    'ops.container.error.runtimeUnavailable': '容器运行时连接不可用',
    'ops.container.action.start.completed': '容器启动操作已完成',
    'ops.container.action.stop.completed': '容器停止操作已完成',
    'ops.container.action.restart.completed': '容器重启操作已完成',
    'ops.container.action.remove.completed': '容器删除操作已完成',
  }),
);

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((promiseResolve, promiseReject) => {
    resolve = promiseResolve;
    reject = promiseReject;
  });
  return { promise, reject, resolve };
}

vi.mock('../../api/container', () => ({
  batchContainerActions: apiMocks.batchContainerActions,
  getContainer: apiMocks.getContainer,
  getContainerLogs: apiMocks.getContainerLogs,
  getContainers: apiMocks.getContainers,
  removeContainer: apiMocks.removeContainer,
  restartContainer: apiMocks.restartContainer,
  startContainer: apiMocks.startContainer,
  stopContainer: apiMocks.stopContainer,
}));

vi.mock('tdesign-vue-next/es/dialog', () => ({
  DialogPlugin: dialogMocks,
}));

vi.mock('tdesign-vue-next/es/message', () => ({
  MessagePlugin: messageMocks,
}));

vi.mock('tdesign-vue-next/es/notification', () => ({
  NotifyPlugin: notifyMocks,
}));

vi.mock('tdesign-icons-vue-next', () => ({
  RefreshIcon: defineComponent({ name: 'RefreshIcon', setup: () => () => h('span') }),
  SearchIcon: defineComponent({ name: 'SearchIcon', setup: () => () => h('span') }),
}));

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>();
  const locale = ref('zh-CN');
  return {
    ...actual,
    useI18n: () => ({
      locale,
      tm: (key: string) => translations[key] ?? key,
      te: (key: string) => key in translations,
      t: (key: string, params?: Record<string, unknown>) =>
        (translations[key] ?? key).replace(/\{(\w+)\}/g, (_, name) => String(params?.[name] ?? `{${name}}`)),
    }),
  };
});

vi.mock('vue-router', () => ({
  useRoute: () => ({
    path: '/ops/containers',
    fullPath: '/ops/containers',
  }),
  useRouter: () => routerMocks,
}));

vi.mock('@/store', () => ({
  useTabsRouterStore: () => tabsRouterStoreMock,
}));

vi.mock('@/shared/observability', async () => {
  const actual = await vi.importActual<typeof import('@/shared/observability')>('@/shared/observability');
  return {
    ...actual,
    formatLocaleDateTime: (value?: string | null) => value || '-',
  };
});

vi.mock('@/utils/route/title', () => ({
  localizeRouteTitleKey: (titleKey: string) => ({
    [LOCALE.ZH_CN]: translations[titleKey] ?? titleKey,
    [LOCALE.EN_US]: titleKey === 'container.detail.title' ? 'Container Detail' : (translations[titleKey] ?? titleKey),
  }),
}));

describe('container list page', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    tabsRouterStoreMock.activeTabKey = '/ops/containers';
    tabsRouterStoreMock.tabRouters = [
      {
        path: '/ops/containers',
        fullPath: '/ops/containers',
        tabKey: '/ops/containers',
      },
    ];
    dialogMocks.instances = [];
    dialogMocks.confirm.mockImplementation((_options) => {
      const instance = {
        hide: vi.fn(),
        setConfirmLoading: vi.fn(),
      };
      dialogMocks.instances.push(instance);
      return instance;
    });
    dialogMocks.alert.mockImplementation(() => ({
      destroy: vi.fn(),
      hide: vi.fn(),
      setConfirmLoading: vi.fn(),
      show: vi.fn(),
      update: vi.fn(),
    }));
    window.localStorage.clear();
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
      resource: {
        available: true,
        stats_available: true,
        cpu_percent: 21.8,
        memory_limit_bytes: 536870912,
        memory_percent: 50,
        memory_usage_bytes: 268435456,
      },
      primary_ip: '172.18.0.2',
      network_summary: 'bridge',
      orchestrator: {
        type: 'compose',
        managed: true,
        confidence: 'high',
        project: 'graft',
        service: 'web',
        group_scope_kind: 'compose_project',
        group_value: 'graft',
        group_display_name: 'graft',
        member_scope_kind: 'compose_service',
        member_value: 'web',
        member_display_name: 'web',
        warnings: [],
        action_level: 'allow',
        batch_action_allowed: true,
      },
      can_start: false,
      can_stop: true,
      can_restart: true,
      can_remove: true,
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
    apiMocks.startContainer.mockResolvedValue({
      action: 'start',
      id: 'container-2',
      runtime: 'first-adapter',
      message_key: 'ops.container.action.start.completed',
      result: 'completed',
      status_after: 'running',
    });
    apiMocks.stopContainer.mockResolvedValue({
      action: 'stop',
      id: 'container-1',
      runtime: 'first-adapter',
      message_key: 'ops.container.action.stop.completed',
      result: 'completed',
      status_after: 'exited',
    });
    apiMocks.restartContainer.mockResolvedValue({
      action: 'restart',
      id: 'container-1',
      runtime: 'first-adapter',
      message_key: 'ops.container.action.restart.completed',
      result: 'completed',
      status_after: 'running',
    });
    apiMocks.removeContainer.mockResolvedValue({
      action: 'remove',
      id: 'container-1',
      runtime: 'first-adapter',
      message_key: 'ops.container.action.remove.completed',
      result: 'completed',
      status_after: 'removed',
    });
    apiMocks.batchContainerActions.mockResolvedValue({
      failed_count: 0,
      items: [],
      success_count: 2,
      total: 2,
    });
  });

  afterEach(() => {
    vi.useRealTimers();
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
    expect(wrapper.text()).toContain('总数 25');
    expect(wrapper.text()).toContain('不健康 0');
    expect(wrapper.text()).toContain('操作已启用');
    expect(wrapper.text()).toContain('graft-web');
    expect(wrapper.text()).toContain('graft/web:latest');
    expect(wrapper.text()).toContain('项目');
    expect(wrapper.text()).toContain('graft');
    expect(wrapper.text()).toContain('服务');
    expect(wrapper.text()).toContain('web');
    expect(wrapper.text()).toContain('21.8%');
    expect(wrapper.text()).toContain('256.0 MiB');
    expect(wrapper.text()).toContain('N/A');
    expect(wrapper.text()).not.toContain('stats_not_collected');
    expect(wrapper.text()).toContain('8080->80/tcp');
    expect(wrapper.text()).toContain('+1');
    expect(wrapper.text()).toContain('运行中');
    expect(wrapper.text()).toContain('详情');
    expect(wrapper.text()).toContain('日志');
    expect(wrapper.text()).toContain('复制 ID');
    expect(wrapper.text()).toContain('检查');
    expect(wrapper.text()).toContain('查看挂载');
    expect(wrapper.text()).toContain('查看网络');
    expect(wrapper.text()).toContain('查看环境变量');
    expect(wrapper.findAll('[data-testid="container-action-start"]').length).toBeGreaterThan(0);
    expect(wrapper.find('[data-testid="container-action-stop"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="container-action-restart"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="container-action-remove"]').exists()).toBe(true);
    expect(wrapper.text()).toContain('第 1-20 条 / 共 25 条');
    expect(wrapper.text()).not.toContain('graft-extra-21');
    wrapper.unmount();
  });

  it('keeps cpu text above 100 percent while clamping progress width', async () => {
    apiMocks.getContainers.mockResolvedValueOnce({
      items: [
        {
          ...createContainerRows(1)[0],
          resource: {
            available: true,
            stats_available: true,
            cpu_percent: 628.6,
            memory_limit_bytes: 536870912,
            memory_percent: 50,
            memory_usage_bytes: 268435456,
          },
        },
      ],
      limit: 20,
      offset: 0,
      runtime: {
        api_version: '1.51',
        architecture: 'x86_64',
        containers_running: 1,
        containers_total: 1,
        endpoint: 'unix:///var/run/docker.sock',
        operating_system: 'Ubuntu 24.04.3 LTS',
        runtime: 'docker',
        server_version: '29.4.1',
        status: 'enabled',
      },
      summary: {
        error: 0,
        health_unavailable: 0,
        healthy: 1,
        running: 1,
        stopped: 0,
        total: 1,
        unhealthy: 0,
      },
      total: 1,
    });

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain('628.6%');
    expect(wrapper.findAll('[data-testid="resource-progress"]').some((node) => node.text() === '100')).toBe(true);
  });

  it('renders per-metric availability independently with N/A fallback', async () => {
    apiMocks.getContainers.mockResolvedValueOnce({
      items: [
        {
          ...createContainerRows(1)[0],
          resource: {
            available: true,
            stats_available: true,
            cpu_percent: undefined,
            memory_limit_bytes: 536870912,
            memory_percent: 25,
            memory_usage_bytes: 134217728,
          },
        },
      ],
      limit: 20,
      offset: 0,
      runtime: {
        api_version: '1.51',
        architecture: 'x86_64',
        containers_running: 1,
        containers_total: 1,
        endpoint: 'unix:///var/run/docker.sock',
        operating_system: 'Ubuntu 24.04.3 LTS',
        runtime: 'docker',
        server_version: '29.4.1',
        status: 'enabled',
      },
      summary: {
        error: 0,
        health_unavailable: 0,
        healthy: 1,
        running: 1,
        stopped: 0,
        total: 1,
        unhealthy: 0,
      },
      total: 1,
    });

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain('N/A');
    expect(wrapper.text()).toContain('128.0 MiB');
    expect(wrapper.text()).toContain('N/A / 25.0%');
  });

  it('replaces resource metrics with the latest list payload on refresh', async () => {
    apiMocks.getContainers
      .mockResolvedValueOnce({
        items: [
          {
            ...createContainerRows(1)[0],
            resource: {
              available: true,
              stats_available: true,
              cpu_percent: 21.8,
              memory_limit_bytes: 536870912,
              memory_percent: 50,
              memory_usage_bytes: 268435456,
            },
          },
        ],
        limit: 20,
        offset: 0,
        runtime: {
          api_version: '1.51',
          architecture: 'x86_64',
          containers_running: 1,
          containers_total: 1,
          endpoint: 'unix:///var/run/docker.sock',
          operating_system: 'Ubuntu 24.04.3 LTS',
          runtime: 'docker',
          server_version: '29.4.1',
          status: 'enabled',
        },
        summary: {
          error: 0,
          health_unavailable: 0,
          healthy: 1,
          running: 1,
          stopped: 0,
          total: 1,
          unhealthy: 0,
        },
        total: 1,
      })
      .mockResolvedValueOnce({
        items: [
          {
            ...createContainerRows(1)[0],
            resource: {
              available: true,
              stats_available: true,
              cpu_percent: undefined,
              memory_limit_bytes: 536870912,
              memory_percent: 25,
              memory_usage_bytes: 134217728,
            },
          },
        ],
        limit: 20,
        offset: 0,
        runtime: {
          api_version: '1.51',
          architecture: 'x86_64',
          containers_running: 1,
          containers_total: 1,
          endpoint: 'unix:///var/run/docker.sock',
          operating_system: 'Ubuntu 24.04.3 LTS',
          runtime: 'docker',
          server_version: '29.4.1',
          status: 'enabled',
        },
        summary: {
          error: 0,
          health_unavailable: 0,
          healthy: 1,
          running: 1,
          stopped: 0,
          total: 1,
          unhealthy: 0,
        },
        total: 1,
      });

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain('21.8%');
    expect(wrapper.text()).toContain('256.0 MiB');
    expect(wrapper.text()).toContain('21.8% / 50.0%');

    await wrapper.get('[data-testid="table-refresh"]').trigger('click');
    await flushPromises();

    expect(apiMocks.getContainers).toHaveBeenCalledTimes(2);
    expect(wrapper.text()).not.toContain('21.8%');
    expect(wrapper.text()).not.toContain('21.8% / 50.0%');
    expect(wrapper.text()).toContain('N/A');
    expect(wrapper.text()).toContain('128.0 MiB');
    expect(wrapper.text()).toContain('N/A / 25.0%');
  });

  it('does not render a list-page realtime toolbar', async () => {
    const wrapper = mountPage();
    await flushPromises();

    expect(apiMocks.getContainers).toHaveBeenCalledTimes(1);
    expect(wrapper.find('[data-testid="container-list-realtime-bar"]').exists()).toBe(false);
    wrapper.unmount();
  });

  it('does not stack concurrent manual refresh requests', async () => {
    vi.useFakeTimers();
    const pending = deferred<{
      items: ReturnType<typeof createContainerRows>;
      limit: number;
      offset: number;
      runtime: {
        runtime: string;
        status: string;
        endpoint: string;
        containers_running: number;
        containers_total: number;
      };
      summary: {
        total: number;
        running: number;
        stopped: number;
        error: number;
        healthy: number;
        unhealthy: number;
        health_unavailable: number;
      };
      total: number;
    }>();

    apiMocks.getContainers.mockReset();
    apiMocks.getContainers.mockReturnValueOnce(pending.promise);

    const wrapper = mountPage();
    await flushPromises();
    expect(apiMocks.getContainers).toHaveBeenCalledTimes(1);

    await wrapper.get('[data-testid="table-refresh"]').trigger('click');
    await wrapper.get('[data-testid="table-refresh"]').trigger('click');
    expect(apiMocks.getContainers).toHaveBeenCalledTimes(1);

    pending.resolve({
      items: createContainerRows(20, 1),
      limit: 20,
      offset: 0,
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
    });
    apiMocks.getContainers.mockResolvedValue({
      items: createContainerRows(20, 1),
      limit: 20,
      offset: 0,
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
    });
    await flushPromises();

    expect(apiMocks.getContainers).toHaveBeenCalledTimes(1);
    await vi.runAllTimersAsync();
    wrapper.unmount();
  });

  it('navigates safe read actions to the non-overview detail tabs', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('[data-testid="container-action-logs"]').trigger('click');
    await flushPromises();

    expect(tabsRouterStoreMock.appendTabRouterList).toHaveBeenLastCalledWith(
      expect.objectContaining({
        fullPath: '/ops/containers/container-1?tab=logs',
        path: '/ops/containers/container-1',
        query: { tab: 'logs' },
        tabKey: '/ops/containers/container-1',
      }),
    );
    expect(routerMocks.push).toHaveBeenCalledWith({
      name: 'ContainerDetailIndex',
      params: { id: 'container-1' },
      query: { tab: 'logs' },
    });

    await wrapper.get('[data-testid="container-action-view-mounts"]').trigger('click');
    await flushPromises();
    expect(tabsRouterStoreMock.appendTabRouterList).toHaveBeenLastCalledWith(
      expect.objectContaining({
        fullPath: '/ops/containers/container-1?tab=storage',
        path: '/ops/containers/container-1',
        query: { tab: 'storage' },
      }),
    );
    expect(routerMocks.push).toHaveBeenCalledWith({
      name: 'ContainerDetailIndex',
      params: { id: 'container-1' },
      query: { tab: 'storage' },
    });

    await wrapper.get('[data-testid="container-action-view-networks"]').trigger('click');
    await flushPromises();
    expect(tabsRouterStoreMock.appendTabRouterList).toHaveBeenLastCalledWith(
      expect.objectContaining({
        fullPath: '/ops/containers/container-1?tab=network',
        path: '/ops/containers/container-1',
        query: { tab: 'network' },
      }),
    );
    expect(routerMocks.push).toHaveBeenCalledWith({
      name: 'ContainerDetailIndex',
      params: { id: 'container-1' },
      query: { tab: 'network' },
    });

    await wrapper.get('[data-testid="container-action-view-env"]').trigger('click');
    await flushPromises();
    expect(tabsRouterStoreMock.appendTabRouterList).toHaveBeenLastCalledWith(
      expect.objectContaining({
        fullPath: '/ops/containers/container-1?tab=config',
        path: '/ops/containers/container-1',
        query: { tab: 'config' },
      }),
    );
    expect(routerMocks.push).toHaveBeenCalledWith({
      name: 'ContainerDetailIndex',
      params: { id: 'container-1' },
      query: { tab: 'config' },
    });

    expect(apiMocks.getContainer).not.toHaveBeenCalled();
    expect(apiMocks.getContainerLogs).not.toHaveBeenCalled();
  });

  it('uses optional column settings without showing started time and restart policy by default', async () => {
    const wrapper = mountPage();
    await flushPromises();

    const table = wrapper.get('[data-testid="container-table"]');
    const columnKeys = JSON.parse(table.attributes('data-column-keys') ?? '[]');

    expect(columnKeys).toEqual([
      'row-select',
      'state',
      'name',
      'image',
      'source',
      'cpu',
      'memory',
      'ports',
      'network',
      'runtime_status',
      'created_at',
      'operation',
    ]);
    expect(columnKeys).not.toContain('started_at');
    expect(columnKeys).not.toContain('restart_policy');
    expect(columnKeys).not.toContain('resource');

    const drawer = wrapper.get('[data-testid="container-column-drawer"]');
    expect(JSON.parse(drawer.attributes('data-default-selected-keys') ?? '[]')).toEqual([
      'row-select',
      'state',
      'name',
      'image',
      'source',
      'cpu',
      'memory',
      'ports',
      'network',
      'runtime_status',
      'created_at',
      'operation',
    ]);
    expect(JSON.parse(drawer.attributes('data-disabled-keys') ?? '[]')).toEqual([
      'row-select',
      'state',
      'name',
      'operation',
    ]);
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
      orchestrator: undefined,
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

  it('applies source quick filters for group and member entry points', async () => {
    const wrapper = mountPage();
    await flushPromises();

    const groupFilters = wrapper.findAll('[data-testid="container-source-group-filter"]');
    expect(groupFilters.length).toBeGreaterThan(0);
    await groupFilters[0].trigger('click');
    await flushPromises();

    expect(apiMocks.getContainers).toHaveBeenLastCalledWith({
      health: undefined,
      keyword: undefined,
      limit: 20,
      offset: 0,
      orchestrator: 'compose',
      source_scope: 'graft',
      source_scope_kind: 'compose_project',
      state: undefined,
    });

    const memberFilters = wrapper.findAll('[data-testid="container-source-member-filter"]');
    expect(memberFilters.length).toBeGreaterThan(0);
    await memberFilters[0].trigger('click');
    await flushPromises();

    expect(apiMocks.getContainers).toHaveBeenLastCalledWith({
      health: undefined,
      keyword: undefined,
      limit: 20,
      offset: 0,
      orchestrator: 'compose',
      source_scope: 'web',
      source_scope_kind: 'compose_service',
      state: undefined,
    });
  });

  it('supports explicit toolbar source scope filtering for compose projects', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('[data-testid="container-filter-orchestrator"]').setValue('compose');
    expect((wrapper.get('[data-testid="container-filter-source-scope-kind"]').element as HTMLSelectElement).value).toBe(
      'compose_project',
    );
    await wrapper.get('[data-testid="container-filter-source-scope"]').setValue('graft');
    await wrapper.get('[data-testid="container-filter-apply"]').trigger('click');
    await flushPromises();

    expect(apiMocks.getContainers).toHaveBeenLastCalledWith({
      health: undefined,
      keyword: undefined,
      limit: 20,
      offset: 0,
      orchestrator: 'compose',
      source_scope: 'graft',
      source_scope_kind: 'compose_project',
      state: undefined,
    });
  });

  it('keeps the submitted source scope query stable until filters are applied again', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('[data-testid="container-filter-orchestrator"]').setValue('compose');
    await wrapper.get('[data-testid="container-filter-source-scope"]').setValue('graft');
    await wrapper.get('[data-testid="container-filter-apply"]').trigger('click');
    await flushPromises();

    expect(apiMocks.getContainers).toHaveBeenLastCalledWith({
      health: undefined,
      keyword: undefined,
      limit: 20,
      offset: 0,
      orchestrator: 'compose',
      source_scope: 'graft',
      source_scope_kind: 'compose_project',
      state: undefined,
    });

    await wrapper.get('[data-testid="container-filter-source-scope"]').setValue('draft-change');
    await wrapper.get('[data-testid="table-refresh"]').trigger('click');
    await flushPromises();

    expect(apiMocks.getContainers).toHaveBeenLastCalledWith({
      health: undefined,
      keyword: undefined,
      limit: 20,
      offset: 0,
      orchestrator: 'compose',
      source_scope: 'graft',
      source_scope_kind: 'compose_project',
      state: undefined,
    });

    await wrapper.get('[data-testid="container-filter-apply"]').trigger('click');
    await flushPromises();

    expect(apiMocks.getContainers).toHaveBeenLastCalledWith({
      health: undefined,
      keyword: undefined,
      limit: 20,
      offset: 0,
      orchestrator: 'compose',
      source_scope: 'draft-change',
      source_scope_kind: 'compose_project',
      state: undefined,
    });
  });

  it('clears incompatible toolbar source scope kinds when orchestrator changes', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('[data-testid="container-filter-source-scope-kind"]').setValue('compose_project');
    await wrapper.get('[data-testid="container-filter-source-scope"]').setValue('graft');
    await wrapper.get('[data-testid="container-filter-orchestrator"]').setValue('swarm');
    expect((wrapper.get('[data-testid="container-filter-source-scope-kind"]').element as HTMLSelectElement).value).toBe(
      'swarm_stack',
    );
    await wrapper.get('[data-testid="container-filter-apply"]').trigger('click');
    await flushPromises();

    expect((wrapper.get('[data-testid="container-filter-source-scope"]').element as HTMLInputElement).value).toBe('');
    expect(apiMocks.getContainers).toHaveBeenLastCalledWith({
      health: undefined,
      keyword: undefined,
      limit: 20,
      offset: 0,
      orchestrator: 'swarm',
      state: undefined,
    });
  });

  it('builds dangerous actions from row availability and submits confirmed runtime actions', async () => {
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain('操作已启用');
    expect(wrapper.findAll('[data-testid="container-action-start"]').length).toBeGreaterThan(0);
    expect(wrapper.find('[data-testid="container-action-stop"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="container-action-restart"]').exists()).toBe(true);

    await wrapper.get('[data-testid="container-action-stop"]').trigger('click');
    await flushPromises();

    expect(dialogMocks.confirm).toHaveBeenCalledWith(
      expect.objectContaining({
        header: '确认停止容器',
        theme: 'danger',
      }),
    );
    expect(renderDialogBodyText(dialogMocks.confirm.mock.calls.at(-1)?.[0].body)).toContain('确认停止容器 graft-web？');
    await dialogMocks.confirm.mock.calls.at(-1)?.[0].onConfirm();
    await flushPromises();

    expect(apiMocks.stopContainer).toHaveBeenCalledWith('container-1');
    expect(messageMocks.success).toHaveBeenCalledWith('容器停止操作已完成');
    expect(apiMocks.getContainers).toHaveBeenCalledTimes(2);
    expect(dialogMocks.instances.at(-1)?.hide).toHaveBeenCalled();
  });

  it('keeps dangerous action confirmation dialogs idempotent while one is open', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('[data-testid="container-action-stop"]').trigger('click');
    await wrapper.get('[data-testid="container-action-stop"]').trigger('click');
    await flushPromises();

    expect(dialogMocks.confirm).toHaveBeenCalledTimes(1);

    await dialogMocks.confirm.mock.calls.at(-1)?.[0].onCancel();
    await flushPromises();

    expect(dialogMocks.instances.at(-1)?.hide).toHaveBeenCalled();

    await wrapper.get('[data-testid="container-action-stop"]').trigger('click');
    await flushPromises();

    expect(dialogMocks.confirm).toHaveBeenCalledTimes(2);
  });

  it('keeps dangerous action events fail-closed when row flags are false', async () => {
    const wrapper = mountPage();
    await flushPromises();

    const vm = wrapper.vm as unknown as {
      handleRowAction: (action: string, row: ReturnType<typeof createContainerRows>[number]) => void;
    };
    vm.handleRowAction('start', createContainerRows(1)[0]);
    await flushPromises();

    expect(messageMocks.warning).toHaveBeenCalledWith('高危操作已禁用或当前状态不允许。');
    expect(apiMocks.startContainer).not.toHaveBeenCalled();
    expect(dialogMocks.confirm).not.toHaveBeenCalled();
  });

  it('confirms running container removal with force selection defaulting to false', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('[data-testid="container-action-remove"]').trigger('click');
    await flushPromises();

    expect(dialogMocks.confirm).toHaveBeenCalledWith(
      expect.objectContaining({
        confirmBtn: '删除',
        header: '确认删除容器',
        theme: 'danger',
      }),
    );
    expect(renderDialogBodyText(dialogMocks.confirm.mock.calls.at(-1)?.[0].body)).toContain('容器 graft-web 正在运行');

    dialogMocks.confirm.mock.calls.at(-1)?.[0].onConfirm();
    await flushPromises();

    expect(apiMocks.removeContainer).toHaveBeenCalledWith('container-1', { force: false });
    expect(messageMocks.success).toHaveBeenCalledWith('容器删除操作已完成');
  });

  it('submits only actionable mixed selections and reports partial failures', async () => {
    apiMocks.batchContainerActions.mockResolvedValueOnce({
      failed_count: 1,
      items: [
        {
          action: 'restart',
          id: 'container-1',
          name: 'graft-web',
          success: true,
        },
        {
          action: 'restart',
          id: 'container-2',
          message: 'runtime rejected restart',
          success: false,
        },
      ],
      success_count: 1,
      total: 2,
    });
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('[data-testid="container-table-select-first-two"]').trigger('click');
    await flushPromises();

    expect(wrapper.text()).toContain('已选择 2 个容器');
    expect(wrapper.get('[data-testid="container-batch-stop"]').attributes('disabled')).toBeUndefined();
    expect(wrapper.get('[data-testid="container-batch-remove"]').attributes('disabled')).toBeUndefined();

    await wrapper.get('[data-testid="container-batch-restart"]').trigger('click');
    await flushPromises();

    expect(dialogMocks.confirm).toHaveBeenCalledWith(
      expect.objectContaining({
        header: '确认批量重启',
        theme: 'danger',
      }),
    );
    expect(renderDialogBodyText(dialogMocks.confirm.mock.calls.at(-1)?.[0].body)).toContain(
      '当前已选择 2 个容器，本次将处理 2 个可操作容器，跳过 0 个。',
    );

    await dialogMocks.confirm.mock.calls.at(-1)?.[0].onConfirm();
    await flushPromises();

    expect(dialogMocks.instances.at(-1)?.setConfirmLoading).toHaveBeenNthCalledWith(1, true);
    expect(apiMocks.batchContainerActions).toHaveBeenCalledWith({
      action: 'restart',
      force: false,
      ids: ['container-1', 'container-2'],
    });
    expect(notifyMocks.warning).toHaveBeenCalledWith(
      expect.objectContaining({
        content: expect.stringContaining('container-2: runtime rejected restart'),
        title: '批量操作部分成功',
      }),
    );
    expect(dialogMocks.instances.at(-1)?.setConfirmLoading).toHaveBeenLastCalledWith(false);
  });

  it('enables batch actions when any selected row is actionable and skips inapplicable rows', async () => {
    apiMocks.batchContainerActions.mockResolvedValue({
      action: 'start',
      failed_count: 0,
      items: [{ action: 'start', id: 'container-2', name: 'graft-extra-2', success: true }],
      success_count: 1,
      total: 1,
    });
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('[data-testid="container-table-select-first-two"]').trigger('click');
    await flushPromises();

    expect(wrapper.get('[data-testid="container-batch-start"]').attributes('disabled')).toBeUndefined();
    expect(wrapper.get('[data-testid="container-batch-stop"]').attributes('disabled')).toBeUndefined();

    await wrapper.get('[data-testid="container-batch-start"]').trigger('click');
    await flushPromises();

    const startDialog = dialogMocks.confirm.mock.calls.at(-1)?.[0];
    expect(startDialog).toEqual(
      expect.objectContaining({
        header: '确认批量启动',
        theme: 'warning',
      }),
    );
    expect(renderDialogBodyText(startDialog?.body)).toContain('确认启动选中的 1 个容器？');
    expect(renderDialogBodyText(startDialog?.body)).toContain(
      '当前已选择 2 个容器，本次将处理 1 个可操作容器，跳过 1 个。',
    );
    expect(renderDialogBodyText(startDialog?.body)).toContain('部分容器因当前状态或权限不适用，将被跳过。');

    await startDialog?.onConfirm();
    await flushPromises();

    expect(apiMocks.batchContainerActions).toHaveBeenCalledWith({
      action: 'start',
      force: false,
      ids: ['container-2'],
    });
    expect(messageMocks.success).toHaveBeenCalledWith('批量操作已完成，成功 1 个。');

    await wrapper.get('[data-testid="container-batch-stop"]').trigger('click');
    await flushPromises();
    await dialogMocks.confirm.mock.calls.at(-1)?.[0].onCancel();

    expect(dialogMocks.instances.at(-1)?.hide).toHaveBeenCalled();
    expect(apiMocks.batchContainerActions).toHaveBeenCalledTimes(1);
  });

  it('keeps batch confirmation dialogs idempotent while one is open', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('[data-testid="container-table-select-first-two"]').trigger('click');
    await flushPromises();

    await wrapper.get('[data-testid="container-batch-start"]').trigger('click');
    await wrapper.get('[data-testid="container-batch-start"]').trigger('click');
    await flushPromises();

    expect(dialogMocks.confirm).toHaveBeenCalledTimes(1);

    await dialogMocks.confirm.mock.calls.at(-1)?.[0].onClose();
    await flushPromises();

    expect(dialogMocks.instances.at(-1)?.hide).toHaveBeenCalled();

    await wrapper.get('[data-testid="container-batch-start"]').trigger('click');
    await flushPromises();

    expect(dialogMocks.confirm).toHaveBeenCalledTimes(2);
  });

  it('navigates inspect action to the overview tab without loading details in the list', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('[data-testid="container-action-inspect"]').trigger('click');
    await flushPromises();

    expect(tabsRouterStoreMock.appendTabRouterList).toHaveBeenCalledWith(
      expect.objectContaining({
        fullPath: '/ops/containers/container-1?tab=overview',
        path: '/ops/containers/container-1',
        query: { tab: 'overview' },
        tabKey: '/ops/containers/container-1',
      }),
    );
    expect(routerMocks.push).toHaveBeenCalledWith({
      name: 'ContainerDetailIndex',
      params: { id: 'container-1' },
      query: { tab: 'overview' },
    });
    expect(apiMocks.getContainer).not.toHaveBeenCalled();
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

  it('localizes resource stats error keys instead of rendering raw keys', async () => {
    apiMocks.getContainers.mockResolvedValue({
      items: [
        {
          ...createContainerRows(1)[0],
          resource: {
            available: false,
            stats_available: false,
            stats_error_key: 'ops.container.error.runtimeUnavailable',
          },
        },
      ],
      limit: 20,
      offset: 0,
      runtime: {
        runtime: 'first-adapter',
        status: 'enabled',
        endpoint: 'unix:///var/run/docker.sock',
        containers_running: 1,
        containers_total: 1,
      },
      summary: {
        total: 1,
        running: 1,
        stopped: 0,
        error: 0,
        healthy: 0,
        unhealthy: 0,
        health_unavailable: 1,
      },
      total: 1,
    });

    const wrapper = mountPage();
    await flushPromises();

    const tooltipContents = wrapper
      .findAll('[data-tooltip-content]')
      .map((tooltip) => tooltip.attributes('data-tooltip-content'));

    expect(tooltipContents).toContain('容器运行时连接不可用');
    expect(wrapper.text()).not.toContain('ops.container.error.runtimeUnavailable');
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

function renderDialogBodyText(body: unknown) {
  if (typeof body === 'string') {
    return body;
  }
  if (typeof body === 'function') {
    const node = body();
    return JSON.stringify(node);
  }
  return String(body ?? '');
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
      resource:
        ordinal === 1
          ? {
              available: true,
              stats_available: true,
              cpu_percent: 21.8,
              memory_limit_bytes: 536870912,
              memory_percent: 50,
              memory_usage_bytes: 268435456,
            }
          : {
              available: true,
              stats_available: true,
              cpu_percent: undefined,
              memory_limit_bytes: 536870912,
              memory_percent: 50,
              memory_usage_bytes: 134217728,
            },
      compose_project: ordinal === 1 ? 'graft' : undefined,
      compose_service: ordinal === 1 ? 'web' : undefined,
      orchestrator:
        ordinal === 1
          ? {
              type: 'compose' as const,
              managed: true,
              confidence: 'high' as const,
              project: 'graft',
              service: 'web',
              group_scope_kind: 'compose_project' as const,
              group_value: 'graft',
              group_display_name: 'graft',
              member_scope_kind: 'compose_service' as const,
              member_value: 'web',
              member_display_name: 'web',
              warnings: [],
              action_level: 'allow' as const,
              batch_action_allowed: true,
            }
          : undefined,
      can_start: ordinal !== 1,
      can_stop: ordinal === 1,
      can_restart: true,
      can_remove: true,
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
              h('section', [slots.head?.(), slots.toolbar?.(), slots.batch?.(), slots.default?.(), slots.footer?.()]),
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
        't-dropdown': defineComponent({
          props: ['options'],
          emits: ['click'],
          setup:
            (props, { emit, slots }) =>
            () =>
              h('div', [
                slots.default?.(),
                ...(props.options as Array<{ content?: string; testId?: string; value?: string }>).map((option) =>
                  h(
                    'button',
                    {
                      disabled: Boolean((option as { disabled?: boolean }).disabled),
                      'data-testid': option.testId,
                      onClick: (event: MouseEvent) => emit('click', option, { e: event }),
                    },
                    option.content,
                  ),
                ),
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
          props: ['modelValue', 'disabled'],
          emits: ['update:modelValue', 'enter'],
          setup:
            (props, { attrs, emit }) =>
            () =>
              h('input', {
                ...attrs,
                disabled: Boolean(props.disabled),
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
        't-progress': defineComponent({
          props: ['percentage'],
          setup: (props) => () => h('span', { 'data-testid': 'resource-progress' }, String(props.percentage ?? 0)),
        }),
        't-select': defineComponent({
          props: ['modelValue', 'disabled'],
          emits: ['update:modelValue'],
          setup:
            (props, { attrs, emit, slots }) =>
            () =>
              h(
                'select',
                {
                  ...attrs,
                  disabled: Boolean(props.disabled),
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
          props: ['columns', 'data', 'selectedRowKeys', 'size'],
          emits: ['select-change'],
          setup:
            (props, { emit, slots }) =>
            () =>
              h(
                'div',
                {
                  'data-column-keys': JSON.stringify(
                    (props.columns as Array<{ colKey: string }> | undefined)?.map((column) => column.colKey) ?? [],
                  ),
                  'data-size': props.size,
                  'data-selected-row-keys': JSON.stringify(props.selectedRowKeys ?? []),
                  'data-testid': 'container-table',
                },
                [
                  h(
                    'button',
                    {
                      'data-testid': 'container-table-select-first-two',
                      onClick: () => emit('select-change', ['container-1', 'container-2']),
                    },
                    'select',
                  ),
                  (props.data as Array<Record<string, unknown>>).length
                    ? (props.data as Array<Record<string, unknown>>).map((row) =>
                        h('div', { 'data-testid': 'container-table-row', key: String(row.id) }, [
                          slots.state?.({ row }),
                          slots.name?.({ row }),
                          slots.image?.({ row }),
                          slots.source?.({ row }),
                          slots.cpu?.({ row }),
                          slots.memory?.({ row }),
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
                ],
              ),
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
              h('span', { 'data-tooltip-content': String(props.content ?? '') }, slots.default?.()),
        }),
      },
    },
  });
}
