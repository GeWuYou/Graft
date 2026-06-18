// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { readFileSync } from 'node:fs';
import { join } from 'node:path';

import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import type { ContainerMountUsage } from '../../types/container';
import ContainerDetailPage from './index.vue';

const sourceText = readFileSync(join(process.cwd(), 'src/modules/container/pages/detail/index.vue'), 'utf8');
const overviewPanelSourceText = readFileSync(
  join(process.cwd(), 'src/modules/container/pages/detail/components/ContainerOverviewPanel.vue'),
  'utf8',
);

const apiMocks = vi.hoisted(() => ({
  getContainer: vi.fn(),
  getContainerLogs: vi.fn(),
  getContainerMountUsage: vi.fn(),
  postContainerMountUsageRefresh: vi.fn(),
}));

const messageMocks = vi.hoisted(() => ({
  error: vi.fn(),
  success: vi.fn(),
  warning: vi.fn(),
}));

const routerMocks = vi.hoisted(() => ({
  back: vi.fn(),
  push: vi.fn(),
  replace: vi.fn(),
}));

const tabStoreState = vi.hoisted(() => ({
  tabRouterList: [
    {
      path: '/ops/containers/container-1',
      tabKey: '/ops/containers/container-1',
      title: { 'zh-CN': '容器详情', 'en-US': 'Container Detail' },
    },
  ] as Array<{
    fullPath?: string;
    path: string;
    tabKey: string;
    title: { 'zh-CN': string; 'en-US': string };
  }>,
}));

const routeState = vi.hoisted(
  (): {
    route: {
      fullPath: string;
      path: string;
      params: { id: string };
      query: { name?: string; tab?: string };
    };
  } => ({
    route: {
      fullPath: '/ops/containers/container-1?tab=config',
      path: '/ops/containers/container-1',
      params: { id: 'container-1' },
      query: { tab: 'config' },
    },
  }),
);

function deferred<T>() {
  let resolve: (value: T) => void = () => undefined;
  let reject: (reason?: unknown) => void = () => undefined;
  const promise = new Promise<T>((promiseResolve, promiseReject) => {
    resolve = promiseResolve;
    reject = promiseReject;
  });
  return { promise, reject, resolve };
}

const translations = vi.hoisted(
  (): Record<string, string> => ({
    'container.detail.back': '返回',
    'container.detail.config.envName': '变量名',
    'container.detail.config.envPolicy': '安全策略',
    'container.detail.config.envValue': '值',
    'container.detail.config.environment': '环境变量',
    'container.detail.config.environmentCount': '{count} 项',
    'container.detail.config.environmentEmptyTitle': '暂无环境变量',
    'container.detail.config.environmentFilterEmptyDescription': '请调整关键字或安全策略筛选。',
    'container.detail.config.environmentFilterEmptyTitle': '未找到匹配的环境变量',
    'container.detail.config.environmentUnavailable': '当前容器无法查看环境变量。',
    'container.detail.config.copyEnvFile': '复制 .env',
    'container.detail.config.copyEnvSuccess': '已复制 .env 内容',
    'container.detail.config.copyRuntimeSuccess': '已复制配置值',
    'container.detail.config.copyRuntimeValue': '复制配置值',
    'container.detail.config.copyVariableValue': '复制变量值',
    'container.detail.config.copyVariableValueSuccess': '已复制变量值',
    'container.detail.config.hiddenValue': '[已隐藏]',
    'container.detail.config.maskedValue': '[已脱敏]',
    'container.detail.config.policy.sensitive': '敏感',
    'container.detail.config.policyFilter.all': '安全策略：全部',
    'container.detail.config.policy.hidden': '隐藏',
    'container.detail.config.policy.masked': '脱敏',
    'container.detail.config.policy.plain': '明文',
    'container.detail.config.policy.unknown': '未知',
    'container.detail.config.runtimeTitle': '运行配置',
    'container.detail.config.searchPlaceholder': '搜索变量名 / 值',
    'container.detail.copy': '复制',
    'container.detail.copyError': '内容复制失败。',
    'container.detail.copySuccess': '内容已复制。',
    'container.detail.description': '查看容器运行时详情、资源、日志、配置、网络和挂载信息。',
    'container.detail.autoRefresh': '自动刷新',
    'container.detail.autoRefreshOff': '关闭',
    'container.detail.autoRefreshPaused': '已暂停',
    'container.detail.autoRefreshSeconds': '{seconds} 秒',
    'container.detail.pauseAutoRefresh': '暂停自动刷新',
    'container.detail.refreshSuccess': '容器详情已刷新',
    'container.detail.resumeAutoRefresh': '恢复自动刷新',
    'container.detail.empty': '暂无容器详情。',
    'container.detail.health.boolean.no': '否',
    'container.detail.health.boolean.yes': '是',
    'container.detail.health.checkCommand': '检查命令',
    'container.detail.health.checkResult': '健康检查结果',
    'container.detail.health.currentStatus': '当前状态',
    'container.detail.health.description.healthy': '健康检查通过，容器运行中。',
    'container.detail.health.description.noHealthcheck': '容器运行中，但未配置 Docker Healthcheck。',
    'container.detail.health.description.notRunning': '容器当前未处于运行状态。',
    'container.detail.health.description.starting': '健康检查仍在启动观察期。',
    'container.detail.health.description.unavailable': '健康状态暂不可用。',
    'container.detail.health.description.unhealthy': '健康检查失败，需要查看最近输出。',
    'container.detail.health.diagnosis.healthy': '健康',
    'container.detail.health.diagnosis.noHealthcheck': '未配置健康检查',
    'container.detail.health.diagnosis.notRunning': '未运行',
    'container.detail.health.diagnosis.starting': '启动中',
    'container.detail.health.diagnosis.unhealthy': '异常',
    'container.detail.health.diagnosisTitle': '健康诊断',
    'container.detail.health.exitCode': 'Exit Code',
    'container.detail.health.exitCodeValue': 'Exit Code: {code}',
    'container.detail.health.healthcheck': 'Healthcheck',
    'container.detail.health.healthcheckStatus.failed': '失败',
    'container.detail.health.healthcheckStatus.passed': '通过',
    'container.detail.health.healthcheckStatus.starting': '启动中',
    'container.detail.health.healthcheckStatus.unavailable': '不可用',
    'container.detail.health.healthcheckStatus.unconfigured': '未配置',
    'container.detail.health.healthcheckUnavailableAlert':
      '当前容器未提供 Docker Healthcheck 明细，仅能根据运行状态判断基础可用性。',
    'container.detail.health.healthcheckUnavailableEmpty': '未配置 Docker Healthcheck',
    'container.detail.health.lastCheck': '最近检查',
    'container.detail.health.lastCheckValue': '最近检查：{time}',
    'container.detail.health.lastExitCode': '最近退出码',
    'container.detail.health.lastOutput': '最近输出',
    'container.detail.health.noOutput': '无异常输出',
    'container.detail.health.noRecentCheck': '暂无最近检查时间',
    'container.detail.health.oomKilled': 'OOMKilled',
    'container.detail.health.recentCheck': '最近检查',
    'container.detail.health.restartAbnormal': '已记录 {count} 次重启',
    'container.detail.health.restartCount': '重启次数',
    'container.detail.health.restartCountValue': '{count} 次',
    'container.detail.health.restartNormal': '无异常重启',
    'container.detail.health.restartUnknown': '重启次数未提供',
    'container.detail.health.status': '健康状态',
    'container.detail.health.stability': '运行稳定性',
    'container.detail.health.stabilityStatus.exit': '异常退出',
    'container.detail.health.stabilityStatus.oom': '发生 OOM',
    'container.detail.health.stabilityStatus.restart': '存在重启',
    'container.detail.health.stabilityStatus.stable': '状态稳定',
    'container.detail.health.updatedFromHealthcheck': '来自最近健康检查',
    'container.detail.health.updatedFromInspect': '来自详情更新时间',
    'container.detail.health.uptime': '已运行',
    'container.detail.health.uptimeHoursMinutes': '{hours} 小时 {minutes} 分钟',
    'container.detail.health.uptimeMinutes': '{minutes} 分钟',
    'container.detail.inspectUpdatedAt': '详情更新时间',
    'container.detail.logs.empty': '暂无日志。',
    'container.detail.logs.allLevels': '全部',
    'container.detail.logs.basicInfo': '基础信息',
    'container.detail.logs.collapseDetail': '收起详情',
    'container.detail.logs.copyJson': '复制 JSON',
    'container.detail.logs.copyLine': '复制本行',
    'container.detail.logs.detailTitle': '日志详情',
    'container.detail.logs.download': '下载',
    'container.detail.logs.level': '级别',
    'container.detail.logs.levelFilter': '级别',
    'container.detail.logs.matchCount': '{count} 个匹配',
    'container.detail.logs.metadata': 'Metadata',
    'container.detail.logs.importantFields': '关键字段',
    'container.detail.logs.message': '完整消息',
    'container.detail.logs.raw': '原始日志',
    'container.detail.logs.copyMessage': '复制消息',
    'container.detail.logs.refresh': '刷新日志',
    'container.detail.logs.refreshScroll': '跟随底部',
    'container.detail.logs.refreshScrollTooltip': '刷新日志后自动滚动到底部',
    'container.detail.logs.searchPlaceholder': '搜索日志内容',
    'container.detail.logs.source': '来源',
    'container.detail.logs.time': '时间',
    'container.detail.logs.truncated': '日志已按当前上限截断。',
    'container.detail.logs.viewDetail': '查看详情',
    'container.detail.logs.wrap': '自动换行',
    'container.detail.missingId': '缺少容器标识。',
    'container.detail.network.gateway': '网关',
    'container.detail.network.aliasDns': '网络别名 / DNS',
    'container.detail.network.aliases': 'Aliases',
    'container.detail.network.connections': '网络连接',
    'container.detail.network.containerPort': '容器端口',
    'container.detail.network.dns': 'DNS',
    'container.detail.network.endpointId': 'Endpoint',
    'container.detail.network.hostname': 'Hostname',
    'container.detail.network.hostPort': '主机端口',
    'container.detail.network.ipAddress': 'IP 地址',
    'container.detail.network.listenAddress': '监听地址',
    'container.detail.network.macAddress': 'MAC 地址',
    'container.detail.network.mapping': '映射',
    'container.detail.network.name': '网络',
    'container.detail.network.networkId': 'Network ID',
    'container.detail.network.noData': '暂无数据',
    'container.detail.network.allInterfaces': '全部地址',
    'container.detail.network.aliasDnsEmpty': '暂无额外网络别名或自定义 DNS 配置',
    'container.detail.network.internalOnlyFull': '未发布到宿主机，仅容器网络内部可访问',
    'container.detail.network.internalOnly': '仅容器网络内部可访问',
    'container.detail.network.notPublished': '未发布到宿主机',
    'container.detail.network.noPublicPorts': '无公开端口',
    'container.detail.network.publishedMapping': '宿主机 {hostPort} → 容器 {privatePort}/{protocol}',
    'container.detail.network.publishedToHost': '已发布到宿主机',
    'container.detail.network.portCount': '{count} 个端口映射',
    'container.detail.network.ports': '端口映射',
    'container.detail.network.primaryIp': '主 IP',
    'container.detail.network.protocol': '协议',
    'container.detail.network.subnet': '子网',
    'container.detail.network.summary': '网络摘要',
    'container.detail.operation': '操作',
    'container.detail.overview.basicInfo': '基础信息',
    'container.detail.overview.fields.containerId': '容器 ID',
    'container.detail.overview.fields.createdAt': '创建时间',
    'container.detail.overview.fields.health': '健康检查',
    'container.detail.overview.fields.image': '镜像',
    'container.detail.overview.fields.imageId': '镜像 ID',
    'container.detail.overview.fields.name': '名称',
    'container.detail.overview.fields.networkMode': '网络模式',
    'container.detail.overview.fields.networkName': '网络名称',
    'container.detail.overview.fields.runtime': '运行时',
    'container.detail.overview.fields.startedAt': '启动时间',
    'container.detail.overview.fields.state': '状态码',
    'container.detail.overview.fields.status': '状态',
    'container.detail.overview.fields.updatedAt': '更新时间',
    'container.detail.overview.resourceNetwork': '资源与网络',
    'container.detail.overview.runtimeInfo': '运行信息',
    'container.detail.raw.description': '敏感字段已脱敏，仅用于只读排查。',
    'container.detail.raw.empty': '暂无原始 JSON。',
    'container.detail.raw.error': '原始 JSON 无法格式化。',
    'container.detail.raw.root': 'container',
    'container.detail.raw.source': '源码视图',
    'container.detail.raw.title': '原始 JSON',
    'container.detail.raw.tree': '树形视图',
    'container.detail.refresh': '刷新',
    'container.detail.refreshNow': '立即刷新',
    'container.detail.refreshTooltip':
      '刷新容器详情、资源、网络、挂载列表和挂载用量最近一次统计结果，不重新扫描挂载空间。',
    'container.detail.resources.available': '已采集',
    'container.detail.resources.collectedAt': '采集时间',
    'container.detail.resources.cpu': 'CPU',
    'container.detail.resources.cpuDetails': 'CPU 详细信息',
    'container.detail.resources.cpuKernelTime': '内核态时间',
    'container.detail.resources.cpuLimit': 'CPU 限制',
    'container.detail.resources.cpuLimitWithOnline': 'CPU 限制 / 在线 CPU',
    'container.detail.resources.cpuPercent': 'CPU 使用率',
    'container.detail.resources.cpuUserTime': '用户态时间',
    'container.detail.resources.cpuUsageInKernelmode': '内核模式',
    'container.detail.resources.cpuUsageInUsermode': '用户模式',
    'container.detail.resources.cpuUsageRate': 'CPU 使用率',
    'container.detail.resources.currentSnapshot': '当前快照',
    'container.detail.resources.dashboard': '资源仪表盘',
    'container.detail.resources.detail': '资源明细',
    'container.detail.resources.detailedMetrics': '详细指标',
    'container.detail.resources.memory': '内存',
    'container.detail.resources.memoryActiveFile': '活跃文件',
    'container.detail.resources.memoryCache': '缓存',
    'container.detail.resources.memoryDetails': '内存详细信息',
    'container.detail.resources.memoryInactiveFile': '非活跃文件',
    'container.detail.resources.memoryLimit': '内存上限',
    'container.detail.resources.memoryPercent': '内存百分比',
    'container.detail.resources.memoryPgfault': '页面错误',
    'container.detail.resources.memoryPgmajfault': '主页面错误',
    'container.detail.resources.memoryRss': 'RSS',
    'container.detail.resources.memoryUsage': '内存使用',
    'container.detail.resources.memoryUsageRate': '内存使用率',
    'container.detail.resources.networkIo': '网络 I/O',
    'container.detail.resources.noData': '无数据',
    'container.detail.resources.notCollected': '暂未采集',
    'container.detail.resources.onlineCpus': '在线 CPU',
    'container.detail.resources.pidsCurrent': 'PIDs 当前数',
    'container.detail.resources.pidsLimit': 'PIDs 限制',
    'container.detail.resources.processInfo': '进程信息',
    'container.detail.resources.rxBytes': '接收',
    'container.detail.resources.rxDropped': '接收丢包',
    'container.detail.resources.rxErrors': '接收错误',
    'container.detail.resources.rxPackets': '接收包',
    'container.detail.resources.status': '采集状态',
    'container.detail.resources.systemCpuUsage': '系统 CPU',
    'container.detail.resources.systemCpuTime': '系统 CPU 时间',
    'container.detail.resources.throttlingCount': 'Throttling 次数',
    'container.detail.resources.throttlingInactiveHint': '未发生 CPU throttling',
    'container.detail.resources.throttlingPeriods': 'CPU throttling 周期',
    'container.detail.resources.throttlingSignalHint': '存在 CPU throttling，可能受限',
    'container.detail.resources.throttlingTime': 'Throttling 时间',
    'container.detail.resources.throttlingThrottledPeriods': 'CPU throttling 次数',
    'container.detail.resources.throttlingThrottledTime': 'CPU throttling 时间',
    'container.detail.resources.totalCpuUsage': 'CPU 总用量',
    'container.detail.resources.totalCpuTime': '累计 CPU 时间',
    'container.detail.resources.txBytes': '发送',
    'container.detail.resources.txDropped': '发送丢包',
    'container.detail.resources.txErrors': '发送错误',
    'container.detail.resources.txPackets': '发送包',
    'container.detail.resources.unavailable': '未采集',
    'container.detail.storage.access': '访问',
    'container.detail.storage.accessLabels.readOnly': '只读',
    'container.detail.storage.accessLabels.readWrite': '读写',
    'container.detail.storage.basicInfo': '基础信息',
    'container.detail.storage.destination': '挂载点',
    'container.detail.storage.emptyDescription': '该容器未配置额外挂载卷',
    'container.detail.storage.emptyTitle': '暂无挂载',
    'container.detail.storage.errorMessage': '挂载用量无法测量',
    'container.detail.storage.measuredAt': '测量时间 {time}',
    'container.detail.storage.mode': '模式',
    'container.detail.storage.notMeasuredMessage': '暂未测量挂载用量',
    'container.detail.storage.pendingMessage': '正在统计宿主机来源路径占用，请稍候',
    'container.detail.storage.pendingSize': '统计中...',
    'container.detail.storage.refreshError': '挂载用量刷新失败。',
    'container.detail.storage.refreshMount': '重新统计',
    'container.detail.storage.refreshMountTooltip': '重新统计当前挂载来源路径占用，不刷新整个容器详情。',
    'container.detail.storage.refreshPending': '统计中...',
    'container.detail.storage.refreshSuccess': '挂载用量已统计',
    'container.detail.storage.retryUsage': '重试',
    'container.detail.storage.syncFailed': '挂载用量同步失败，可稍后重试',
    'container.detail.storage.source': '来源',
    'container.detail.storage.sourceUnavailable': '无来源',
    'container.detail.storage.type': '类型',
    'container.detail.storage.typeLabels.bind': 'Bind',
    'container.detail.storage.typeLabels.tmpfs': 'Tmpfs',
    'container.detail.storage.typeLabels.unknown': '未知',
    'container.detail.storage.typeLabels.volume': 'Volume',
    'container.detail.storage.unsupportedMessage': '此挂载暂不支持用量测量',
    'container.detail.storage.unsupportedTooltip': '当前挂载类型暂不支持用量统计。',
    'container.detail.storage.usage': '用量',
    'container.detail.storage.usageStatus.error': '测量失败',
    'container.detail.storage.usageStatus.measured': '已测量',
    'container.detail.storage.usageStatus.not_found': '挂载不存在',
    'container.detail.storage.usageStatus.not_measured': '未测量',
    'container.detail.storage.usageStatus.pending': '测量中',
    'container.detail.storage.usageStatus.permission_denied': '权限不足',
    'container.detail.storage.usageStatus.timeout': '测量超时',
    'container.detail.storage.usageStatus.unsupported': '不支持',
    'container.detail.summary.identity': '身份信息',
    'container.detail.summary.network': '网络访问',
    'container.detail.summary.resources': '资源使用',
    'container.detail.summary.runtime': '运行状态',
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
    'container.list.detail.portEmpty': '暂无端口映射。',
    'container.list.detail.workingDir': '工作目录',
    'container.list.eyebrow': '运维管理',
    'container.list.fields.createdAt': '创建时间',
    'container.list.fields.id': 'ID',
    'container.list.fields.image': '镜像',
    'container.list.fields.imageId': '镜像 ID',
    'container.list.fields.name': '名称',
    'container.list.fields.restartPolicy': '重启策略',
    'container.list.fields.runtime': '运行时',
    'container.list.fields.startedAt': '启动时间',
    'container.list.fields.state': '状态码',
    'container.list.fields.status': '状态',
    'container.list.health.healthy': '健康',
    'container.list.health.none': '未配置健康检查',
    'container.list.health.unavailable': '健康未知',
    'container.list.logs.loadFailed': '容器日志加载失败。',
    'container.list.retry': '重试',
    'container.list.states.running': '运行中',
  }),
);

vi.mock('../../api/container', () => ({
  getContainer: apiMocks.getContainer,
  getContainerLogs: apiMocks.getContainerLogs,
  getContainerMountUsage: apiMocks.getContainerMountUsage,
  postContainerMountUsageRefresh: apiMocks.postContainerMountUsageRefresh,
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

vi.mock('vue-router', async () => {
  const { reactive } = await vi.importActual<typeof import('vue')>('vue');
  routeState.route = reactive(routeState.route);
  return {
    useRoute: () => routeState.route,
    useRouter: () => routerMocks,
  };
});

vi.mock('@/store', () => ({
  useTabsRouterStore: () => tabStoreState,
}));

vi.mock('@/shared/observability', async () => {
  const actual = await vi.importActual<typeof import('@/shared/observability')>('@/shared/observability');
  return {
    ...actual,
    copyText: vi.fn().mockResolvedValue(true),
    formatLocaleDateTime: (value?: string | null) => value || '-',
  };
});

describe('container detail page', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    tabStoreState.tabRouterList = [
      {
        path: '/ops/containers/container-1',
        tabKey: '/ops/containers/container-1',
        title: { 'zh-CN': '容器详情', 'en-US': 'Container Detail' },
      },
    ];
    routeState.route.fullPath = '/ops/containers/container-1?tab=config';
    routeState.route.path = '/ops/containers/container-1';
    routeState.route.params.id = 'container-1';
    routeState.route.query.name = undefined;
    routeState.route.query.tab = 'config';
    Object.defineProperty(window, 'history', {
      configurable: true,
      value: { length: 1 },
    });
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: { writeText: vi.fn().mockResolvedValue(undefined) },
    });
    apiMocks.getContainer.mockResolvedValue(createContainerDetail());
    apiMocks.getContainerMountUsage.mockResolvedValue({
      items: createContainerDetail()
        .mounts.map((mount) => mount.usage)
        .filter(Boolean),
    });
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

  it('loads detail from the route id and renders the container overview workbench', async () => {
    const wrapper = mountPage();
    await flushPromises();

    expect(apiMocks.getContainer).toHaveBeenCalledWith('container-1');
    expect(wrapper.find('h1').text()).toBe('graft-web');
    expect(tabStoreState.tabRouterList[0]?.title?.['zh-CN']).toBe('容器详情 - graft-web');
    expect(tabStoreState.tabRouterList[0]?.title?.['en-US']).toBe('Container Detail - graft-web');
    expect(wrapper.text()).toContain('graft/web:latest');
    expect(wrapper.text()).not.toContain('容器详情 - graft-web');
    expect(wrapper.text()).toContain('graft/web:latest');
    expect(wrapper.text()).toContain('172.18.0.2');
    expect(wrapper.text()).toContain('21.8%');
    expect(wrapper.text()).toContain('31.25 GiB / 31.25 GiB');
    expect(wrapper.text()).toContain('8080:80/tcp');
    expect(wrapper.text()).toContain('身份信息');
    expect(wrapper.text()).toContain('运行状态');
    expect(wrapper.text()).toContain('资源使用');
    expect(wrapper.text()).toContain('资源仪表盘');
    expect(wrapper.text()).toContain('详细指标');
    expect(wrapper.text()).toContain('内存详细信息');
    expect(wrapper.text()).toContain('CPU 详细信息');
    expect(wrapper.text()).toContain('网络 I/O');
    expect(wrapper.text()).toContain('进程信息');
    expect(wrapper.text()).toContain('在线 CPU 4 CPU');
    expect(wrapper.text()).toContain('CPU 限制 / 在线 CPU— / 4 CPU');
    expect(wrapper.text()).toContain('累计 CPU 时间40.4 ms');
    expect(wrapper.text()).toContain('系统 CPU 时间50,468.37 s');
    expect(wrapper.text()).toContain('用户态时间10 ms');
    expect(wrapper.text()).toContain('内核态时间20 ms');
    expect(wrapper.text()).toContain('Throttling 次数0');
    expect(wrapper.text()).toContain('Throttling 时间0 ms');
    expect(wrapper.text()).toContain('未发生 CPU throttling');
    expect(wrapper.text()).not.toContain('50,468,370,000,000');
    expect(wrapper.text()).not.toContain('40401700');
    expect(wrapper.text()).not.toContain('10000000');
    expect(wrapper.text()).toContain('接收0.00 MiB');
    expect(wrapper.text()).toContain('发送0.00 MiB');
    expect(wrapper.text()).toContain('PIDs 当前数14');
    expect(wrapper.text()).toContain('暂未采集');
    expect(wrapper.text()).not.toContain('undefined');
    expect(wrapper.text()).not.toContain('null');
    expect(wrapper.text()).toContain('网络访问');
    expect(wrapper.text()).toContain('基础信息');
    expect(wrapper.text()).toContain('运行信息');
    expect(wrapper.get('.container-overview-panel').text()).toContain('资源与网络');
    expect(wrapper.get('.container-overview-panel').text()).not.toContain('资源摘要');
    expect(wrapper.get('.container-overview-panel').text()).not.toContain('网络摘要');
    expect(wrapper.text()).toContain('容器 IDcontainer-1');
    expect(wrapper.text()).toContain('镜像 IDbbbbbbbbbbbbbbbbbb...bbbbbbbbbb');
    expect(wrapper.text()).toContain('健康检查健康');
    expect(wrapper.text()).toContain('网络模式bridge');
    expect(wrapper.text()).toContain('网络名称bridge');
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

  it('uses the prefilled tab title before the detail request resolves', async () => {
    let resolveDetail: (value: ReturnType<typeof createContainerDetail>) => void = () => undefined;
    apiMocks.getContainer.mockReturnValue(
      new Promise((resolve) => {
        resolveDetail = resolve;
      }),
    );
    tabStoreState.tabRouterList = [
      {
        fullPath: '/ops/containers/container-1?tab=config',
        path: '/ops/containers/container-1',
        tabKey: '/ops/containers/container-1',
        title: { 'zh-CN': '容器详情 - list-name', 'en-US': 'Container Detail - list-name' },
      },
    ];

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.find('h1').text()).toBe('list-name');
    expect(tabStoreState.tabRouterList[0]?.title?.['zh-CN']).toBe('容器详情 - list-name');
    expect(wrapper.find('.container-detail-body').exists()).toBe(true);
    expect(wrapper.find('.container-detail-state').exists()).toBe(true);
    expect(wrapper.text()).not.toContain('暂无容器详情。');

    resolveDetail(createContainerDetail());
    await flushPromises();

    expect(wrapper.find('h1').text()).toBe('graft-web');
    expect(tabStoreState.tabRouterList[0]?.title?.['zh-CN']).toBe('容器详情 - graft-web');
  });

  it('keeps the fallback title when detail loading fails', async () => {
    apiMocks.getContainer.mockRejectedValue(new Error('network failed'));
    tabStoreState.tabRouterList = [
      {
        fullPath: '/ops/containers/container-1?tab=config',
        path: '/ops/containers/container-1',
        tabKey: '/ops/containers/container-1',
        title: { 'zh-CN': '容器详情 - list-name', 'en-US': 'Container Detail - list-name' },
      },
    ];

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.find('h1').text()).toBe('list-name');
    expect(tabStoreState.tabRouterList[0]?.title?.['zh-CN']).toBe('容器详情 - list-name');
    expect(wrapper.text()).toContain('network failed');
    expect(wrapper.text()).toContain('重试');
    expect(wrapper.find('.container-detail-state-alert').exists()).toBe(true);
    expect(wrapper.text()).not.toContain('暂无容器详情。');
    expect(wrapper.text()).not.toContain('undefined');
  });

  it('shows the empty state only after loading resolves without detail', async () => {
    let resolveDetail: (value: null) => void = () => undefined;
    apiMocks.getContainer.mockReturnValue(
      new Promise((resolve) => {
        resolveDetail = resolve;
      }),
    );

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.find('.container-detail-state').exists()).toBe(true);
    expect(wrapper.text()).toContain('loading');
    expect(wrapper.text()).not.toContain('暂无容器详情。');

    resolveDetail(null);
    await flushPromises();

    expect(wrapper.text()).toContain('暂无容器详情。');
    expect(wrapper.text()).not.toContain('loading');
  });

  it('falls back to the short route id for direct URL entry', async () => {
    let resolveDetail: (value: ReturnType<typeof createContainerDetail>) => void = () => undefined;
    apiMocks.getContainer.mockReturnValue(
      new Promise((resolve) => {
        resolveDetail = resolve;
      }),
    );
    tabStoreState.tabRouterList = [
      {
        path: '/ops/containers/338d02f869494842b74a70c84a64a84d7a9ce8caa945d552823bad060f7002e59',
        tabKey: '/ops/containers/338d02f869494842b74a70c84a64a84d7a9ce8caa945d552823bad060f7002e59',
        title: { 'zh-CN': '容器详情', 'en-US': 'Container Detail' },
      },
    ];
    routeState.route.path = '/ops/containers/338d02f869494842b74a70c84a64a84d7a9ce8caa945d552823bad060f7002e59';
    routeState.route.fullPath = `${routeState.route.path}?tab=overview`;
    routeState.route.params.id = '338d02f869494842b74a70c84a64a84d7a9ce8caa945d552823bad060f7002e59';
    routeState.route.query.tab = 'overview';

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.find('h1').text()).toBe('338d02f86949');
    expect(tabStoreState.tabRouterList[0]?.title?.['zh-CN']).toBe('容器详情 - 338d02f86949');

    resolveDetail(createContainerDetail());
    await flushPromises();
  });

  it('normalizes missing detail arrays without rendering failures', async () => {
    apiMocks.getContainer.mockResolvedValue({
      ...createContainerDetail(),
      command: null,
      entrypoint: null,
      environment: null,
      mounts: null,
      names: null,
      networks: null,
      ports: null,
    });

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain('无公开端口');
    expect(wrapper.text()).toContain('暂无网络信息。');
    expect(wrapper.text()).toContain('暂无挂载');
    expect(wrapper.text()).toContain('该容器未配置额外挂载卷');
    expect(wrapper.text()).not.toContain('暂无挂载。');
    expect(wrapper.text()).not.toContain('undefined');
  });

  it('renders mount cards with semantic type, access, usage, and weak states', async () => {
    routeState.route.query.tab = 'storage';

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.findAll('.container-mount-card')).toHaveLength(5);
    expect(wrapper.text()).toContain('/app');
    expect(wrapper.text()).toContain('基础信息');
    expect(wrapper.text()).toContain('/etc/graft');
    expect(wrapper.text()).toContain('/var/lib/graft');
    expect(wrapper.text()).toContain('/run');
    expect(wrapper.text()).toContain('/broken');
    expect(wrapper.text()).toContain('Bind');
    expect(wrapper.text()).toContain('Volume');
    expect(wrapper.text()).toContain('Tmpfs');
    expect(wrapper.text()).toContain('读写');
    expect(wrapper.text()).toContain('只读');
    expect(wrapper.text()).toContain('读写 rw');
    expect(wrapper.text()).toContain('只读 ro');
    expect(wrapper.text()).toContain('1.0 MiB');
    expect(wrapper.text()).toContain('5.0 MiB');
    expect(wrapper.text()).toContain('测量时间');
    expect(wrapper.text()).toContain('暂未测量挂载用量');
    expect(wrapper.text()).toContain('重新统计');
    expect(wrapper.text()).toContain('tmpfs is runtime memory backed');
    expect(
      findMountCardByDestination(wrapper, '/run').find('[data-testid="mount-refresh-unsupported-3"]').exists(),
    ).toBe(true);
    expect(wrapper.text()).toContain('permission denied');
    expect(findMountCardByDestination(wrapper, '/broken').find('[data-testid="mount-refresh-1"]').text()).toContain(
      '重试',
    );
    expect(wrapper.text()).toContain('Shared host path');
    expect(sourceText).not.toContain('const mountColumns');
    expect(sourceText).toContain('container-mount-card-grid');
  });

  it('renders an empty mount state when the container has no mounts', async () => {
    routeState.route.query.tab = 'storage';
    apiMocks.getContainer.mockResolvedValue({
      ...createContainerDetail(),
      mounts: [],
    });

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.findAll('.container-mount-card')).toHaveLength(0);
    expect(wrapper.text()).toContain('暂无挂载');
    expect(wrapper.text()).toContain('该容器未配置额外挂载卷');
    expect(wrapper.text()).not.toContain('暂无挂载。');
    expect(wrapper.find('.container-mount-empty__icon').exists()).toBe(true);
    expect(wrapper.find('.container-mount-empty__title').text()).toBe('暂无挂载');
    expect(wrapper.find('.container-mount-empty__description').text()).toBe('该容器未配置额外挂载卷');
    expect(wrapper.find('.container-mount-empty .t-empty-stub').exists()).toBe(false);
  });

  it('middle-truncates long mount paths while copying full values', async () => {
    routeState.route.query.tab = 'storage';
    const { copyText } = await import('@/shared/observability');
    const fullSource = '/srv/graft/releases/2026/06/14/containers/graft-web/shared/runtime/configuration/application';

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain('/srv/graft/releases...uration/application');
    expect(wrapper.text()).not.toContain(fullSource);

    await wrapper.get('[data-testid="mount-source-copy-0"]').trigger('click');
    await wrapper.get('[data-testid="mount-destination-copy-0"]').trigger('click');

    expect(copyText).toHaveBeenCalledWith(fullSource);
    expect(copyText).toHaveBeenCalledWith('/app');
  });

  it('refreshes only the selected mount card usage', async () => {
    routeState.route.query.tab = 'storage';
    const initialDetail = createContainerDetail();
    apiMocks.getContainer.mockResolvedValueOnce(initialDetail);
    apiMocks.postContainerMountUsageRefresh.mockResolvedValueOnce({
      container_id: 'container-1',
      destination: '/etc/graft',
      mount_id: 'mount-bind-ro',
      source: '/srv/graft/readonly/config',
      status: 'measured',
      type: 'bind',
      size_bytes: 2097152,
      measured_at: '2026-06-14T01:11:00Z',
      message: 'ro bind refreshed',
    });

    const wrapper = mountPage();
    await flushPromises();

    expect(findMountCardByDestination(wrapper, '/etc/graft').text()).toContain('未测量');
    await findMountCardByDestination(wrapper, '/etc/graft').get('[data-testid="mount-refresh-2"]').trigger('click');
    await flushPromises();

    expect(apiMocks.getContainer).toHaveBeenCalledTimes(1);
    expect(apiMocks.getContainerMountUsage).toHaveBeenCalledTimes(1);
    expect(apiMocks.postContainerMountUsageRefresh).toHaveBeenCalledWith('container-1', 'mount-bind-ro');
    expect(findMountCardByDestination(wrapper, '/app').text()).toContain('1.0 MiB');
    expect(findMountCardByDestination(wrapper, '/etc/graft').text()).toContain('2.0 MiB');
    expect(findMountCardByDestination(wrapper, '/etc/graft').text()).toContain('ro bind refreshed');
    expect(findMountCardByDestination(wrapper, '/var/lib/graft').text()).toContain('5.0 MiB');
    expect(messageMocks.success).toHaveBeenCalledWith('挂载用量已统计');
  });

  it('refreshes container detail and cached mount usage without triggering mount usage recompute', async () => {
    routeState.route.query.tab = 'storage';
    const wrapper = mountPage();
    await flushPromises();
    vi.clearAllMocks();
    apiMocks.getContainer.mockResolvedValueOnce({
      ...createContainerDetail(),
      inspect_updated_at: '2026-06-14T01:12:00Z',
    });
    apiMocks.getContainerMountUsage.mockResolvedValueOnce({
      items: [
        {
          container_id: 'container-1',
          destination: '/etc/graft',
          mount_id: 'mount-bind-ro',
          source: '/srv/graft/readonly/config',
          status: 'measured',
          type: 'bind',
          size_bytes: 3145728,
          measured_at: '2026-06-14T01:12:30Z',
        },
      ],
    });

    await wrapper.get('[data-refresh-now="true"]').trigger('click');
    await vi.waitFor(() => {
      expect(apiMocks.getContainer).toHaveBeenCalledWith('container-1');
    });
    await flushPromises();

    expect(apiMocks.getContainerMountUsage).toHaveBeenCalledWith('container-1');
    expect(apiMocks.postContainerMountUsageRefresh).not.toHaveBeenCalled();
    expect(findMountCardByDestination(wrapper, '/etc/graft').text()).toContain('3.0 MiB');
    expect(messageMocks.success).toHaveBeenCalledWith('容器详情已刷新');
  });

  it('renders manual refresh status through RefreshControlBar when auto refresh is off', async () => {
    routeState.route.query.tab = 'storage';
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.get('[data-refresh-countdown="true"]').text()).toContain('关闭');
    expect(wrapper.find('[data-refresh-toggle-auto="true"]').exists()).toBe(false);
    expect(wrapper.find('[data-testid="detail-back"]').exists()).toBe(false);
    expect(wrapper.text()).not.toContain('返回');
  });

  it('sorts mount cards by destination, source, and type instead of API order', async () => {
    routeState.route.query.tab = 'storage';
    const detailWithUnstableMountOrder = createContainerDetail();
    detailWithUnstableMountOrder.mounts = [...detailWithUnstableMountOrder.mounts].reverse();
    apiMocks.getContainer.mockResolvedValueOnce(detailWithUnstableMountOrder);
    apiMocks.getContainerMountUsage.mockResolvedValueOnce({ items: [] });

    const wrapper = mountPage();
    await flushPromises();

    expect(readMountDestinationOrder(wrapper)).toEqual(['/app', '/broken', '/etc/graft', '/run', '/var/lib/graft']);
    expect(sourceText).toContain(':key="mount.key"');
    expect(sourceText).not.toContain(':key="index"');
    expect(sourceText).toContain('refreshingMountKeys = ref<Set<string>>(new Set())');
  });

  it('keeps existing mount card order when refreshed inspect data returns mounts in a different order', async () => {
    routeState.route.query.tab = 'storage';
    const wrapper = mountPage();
    await flushPromises();
    vi.clearAllMocks();

    const reversedDetail = createContainerDetail();
    reversedDetail.mounts = [...reversedDetail.mounts].reverse();
    apiMocks.getContainer.mockResolvedValueOnce(reversedDetail);
    apiMocks.getContainerMountUsage.mockResolvedValueOnce({ items: [] });

    const beforeOrder = wrapper.findAll('.container-mount-card').map((card) => card.find('header').text());
    await wrapper.get('[data-refresh-now="true"]').trigger('click');
    await flushPromises();
    const afterOrder = wrapper.findAll('.container-mount-card').map((card) => card.find('header').text());

    expect(afterOrder).toEqual(beforeOrder);
  });

  it('keeps card order and reads recomputed usage back from cached top refresh results', async () => {
    routeState.route.query.tab = 'storage';
    const wrapper = mountPage();
    await flushPromises();
    vi.clearAllMocks();

    apiMocks.postContainerMountUsageRefresh.mockResolvedValueOnce({
      container_id: 'container-1',
      destination: '/etc/graft',
      mount_id: 'mount-bind-ro',
      source: '/srv/graft/readonly/config',
      status: 'measured',
      type: 'bind',
      size_bytes: 2097152,
      measured_at: '2026-06-14T01:11:00Z',
      message: 'cached from recompute',
    });

    await findMountCardByDestination(wrapper, '/etc/graft').get('[data-testid="mount-refresh-2"]').trigger('click');
    await flushPromises();

    expect(findMountCardByDestination(wrapper, '/etc/graft').text()).toContain('2.0 MiB');

    const reversedDetail = createContainerDetail();
    reversedDetail.mounts = [...reversedDetail.mounts].reverse();
    apiMocks.getContainer.mockResolvedValueOnce(reversedDetail);
    apiMocks.getContainerMountUsage.mockResolvedValueOnce({
      items: [
        {
          container_id: 'container-1',
          destination: '/etc/graft',
          mount_id: 'mount-bind-ro',
          source: '/srv/graft/readonly/config',
          status: 'measured',
          type: 'bind',
          size_bytes: 2097152,
          measured_at: '2026-06-14T01:11:00Z',
          message: 'cached from top refresh',
        },
      ],
    });

    const beforeOrder = wrapper.findAll('.container-mount-card').map((card) => card.find('header').text());
    await wrapper.get('[data-refresh-now="true"]').trigger('click');
    await flushPromises();
    const cards = wrapper.findAll('.container-mount-card');
    const afterOrder = cards.map((card) => card.find('header').text());

    expect(afterOrder).toEqual(beforeOrder);
    expect(findMountCardByDestination(wrapper, '/etc/graft').text()).toContain('2.0 MiB');
    expect(findMountCardByDestination(wrapper, '/etc/graft').text()).toContain('cached from top refresh');
    expect(apiMocks.postContainerMountUsageRefresh).toHaveBeenCalledTimes(1);
  });

  it('keeps top refresh loading independent from mount recompute loading', async () => {
    routeState.route.query.tab = 'storage';
    const wrapper = mountPage();
    await flushPromises();
    vi.clearAllMocks();
    const detailRequest = deferred<ReturnType<typeof createContainerDetail>>();
    apiMocks.getContainer.mockReturnValueOnce(detailRequest.promise);

    await wrapper.get('[data-refresh-now="true"]').trigger('click');
    await wrapper.vm.$nextTick();

    expect(wrapper.get('[data-refresh-now="true"]').attributes('data-loading')).toBe('true');
    expect(
      findMountCardByDestination(wrapper, '/etc/graft')
        .get('[data-testid="mount-refresh-2"]')
        .attributes('data-loading'),
    ).toBe('false');

    detailRequest.resolve(createContainerDetail());
    await flushPromises();
  });

  it('keeps mount recompute loading independent from top refresh loading', async () => {
    routeState.route.query.tab = 'storage';
    const wrapper = mountPage();
    await flushPromises();
    vi.clearAllMocks();
    const usageRequest = deferred<ContainerMountUsage>();
    apiMocks.postContainerMountUsageRefresh.mockReturnValueOnce(usageRequest.promise);

    await findMountCardByDestination(wrapper, '/etc/graft').get('[data-testid="mount-refresh-2"]').trigger('click');
    await wrapper.vm.$nextTick();

    expect(findMountCardByDestination(wrapper, '/etc/graft').get('[data-testid="mount-refresh-2"]').text()).toContain(
      '统计中...',
    );
    expect(
      findMountCardByDestination(wrapper, '/etc/graft')
        .get('[data-testid="mount-refresh-2"]')
        .attributes('data-loading'),
    ).toBe('true');
    expect(wrapper.get('[data-refresh-now="true"]').attributes('data-loading')).toBe('false');

    usageRequest.resolve({
      container_id: 'container-1',
      destination: '/etc/graft',
      mount_id: 'mount-bind-ro',
      source: '/srv/graft/readonly/config',
      status: 'measured',
      type: 'bind',
      size_bytes: 2097152,
    });
    await flushPromises();
  });

  it('does not overwrite a pending mount card with stale cached usage from top refresh', async () => {
    routeState.route.query.tab = 'storage';
    const wrapper = mountPage();
    await flushPromises();
    vi.clearAllMocks();
    const usageRequest = deferred<ContainerMountUsage>();
    apiMocks.postContainerMountUsageRefresh.mockReturnValueOnce(usageRequest.promise);

    await findMountCardByDestination(wrapper, '/etc/graft').get('[data-testid="mount-refresh-2"]').trigger('click');
    await wrapper.vm.$nextTick();

    apiMocks.getContainer.mockResolvedValueOnce(createContainerDetail());
    apiMocks.getContainerMountUsage.mockResolvedValueOnce({
      items: [
        {
          container_id: 'container-1',
          destination: '/etc/graft',
          mount_id: 'mount-bind-ro',
          source: '/srv/graft/readonly/config',
          status: 'measured',
          type: 'bind',
          size_bytes: 1024,
          measured_at: '2026-06-14T01:00:00Z',
        },
      ],
    });

    await wrapper.get('[data-refresh-now="true"]').trigger('click');
    await flushPromises();

    expect(findMountCardByDestination(wrapper, '/etc/graft').text()).toContain('统计中...');
    expect(findMountCardByDestination(wrapper, '/etc/graft').text()).not.toContain('1.0 KiB');

    usageRequest.resolve({
      container_id: 'container-1',
      destination: '/etc/graft',
      mount_id: 'mount-bind-ro',
      source: '/srv/graft/readonly/config',
      status: 'measured',
      type: 'bind',
      size_bytes: 4194304,
    });
    await flushPromises();

    expect(findMountCardByDestination(wrapper, '/etc/graft').text()).toContain('4.0 MiB');
  });

  it('auto refreshes with cached usage only and cleans up the timer', async () => {
    vi.useFakeTimers();
    routeState.route.query.tab = 'storage';
    const wrapper = mountPage();
    await flushPromises();
    vi.clearAllMocks();

    await wrapper.get('[data-refresh-interval-select="true"]').setValue('5');
    await wrapper.vm.$nextTick();
    expect(wrapper.get('[data-refresh-countdown="true"]').text()).toContain('自动刷新5s');

    await vi.advanceTimersByTimeAsync(5000);
    await flushPromises();

    expect(apiMocks.getContainer).toHaveBeenCalledTimes(1);
    expect(apiMocks.getContainerMountUsage).toHaveBeenCalledTimes(1);
    expect(apiMocks.postContainerMountUsageRefresh).not.toHaveBeenCalled();
    expect(wrapper.get('[data-refresh-countdown="true"]').text()).toContain('自动刷新5s');

    await wrapper.get('[data-testid="tab-logs"]').trigger('click');
    await wrapper.get('[data-testid="tab-logs"]').trigger('click');
    await vi.advanceTimersByTimeAsync(5000);
    await flushPromises();

    expect(apiMocks.getContainer).toHaveBeenCalledTimes(2);

    wrapper.unmount();
    await vi.advanceTimersByTimeAsync(10000);

    expect(apiMocks.getContainer).toHaveBeenCalledTimes(2);
    vi.useRealTimers();
  });

  it('pauses auto refresh while hidden and refreshes once when visible again', async () => {
    vi.useFakeTimers();
    routeState.route.query.tab = 'storage';
    const wrapper = mountPage();
    await flushPromises();
    vi.clearAllMocks();

    await wrapper.get('[data-refresh-interval-select="true"]').setValue('5');
    await flushPromises();
    vi.clearAllMocks();

    Object.defineProperty(document, 'visibilityState', {
      configurable: true,
      value: 'hidden',
    });
    document.dispatchEvent(new Event('visibilitychange'));
    await vi.advanceTimersByTimeAsync(5000);
    await flushPromises();

    expect(apiMocks.getContainer).not.toHaveBeenCalled();

    Object.defineProperty(document, 'visibilityState', {
      configurable: true,
      value: 'visible',
    });
    document.dispatchEvent(new Event('visibilitychange'));
    await flushPromises();

    expect(apiMocks.getContainer).toHaveBeenCalled();
    wrapper.unmount();
    vi.useRealTimers();
  });

  it('renders single network details as cards with one port mapping', async () => {
    routeState.route.query.tab = 'network';
    apiMocks.getContainer.mockResolvedValue({
      ...createContainerDetail(),
      ports: [{ ip: '0.0.0.0', private_port: 80, public_port: 8080, type: 'tcp' }],
      networks: [
        {
          name: 'arcane_default',
          ip_address: '172.24.0.2',
          gateway: '172.24.0.1',
          mac_address: 'd2:13:55:9f:0b:21',
          network_id: 'network-id-1',
          endpoint_id: 'endpoint-id-1',
        },
      ],
    });

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain('网络连接');
    expect(wrapper.find('.container-network-connection-card').exists()).toBe(true);
    expect(wrapper.text()).toContain('arcane_default');
    expect(wrapper.text()).toContain('172.24.0.2');
    expect(wrapper.text()).toContain('172.24.0.1');
    expect(wrapper.text()).toContain('d2:13:55:9f:0b:21');
    expect(wrapper.text()).toContain('network-id-1...k-id-1');
    expect(wrapper.text()).toContain('endpoint-id-...t-id-1');
    expect(wrapper.text()).toContain('映射');
    expect(wrapper.text()).toContain('宿主机 8080 → 容器 80/tcp');
    expect(wrapper.text()).toContain('监听地址');
    expect(wrapper.text()).toContain('0.0.0.0');
    expect(wrapper.text()).not.toContain('网络arcane_default');
    expect(wrapper.text()).not.toContain('0.0.0.0:8080->80/tcp');
    expect(wrapper.find('.container-port-mapping-card').exists()).toBe(true);
    expect(wrapper.text()).toContain('已发布到宿主机');
    expect(wrapper.text()).toContain('网络别名 / DNS');
    expect(wrapper.text()).toContain('暂无额外网络别名或自定义 DNS 配置');
  });

  it('aggregates IPv4 and IPv6 bindings for the same published port', async () => {
    const { copyText } = await import('@/shared/observability');
    routeState.route.query.tab = 'network';
    apiMocks.getContainer.mockResolvedValue({
      ...createContainerDetail(),
      ports: [
        { ip: '0.0.0.0', private_port: 3552, public_port: 3552, type: 'tcp' },
        { ip: '::', private_port: 3552, public_port: 3552, type: 'tcp' },
      ],
    });

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.find('.container-port-mapping-card').exists()).toBe(true);
    expect(wrapper.findAll('[data-testid^="port-mapping-copy-"]')).toHaveLength(1);
    expect(wrapper.text()).toContain('宿主机 3552 → 容器 3552/tcp');
    expect(wrapper.text()).toContain('0.0.0.0, ::');
    expect(wrapper.text()).not.toContain(':::3552->3552/tcp');

    await wrapper.get('[data-testid="port-mapping-copy-0"]').trigger('click');
    await flushPromises();

    expect(copyText).toHaveBeenCalledWith('0.0.0.0:3552->3552/tcp\n:::3552->3552/tcp');
  });

  it('renders a single IPv4 host binding as one semantic mapping', async () => {
    const { copyText } = await import('@/shared/observability');
    routeState.route.query.tab = 'network';
    apiMocks.getContainer.mockResolvedValue({
      ...createContainerDetail(),
      ports: [{ ip: '127.0.0.1', private_port: 8080, public_port: 18080, type: 'tcp' }],
    });

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.findAll('[data-testid^="port-mapping-copy-"]')).toHaveLength(1);
    expect(wrapper.text()).toContain('宿主机 18080 → 容器 8080/tcp');
    expect(wrapper.text()).toContain('127.0.0.1');
    expect(wrapper.text()).not.toContain('127.0.0.1:18080->8080/tcp');

    await wrapper.get('[data-testid="port-mapping-copy-0"]').trigger('click');
    await flushPromises();

    expect(copyText).toHaveBeenCalledWith('127.0.0.1:18080->8080/tcp');
  });

  it('renders multiple networks in the network table', async () => {
    routeState.route.query.tab = 'network';
    apiMocks.getContainer.mockResolvedValue({
      ...createContainerDetail(),
      networks: [
        {
          name: 'frontend',
          ip_address: '172.24.0.2',
          gateway: '172.24.0.1',
          mac_address: '02:42:ac:18:00:02',
        },
        {
          name: 'backend',
          ip_address: '172.25.0.2',
          gateway: '172.25.0.1',
          mac_address: '02:42:ac:19:00:02',
        },
      ],
    });

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.find('.container-network-connection-card').exists()).toBe(false);
    expect(wrapper.text()).toContain('frontend');
    expect(wrapper.text()).toContain('backend');
    expect(wrapper.text()).toContain('172.25.0.2');
  });

  it('renders an explicit empty port mapping section', async () => {
    routeState.route.query.tab = 'network';
    apiMocks.getContainer.mockResolvedValue({
      ...createContainerDetail(),
      ports: [],
    });

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain('端口映射');
    expect(wrapper.text()).toContain('暂无端口映射。');
  });

  it('renders unpublished container ports as internal-only compact rows', async () => {
    routeState.route.query.tab = 'network';
    apiMocks.getContainer.mockResolvedValue({
      ...createContainerDetail(),
      ports: [{ private_port: 8082, type: 'tcp' }],
    });

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.find('.container-port-mapping-card').exists()).toBe(true);
    expect(wrapper.text()).toContain('未发布到宿主机，仅容器网络内部可访问');
    expect(wrapper.text()).toContain('仅容器网络内部可访问');
    expect(wrapper.text()).not.toContain('8082/tcp / 未发布到宿主机');
    expect(wrapper.text()).not.toContain('暂无数据->8082/tcp');
    expect(wrapper.text()).not.toContain('暂无数据 -> 8082/tcp');
  });

  it('renders published host bindings without replacing empty listen addresses with no data', async () => {
    routeState.route.query.tab = 'network';
    apiMocks.getContainer.mockResolvedValue({
      ...createContainerDetail(),
      ports: [{ private_port: 80, public_port: 8080, type: 'tcp' }],
    });

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain('宿主机 8080 → 容器 80/tcp');
    expect(wrapper.text()).toContain('全部地址');
    expect(wrapper.text()).toContain('已发布到宿主机');
    expect(wrapper.text()).not.toContain('暂无数据:8080->80/tcp');
  });

  it('renders multiple port mappings in a table', async () => {
    routeState.route.query.tab = 'network';
    apiMocks.getContainer.mockResolvedValue({
      ...createContainerDetail(),
      ports: [
        { ip: '127.0.0.1', private_port: 80, public_port: 8080, type: 'tcp' },
        { private_port: 8082, type: 'tcp' },
      ],
    });

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.find('.container-port-mapping-card').exists()).toBe(false);
    expect(wrapper.text()).toContain('容器端口');
    expect(wrapper.text()).toContain('宿主机 8080 → 容器 80/tcp');
    expect(wrapper.text()).toContain('127.0.0.1');
    expect(wrapper.text()).toContain('未发布到宿主机，仅容器网络内部可访问');
    expect(wrapper.text()).not.toContain('127.0.0.1:8080->80/tcp');
  });

  it('shows fallback text for missing optional network fields', async () => {
    routeState.route.query.tab = 'network';
    apiMocks.getContainer.mockResolvedValue({
      ...createContainerDetail(),
      networks: [
        {
          name: 'bridge',
        },
      ],
    });

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain('子网');
    expect(wrapper.text()).toContain('Network ID');
    expect(wrapper.text()).toContain('Endpoint');
    expect(wrapper.text()).toContain('暂无数据');
    expect(wrapper.text()).not.toContain('undefined');
    expect(wrapper.text()).not.toContain('null');
  });

  it('renders copy buttons for network and port mapping fields', async () => {
    routeState.route.query.tab = 'network';
    apiMocks.getContainer.mockResolvedValue({
      ...createContainerDetail(),
      ports: [{ ip: '127.0.0.1', private_port: 3552, public_port: 3552, type: 'tcp' }],
      networks: [
        {
          name: 'arcane_default',
          ip_address: '172.24.0.2',
          mac_address: 'd2:13:55:9f:0b:21',
          network_id: 'network-id-1',
          endpoint_id: 'endpoint-id-1',
        },
      ],
    });

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.find('[data-testid="network-name-copy-0"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="network-ip-copy-0"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="network-mac-copy-0"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="network-id-copy-0"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="network-endpoint-copy-0"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="port-mapping-copy-0"]').exists()).toBe(true);
  });

  it('truncates network technical identifiers while copying full values', async () => {
    const { copyText } = await import('@/shared/observability');
    const networkId = '68e6eb2631f46e05e24714b3e8d03452e44cc91540f26f8c99bae3d953a0a9fb';
    const endpointId = 'd7fc919985a5657754bbf066b6cf31f44926f152a2876d7118b554dc411547b8';
    routeState.route.query.tab = 'network';
    apiMocks.getContainer.mockResolvedValue({
      ...createContainerDetail(),
      networks: [
        {
          name: 'bridge',
          ip_address: '172.17.0.5',
          gateway: '172.17.0.1',
          mac_address: 'ae:e8:86:3c:21:c5',
          network_id: networkId,
          endpoint_id: endpointId,
        },
      ],
    });

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain('68e6eb2631f4...a0a9fb');
    expect(wrapper.text()).toContain('d7fc919985a5...1547b8');
    expect(wrapper.text()).not.toContain(networkId);
    expect(wrapper.text()).not.toContain(endpointId);

    await wrapper.get('[data-testid="network-id-copy-0"]').trigger('click');
    await wrapper.get('[data-testid="network-endpoint-copy-0"]').trigger('click');
    await flushPromises();

    expect(copyText).toHaveBeenCalledWith(networkId);
    expect(copyText).toHaveBeenCalledWith(endpointId);
  });

  it('shows the alias and dns empty text when no extra network metadata exists', async () => {
    routeState.route.query.tab = 'network';
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain('Hostname');
    expect(wrapper.text()).toContain('graft-web');
    expect(wrapper.text()).toContain('暂无额外网络别名或自定义 DNS 配置');
    expect(wrapper.text()).not.toContain('Aliases暂无数据');
    expect(wrapper.text()).not.toContain('DNS暂无数据');
  });

  it('renders aliases and dns only when values exist', async () => {
    routeState.route.query.tab = 'network';
    apiMocks.getContainer.mockResolvedValue({
      ...createContainerDetail(),
      hostname: 'web-host',
      aliases: ['web', 'frontend'],
      dns: ['10.0.0.2', '10.0.0.3'],
    });

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain('Hostname');
    expect(wrapper.text()).toContain('web-host');
    expect(wrapper.text()).toContain('Aliases');
    expect(wrapper.text()).toContain('web, frontend');
    expect(wrapper.text()).toContain('DNS');
    expect(wrapper.text()).toContain('10.0.0.2, 10.0.0.3');
    expect(wrapper.text()).not.toContain('暂无额外网络别名或自定义 DNS 配置');
  });

  it('keeps footer visible globally and renders mutually exclusive detail states', () => {
    expect(sourceText).toContain('class="container-detail-body"');
    expect(sourceText).toContain('detailRefreshing && !safeDetail && !error');
    expect(sourceText).toContain('v-else-if="error"');
    expect(sourceText).toContain('v-else-if="safeDetail"');
    expect(sourceText).toContain('<t-empty v-else class="container-detail-state"');
    expect(sourceText).not.toContain('footer: false');
    expect(sourceText).not.toContain('calc(100vh');
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
    const { copyText } = await import('@/shared/observability');
    const wrapper = mountPage();
    await flushPromises();

    const copyButtons = wrapper.findAll('[data-testid="env-copy"]');
    expect(copyButtons).toHaveLength(1);

    await copyButtons[0].trigger('click');
    await flushPromises();

    expect(copyText).toHaveBeenCalledWith('production');
    expect(messageMocks.success).toHaveBeenCalledWith('已复制变量值');
  });

  it('preserves raw environment values when copyable', async () => {
    const { copyText } = await import('@/shared/observability');
    apiMocks.getContainer.mockResolvedValue({
      ...createContainerDetail(),
      environment: [
        {
          key: 'PADDED_VALUE',
          masked: false,
          sensitive: false,
          source: 'config',
          value: '  keep surrounding spaces  ',
        },
      ],
    });
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('[data-testid="env-copy"]').trigger('click');
    await flushPromises();

    expect(copyText).toHaveBeenCalledWith('  keep surrounding spaces  ');
  });

  it('copies masked environment values only from copy_value', async () => {
    const { copyText } = await import('@/shared/observability');
    apiMocks.getContainer.mockResolvedValue({
      ...createContainerDetail(),
      environment: [
        {
          copy_value: 'real-token-value',
          key: 'API_TOKEN',
          masked: true,
          sensitive: true,
          source: 'config',
        },
      ],
    });
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain('******');
    expect(wrapper.find('.container-env-table').text()).not.toContain('real-token-value');
    expect(wrapper.findAll('[data-testid="t-tag"]').filter((tag) => tag.text() === '******')).toHaveLength(0);

    await wrapper.get('[data-testid="env-copy"]').trigger('click');
    await flushPromises();

    expect(copyText).toHaveBeenCalledWith('real-token-value');
  });

  it('clears stale detail and reloads logs when the route id changes on the logs tab', async () => {
    routeState.route.query.tab = 'logs';
    const wrapper = mountPage();
    await flushPromises();

    const headingText = () => wrapper.get('h1').text();
    const logsText = () => wrapper.get('.log-viewer').text();

    expect(headingText()).toBe('graft-web');
    expect(logsText()).toContain('server started');

    const nextDetail = deferred<ReturnType<typeof createContainerDetail>>();
    const nextLogs = deferred<{
      id: string;
      lines: string[];
      runtime: string;
      stderr: boolean;
      stdout: boolean;
      tail: number;
      timestamps: boolean;
      truncated: boolean;
    }>();
    apiMocks.getContainer.mockReturnValue(nextDetail.promise);
    apiMocks.getContainerLogs.mockReturnValue(nextLogs.promise);

    const container2Detail = {
      ...createContainerDetail(),
      id: 'container-2',
      short_id: 'container-2',
      name: 'graft-api',
      names: ['graft-api'],
      image: 'graft/api:latest',
    };
    const container2Logs = {
      id: 'container-2',
      lines: ['api started'],
      runtime: 'docker',
      stderr: true,
      stdout: true,
      tail: 200,
      timestamps: false,
      truncated: false,
    };

    const detailCallCount = apiMocks.getContainer.mock.calls.length;
    const logsCallCount = apiMocks.getContainerLogs.mock.calls.length;

    routeState.route.params.id = 'container-2';
    await wrapper.vm.$nextTick();
    expect(apiMocks.getContainer.mock.calls.length).toBeGreaterThan(detailCallCount);
    expect(apiMocks.getContainerLogs.mock.calls.length).toBeGreaterThan(logsCallCount);
    expect(apiMocks.getContainer).toHaveBeenLastCalledWith('container-2');
    expect(apiMocks.getContainerLogs).toHaveBeenLastCalledWith('container-2', {
      tail: 200,
      since: undefined,
      stderr: true,
      stdout: true,
      timestamps: false,
    });

    expect(headingText()).toBe('container-2');
    expect(wrapper.find('.log-viewer').exists()).toBe(false);

    nextDetail.resolve(container2Detail);
    nextLogs.resolve(container2Logs);
    await flushPromises();
    expect(headingText()).toBe('graft-api');
    expect(logsText()).toContain('api started');
    wrapper.unmount();
  });

  it('clears stale detail on missing route id and load failure', async () => {
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain('graft-web');

    routeState.route.params.id = '';
    await wrapper.vm.$nextTick();
    await flushPromises();

    expect(wrapper.text()).not.toContain('graft/web:latest');
    expect(wrapper.text()).toContain('缺少容器标识。');

    routeState.route.params.id = 'container-failed';
    apiMocks.getContainer.mockRejectedValue(new Error('boom'));
    await wrapper.vm.$nextTick();
    await flushPromises();

    expect(wrapper.text()).not.toContain('graft/web:latest');
    expect(wrapper.text()).toContain('boom');
  });

  it('copies full identifiers from the overview section', async () => {
    const { copyText } = await import('@/shared/observability');
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('[data-testid="container-id-copy"]').trigger('click');
    await wrapper.get('[data-testid="image-id-copy"]').trigger('click');
    await flushPromises();

    expect(copyText).toHaveBeenCalledWith('ff007d095ed9faafdf39957cf4e2134dc9644a935c0e8d94bc3e599bcc518edb');
    expect(copyText).toHaveBeenCalledWith('bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb');
    expect(messageMocks.success).toHaveBeenCalledWith('内容已复制。');
  });

  it('renders missing healthcheck as not configured and avoids health unknown in overview', async () => {
    apiMocks.getContainer.mockResolvedValue({
      ...createContainerDetail(),
      health: undefined,
      healthcheck: undefined,
    });
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain('未配置健康检查');
    expect(wrapper.text()).not.toContain('健康未知');
  });

  it('renders the health tab as diagnostic cards for a healthy container', async () => {
    routeState.route.query.tab = 'health';
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain('健康诊断');
    expect(wrapper.text()).toContain('当前状态');
    expect(wrapper.text()).toContain('健康检查通过，容器运行中。');
    expect(wrapper.text()).toContain('重启次数');
    expect(wrapper.text()).toContain('0 次');
    expect(wrapper.text()).toContain('无异常重启');
    expect(wrapper.text()).toContain('健康检查结果');
    expect(wrapper.text()).toContain('Healthcheck');
    expect(wrapper.text()).toContain('通过');
    expect(wrapper.text()).toContain('Exit Code: 0');
    expect(wrapper.text()).toContain('0');
    expect(wrapper.text()).toContain('CMD-SHELL curl -f http://localhost:8080/health || exit 1');
    expect(wrapper.text()).toContain('无异常');
    expect(wrapper.text()).toContain('最近检查：2026-06-14T01:07:00Z');
    expect(wrapper.text()).toContain('运行稳定性');
    expect(wrapper.text()).toContain('状态稳定');
    expect(wrapper.text()).toContain('最近退出码');
    expect(wrapper.text()).toContain('OOMKilled');
    expect(wrapper.text()).toContain('否');
  });

  it('renders failed healthcheck diagnostics for an unhealthy container', async () => {
    routeState.route.query.tab = 'health';
    apiMocks.getContainer.mockResolvedValue({
      ...createContainerDetail(),
      health: 'unhealthy',
      healthcheck: {
        configured: true,
        status: 'unhealthy',
        command: ['CMD', 'curl', '-f', 'http://localhost:8080/health'],
        exit_code: 1,
        output: 'connection refused',
        checked_at: '2026-06-14T01:09:00Z',
        failing_streak: 3,
        failure_message: 'connection refused',
      },
    });
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain('异常');
    expect(wrapper.text()).toContain('健康检查失败，需要查看最近输出。');
    expect(wrapper.text()).toContain('失败');
    expect(wrapper.text()).toContain('connection refused');
    expect(wrapper.text()).toContain('CMD curl -f http://localhost:8080/health');
  });

  it('distinguishes a running container without Docker healthcheck details', async () => {
    routeState.route.query.tab = 'health';
    apiMocks.getContainer.mockResolvedValue({
      ...createContainerDetail(),
      health: 'none',
      healthcheck: undefined,
    });
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain('未配置健康检查');
    expect(wrapper.text()).toContain('容器运行中，但未配置 Docker Healthcheck。');
    expect(wrapper.text()).toContain('当前容器未提供 Docker Healthcheck 明细，仅能根据运行状态判断基础可用性。');
    expect(wrapper.text()).toContain('未配置 Docker Healthcheck');
    expect(wrapper.text()).not.toContain('Exit Code: 0');
  });

  it('marks exited containers as not running in health diagnostics', async () => {
    routeState.route.query.tab = 'health';
    apiMocks.getContainer.mockResolvedValue({
      ...createContainerDetail(),
      state: 'exited',
      status: 'Exited (0) 5 minutes ago',
      health: 'healthy',
      last_exit_code: 0,
    });
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain('未运行');
    expect(wrapper.text()).toContain('容器当前未处于运行状态。');
    expect(wrapper.text()).toContain('最近退出码');
    expect(wrapper.text()).toContain('0');
  });

  it('shows starting healthcheck as in progress without marking it passed', async () => {
    routeState.route.query.tab = 'health';
    apiMocks.getContainer.mockResolvedValue({
      ...createContainerDetail(),
      health: 'starting',
      healthcheck: {
        configured: true,
        status: 'starting',
        command: ['CMD', 'wget', '-q', '-T', '5', '-O', '/dev/null', 'http://localhost:8080/health'],
        exit_code: undefined,
        output: '',
        checked_at: '',
        failing_streak: 0,
      },
    });
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain('启动中');
    expect(wrapper.text()).toContain('健康检查仍在启动观察期。');
    expect(wrapper.text()).toContain('Exit Code: -');
    expect(wrapper.text()).toContain('无异常输出');
    expect(wrapper.text()).not.toContain('健康检查通过，容器运行中。');
  });

  it('surfaces non-zero restart counts as runtime stability signal', async () => {
    routeState.route.query.tab = 'health';
    apiMocks.getContainer.mockResolvedValue({
      ...createContainerDetail(),
      restart_count: 5,
    });
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain('5 次');
    expect(wrapper.text()).toContain('已记录 5 次重启');
    expect(wrapper.text()).toContain('存在重启');
    expect(wrapper.text()).toContain('重启策略');
    expect(wrapper.text()).toContain('unless-stopped');
  });

  it('prioritizes OOMKilled in runtime stability conclusion', async () => {
    routeState.route.query.tab = 'health';
    apiMocks.getContainer.mockResolvedValue({
      ...createContainerDetail(),
      last_exit_code: 137,
      oom_killed: true,
      restart_count: 3,
    });
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain('发生 OOM');
    expect(wrapper.text()).toContain('137');
    expect(wrapper.text()).toContain('是');
  });

  it('shows abnormal exit when the latest exit code is non-zero', async () => {
    routeState.route.query.tab = 'health';
    apiMocks.getContainer.mockResolvedValue({
      ...createContainerDetail(),
      last_exit_code: 2,
      oom_killed: false,
      restart_count: 0,
    });
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain('异常退出');
    expect(wrapper.text()).toContain('2');
  });

  it('uses short identifiers and renders an explicit empty port state', async () => {
    apiMocks.getContainer.mockResolvedValue({
      ...createContainerDetail(),
      ports: [],
      short_id: '',
    });
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain('ff007d095ed9');
    expect(wrapper.text()).toContain('无公开端口');
  });

  it('uses shared log and JSON viewers instead of raw pre blocks', () => {
    expect(sourceText).toContain('<log-viewer');
    expect(sourceText).toContain('<json-viewer');
    expect(sourceText).not.toContain('container-detail-code');
  });

  it('keeps overview as a single-column grouped information flow without nested scrolling', () => {
    const overviewStart = sourceText.indexOf('<container-overview-panel');
    const overviewEnd = sourceText.indexOf('<t-tab-panel value="resources"', overviewStart);
    const overviewSource = sourceText.slice(overviewStart, overviewEnd);

    expect(overviewSource).not.toContain('<t-card');
    expect(sourceText).not.toContain('container-overview-main-grid');
    expect(sourceText).not.toContain('container-overview-summary-strip');
    expect(overviewPanelSourceText).toContain('container-overview-panel');
    expect(overviewPanelSourceText).not.toContain('container-detail-scrollbar');
    expect(overviewPanelSourceText).toContain('container-info-section');
    expect(overviewPanelSourceText).toContain('container-info-row');
    expect(overviewPanelSourceText).toContain('width: 100%;');
    expect(overviewPanelSourceText).toContain('grid-template-columns: 112px minmax(0, 1fr);');
    expect(overviewPanelSourceText).not.toContain('overflow: hidden auto;');
    expect(overviewPanelSourceText).not.toContain('max-height: clamp');
    expect(overviewPanelSourceText).not.toContain('calc(100vh');
    expect(overviewPanelSourceText).not.toContain('scrollbar-color: var(--td-scrollbar-color) transparent;');
    expect(overviewSource).not.toContain('container.detail.overview.resourceSummary');
    expect(overviewSource).not.toContain('container.detail.overview.networkSummary');
  });

  it('renders resource details as a lightweight section instead of a bordered descriptions table', () => {
    const resourcesStart = sourceText.indexOf('<t-tab-panel value="resources"');
    const resourcesEnd = sourceText.indexOf('<t-tab-panel value="logs"', resourcesStart);
    const resourcesSource = sourceText.slice(resourcesStart, resourcesEnd);

    expect(resourcesSource).toContain('container-detail-resource-grid');
    expect(resourcesSource).toContain('container-resource-dashboard-section');
    expect(resourcesSource).toContain('container-resource-dashboard-grid');
    expect(resourcesSource).toContain('container-resource-detail-section');
    expect(resourcesSource).toContain('container-resource-detail-grid');
    expect(resourcesSource).toContain('container-resource-cpu-metric-grid');
    expect(resourcesSource).toContain('container-resource-detail-card--memory');
    expect(resourcesSource).toContain('cpuDetailMetrics');
    expect(resourcesSource).toContain('resourceDetailGroups');
    expect(resourcesSource).toContain('container-resource-detail-row');
    expect(resourcesSource).toContain('container.detail.resources.dashboard');
    expect(resourcesSource).toContain('container.detail.resources.detailedMetrics');
    expect(resourcesSource).toContain('container-resource-cpu-metric--warning');
    expect(resourcesSource).not.toContain('<t-descriptions');
    expect(resourcesSource).not.toContain('<t-descriptions-item');
    expect(resourcesSource).not.toContain('bordered');
    expect(resourcesSource).not.toContain("['system_cpu_usage'");
    expect(resourcesSource).not.toContain("['total_cpu_usage'");
    expect(resourcesSource).not.toContain("['cpu_usage_in_usermode'");
    expect(resourcesSource).not.toContain("['cpu_usage_in_kernelmode'");
  });

  it('renders health diagnostics as cards instead of one bordered descriptions table', () => {
    const healthStart = sourceText.indexOf('<t-tab-panel value="health"');
    const healthEnd = sourceText.indexOf('<t-tab-panel value="config"', healthStart);
    const healthSource = sourceText.slice(healthStart, healthEnd);

    expect(healthSource).toContain('container-health-summary-grid');
    expect(healthSource).toContain('container-health-info-card');
    expect(healthSource).toContain('healthcheckDetails');
    expect(healthSource).toContain('runtimeStability');
    expect(healthSource).toContain('<t-alert theme="info"');
    expect(healthSource).toContain('<t-empty');
    expect(healthSource).not.toContain(
      '<t-descriptions :column="2" item-layout="vertical" bordered table-layout="fixed">',
    );
  });

  it('renders the config tab as runtime config and searchable environment management', async () => {
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain('运行配置');
    expect(wrapper.text()).toContain('命令npm run serve');
    expect(wrapper.text()).toContain('入口docker-entrypoint.sh');
    expect(wrapper.text()).toContain('工作目录-');
    expect(wrapper.text()).toContain('环境变量');
    expect(wrapper.text()).toContain('3 项');
    expect(wrapper.get('input[placeholder="搜索变量名 / 值"]').attributes('placeholder')).toBe('搜索变量名 / 值');
    expect(wrapper.text()).toContain('安全策略：全部');
    expect(wrapper.text()).toContain('复制 .env');
    expect(wrapper.text()).toContain('安全策略');
    expect(wrapper.text()).toContain('******');
    expect(wrapper.text()).toContain('[已隐藏]');
    expect(wrapper.text()).not.toContain('API_TOKENundefined');

    await wrapper.get('input[placeholder="搜索变量名 / 值"]').setValue('APP');
    await flushPromises();

    expect(wrapper.text()).toContain('APP_MODE');
    expect(wrapper.text()).not.toContain('API_TOKEN');
    expect(wrapper.text()).not.toContain('SECRET_KEY');

    const select = wrapper.findAll('select').find((item) => item.text().includes('脱敏'));
    expect(select).toBeTruthy();
    await select!.setValue('masked');
    await flushPromises();

    expect(wrapper.text()).toContain('未找到匹配的环境变量');

    await wrapper.get('input[placeholder="搜索变量名 / 值"]').setValue('');
    await flushPromises();

    expect(wrapper.text()).toContain('API_TOKEN');
    expect(wrapper.text()).not.toContain('SECRET_KEY');
  });

  it('copies the filtered environment as safe dotenv content', async () => {
    const { copyText } = await import('@/shared/observability');
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('input[placeholder="搜索变量名 / 值"]').setValue('token');
    await flushPromises();
    await wrapper
      .findAll('button')
      .find((button) => button.text().includes('复制 .env'))
      ?.trigger('click');
    await flushPromises();

    expect(copyText).toHaveBeenCalledWith('API_TOKEN=******');
    expect(copyText).not.toHaveBeenCalledWith(expect.stringContaining('undefined'));
    expect(messageMocks.success).toHaveBeenCalledWith('已复制 .env 内容');
  });

  it('keeps detailed metric cards in a full-width memory and CPU flow before the lower two-column cards', () => {
    const memoryGroupStart = sourceText.indexOf("key: 'memory'");
    const cpuGroupStart = sourceText.indexOf("key: 'cpu'", memoryGroupStart);
    const networkGroupStart = sourceText.indexOf("key: 'network'", cpuGroupStart);
    const processGroupStart = sourceText.indexOf("key: 'process'", networkGroupStart);
    const styleStart = sourceText.indexOf('<style scoped lang="less">');
    const styleSource = sourceText.slice(styleStart);

    expect(memoryGroupStart).toBeGreaterThan(-1);
    expect(cpuGroupStart).toBeGreaterThan(memoryGroupStart);
    expect(networkGroupStart).toBeGreaterThan(cpuGroupStart);
    expect(processGroupStart).toBeGreaterThan(networkGroupStart);
    expect(styleSource).toContain('.container-resource-detail-card--cpu,');
    expect(styleSource).toContain('.container-resource-detail-card--memory');
    expect(styleSource).toContain('grid-column: 1 / -1;');
    expect(styleSource).toContain('.container-resource-detail-card__body--memory');
    expect(styleSource).toContain('grid-template-columns: repeat(2, minmax(0, 1fr));');
    expect(styleSource).toContain('.container-resource-detail-grid,');
    expect(styleSource).toContain('.container-resource-detail-card__body--memory,');
    expect(styleSource).toContain('.container-resource-cpu-metric-grid,');
    expect(styleSource).toContain('@media (width <= 720px)');
    expect(styleSource).toContain('grid-template-columns: 1fr;');
  });

  it('renders memory and CPU as full-width detail cards before the lower metric cards', async () => {
    const wrapper = mountPage();
    await flushPromises();

    const cards = wrapper.findAll('.container-resource-detail-card');
    expect(cards).toHaveLength(4);
    expect(cards[0].classes()).toContain('container-resource-detail-card--memory');
    expect(cards[0].find('.container-resource-detail-card__body--memory').exists()).toBe(true);
    expect(cards[1].classes()).toContain('container-resource-detail-card--cpu');
    expect(cards[1].findAll('.container-resource-cpu-metric')).toHaveLength(8);
    expect(cards[2].classes()).not.toContain('container-resource-detail-card--memory');
    expect(cards[2].classes()).not.toContain('container-resource-detail-card--cpu');
    expect(cards[2].text()).toContain('网络 I/O');
    expect(cards[3].classes()).not.toContain('container-resource-detail-card--memory');
    expect(cards[3].classes()).not.toContain('container-resource-detail-card--cpu');
    expect(cards[3].text()).toContain('进程信息');
  });

  it('orders memory detail rows as paired two-column metrics with a final placeholder', () => {
    const memoryGroupStart = sourceText.indexOf("key: 'memory'");
    const cpuGroupStart = sourceText.indexOf("key: 'cpu'", memoryGroupStart);
    const memorySource = sourceText.slice(memoryGroupStart, cpuGroupStart);

    const expectedOrder = [
      'memory_usage_bytes',
      'memory_cache',
      'memory_limit_bytes',
      'memory_rss',
      'memory_percent',
      'memory_active_file',
      'memory_inactive_file',
      'memory_pgfault',
      'memory_pgmajfault',
      'memory-placeholder',
    ];

    let previousIndex = -1;
    for (const marker of expectedOrder) {
      const nextIndex = memorySource.indexOf(marker);
      expect(nextIndex).toBeGreaterThan(previousIndex);
      previousIndex = nextIndex;
    }
    expect(memorySource).toContain("type: 'placeholder'");
    expect(memorySource).toContain("value: '—'");
  });
});

function createContainerDetail() {
  return {
    id: 'ff007d095ed9faafdf39957cf4e2134dc9644a935c0e8d94bc3e599bcc518edb',
    short_id: 'container-1',
    name: 'graft-web',
    names: ['graft-web'],
    image: 'graft/web:latest',
    image_id: 'sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb',
    labels: { 'com.docker.compose.project': 'graft' },
    status: 'Up 10 minutes',
    state: 'running',
    health: 'healthy',
    healthcheck: {
      configured: true,
      status: 'healthy',
      command: ['CMD-SHELL', 'curl -f http://localhost:8080/health || exit 1'],
      exit_code: 0,
      output: '无异常',
      checked_at: '2026-06-14T01:07:00Z',
      failing_streak: 0,
    },
    command: ['npm', 'run', 'serve'],
    entrypoint: ['docker-entrypoint.sh'],
    runtime: 'docker',
    created_at: '2026-06-14T01:00:00Z',
    started_at: '2026-06-14T01:05:00Z',
    inspect_updated_at: '2026-06-14T01:08:00Z',
    ports: [{ private_port: 80, public_port: 8080, type: 'tcp' }],
    mounts: [
      {
        mount_id: 'mount-bind-rw',
        type: 'bind',
        source: '/srv/graft/releases/2026/06/14/containers/graft-web/shared/runtime/configuration/application',
        destination: '/app',
        mode: 'rw',
        read_only: false,
        usage: {
          status: 'measured',
          size_bytes: 1048576,
          measured_at: '2026-06-14T01:09:00Z',
          message: 'du complete',
          shared_hint: 'Shared host path',
        },
      },
      {
        mount_id: 'mount-bind-ro',
        type: 'bind',
        source: '/srv/graft/readonly/config',
        destination: '/etc/graft',
        mode: 'ro',
        read_only: true,
        usage: {
          status: 'not_measured',
        },
      },
      {
        mount_id: 'mount-volume',
        type: 'volume',
        name: 'graft_data',
        source: '/var/lib/docker/volumes/graft_data/_data',
        destination: '/var/lib/graft',
        mode: 'rw',
        read_only: false,
        usage: {
          status: 'measured',
          size_bytes: 5242880,
          measured_at: '2026-06-14T01:10:00Z',
        },
      },
      {
        mount_id: 'mount-tmpfs',
        type: 'tmpfs',
        destination: '/run',
        mode: 'rw',
        read_only: false,
        usage: {
          status: 'unsupported',
          message: 'tmpfs is runtime memory backed',
        },
      },
      {
        mount_id: 'mount-error',
        type: 'bind',
        source: '/srv/graft/error',
        destination: '/broken',
        mode: 'rw',
        read_only: false,
        usage: {
          status: 'error',
          message: 'permission denied',
        },
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
      cpu_usage_in_kernelmode: 20_000_000,
      cpu_usage_in_usermode: 10_000_000,
      memory_limit_bytes: 33557250099.2,
      memory_percent: 100,
      memory_usage_bytes: 33557250099.2,
      memory_cache: 2048,
      memory_rss: 1024,
      online_cpus: 4,
      pids_current: 14,
      rx_bytes: 1536,
      rx_packets: 25,
      system_cpu_usage: 50_468_370_000_000,
      throttling_throttled_periods: 0,
      throttling_throttled_time: 0,
      total_cpu_usage: 40_401_700,
      tx_bytes: 126,
      tx_packets: 3,
    },
    primary_ip: '172.18.0.2',
    network_summary: 'bridge',
    restart_count: 0,
    restart_policy: 'unless-stopped',
    last_exit_code: 0,
    oom_killed: false,
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
          props: ['disabled', 'loading'],
          emits: ['click'],
          setup:
            (props, { attrs, emit, slots }) =>
            () => {
              const label = slots
                .default?.()
                .map((node) => String(node.children ?? ''))
                .join('');
              return h(
                'button',
                {
                  ...attrs,
                  disabled: Boolean(props.disabled),
                  'data-loading': String(Boolean(props.loading)),
                  'data-testid': attrs['data-testid'] ?? (label === '立即刷新' ? 'detail-refresh' : undefined),
                  onClick: () => emit('click'),
                  ...(attrs.onClick ? { onClick: attrs.onClick as () => void } : {}),
                },
                [slots.icon?.(), slots.default?.()],
              );
            },
        }),
        't-card': defineComponent({
          props: ['title'],
          setup:
            (props, { slots }) =>
            () =>
              h('section', [
                h('h2', slots.title?.() ?? String(props.title ?? '')),
                slots.actions?.(),
                slots.default?.(),
              ]),
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
          props: ['description', 'title'],
          setup:
            (props, { slots }) =>
            () =>
              h('div', { class: 't-empty-stub' }, [
                h('div', { class: 't-empty-stub__title' }, slots.title?.() ?? String(props.title ?? '')),
                h(
                  'div',
                  { class: 't-empty-stub__description' },
                  slots.description?.() ?? String(props.description ?? ''),
                ),
              ]),
        }),
        't-drawer': defineComponent({
          setup:
            (_, { slots }) =>
            () =>
              h('div', slots.default?.()),
        }),
        't-icon': defineComponent({
          props: ['name'],
          setup: (props) => () => h('span', String(props.name ?? '')),
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
          props: ['modelValue'],
          emits: ['update:modelValue'],
          setup:
            (props, { attrs, emit }) =>
            () =>
              h('input', {
                ...attrs,
                value: String(props.modelValue ?? ''),
                onInput: (event: Event) => emit('update:modelValue', (event.target as HTMLInputElement).value),
              }),
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
          props: ['modelValue', 'options', 'value'],
          emits: ['change', 'update:modelValue', 'update:value'],
          setup:
            (props, { attrs, emit }) =>
            () =>
              h(
                'select',
                {
                  ...attrs,
                  'data-testid': attrs['data-testid'] ?? undefined,
                  value: String(props.value ?? props.modelValue ?? ''),
                  onChange: (event: Event) => {
                    const rawValue = (event.target as HTMLSelectElement).value;
                    const value = Number.isNaN(Number(rawValue)) ? rawValue : Number(rawValue);
                    emit('update:modelValue', value);
                    emit('update:value', value);
                    emit('change', value);
                  },
                },
                (props.options as Array<{ label: string; value: string | number }> | undefined)?.map((option) =>
                  h('option', { value: option.value }, option.label),
                ) ?? [h('option', { value: 200 }, '200')],
              ),
        }),
        't-skeleton': defineComponent({
          setup: () => () => h('div', 'loading'),
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
            () => {
              const rows = props.data as Array<Record<string, unknown>>;
              return h('div', [
                ...(props.columns as Array<{ colKey: string; title: string }>).map((column) =>
                  h('strong', column.title),
                ),
                rows.length === 0 ? slots.empty?.() : undefined,
                ...rows.map((row, rowIndex) =>
                  h('div', { key: String(row.name ?? row.key ?? row.destination) }, [
                    (props.columns as Array<{ colKey: string }>).map(
                      (column) =>
                        slots[column.colKey]?.({ row, rowIndex }) ?? h('span', String(row[column.colKey] ?? '')),
                    ),
                  ]),
                ),
              ]);
            },
        }),
        't-tag': defineComponent({
          setup:
            (_, { slots }) =>
            () =>
              h('span', { 'data-testid': 't-tag' }, slots.default?.()),
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

function readMountDestinationOrder(wrapper: ReturnType<typeof mountPage>) {
  return wrapper
    .findAll('.container-mount-card')
    .map((card) =>
      ['/app', '/broken', '/etc/graft', '/run', '/var/lib/graft'].find((destination) =>
        card.text().includes(destination),
      ),
    )
    .filter((destination): destination is string => Boolean(destination));
}

function findMountCardByDestination(wrapper: ReturnType<typeof mountPage>, destination: string) {
  const card = wrapper.findAll('.container-mount-card').find((item) => item.text().includes(destination));
  if (!card) {
    throw new Error(`mount card not found: ${destination}`);
  }
  return card;
}
