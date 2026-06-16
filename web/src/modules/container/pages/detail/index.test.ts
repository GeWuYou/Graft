// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { readFileSync } from 'node:fs';
import { join } from 'node:path';

import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import { CONTAINER_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';
import ContainerDetailPage from './index.vue';

const sourceText = readFileSync(join(process.cwd(), 'src/modules/container/pages/detail/index.vue'), 'utf8');
const overviewPanelSourceText = readFileSync(
  join(process.cwd(), 'src/modules/container/pages/detail/components/ContainerOverviewPanel.vue'),
  'utf8',
);

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

const routeState = vi.hoisted(
  (): {
    route: {
      params: { id: string };
      query: { tab?: string };
    };
  } => ({
    route: {
      params: { id: 'container-1' },
      query: { tab: 'config' },
    },
  }),
);

const translations = vi.hoisted(
  (): Record<string, string> => ({
    'container.detail.back': '返回',
    'container.detail.config.envName': '变量名',
    'container.detail.config.envPolicy': '策略',
    'container.detail.config.envValue': '值',
    'container.detail.config.environment': '环境变量',
    'container.detail.config.environmentUnavailable': '当前容器无法查看环境变量。',
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
    'container.detail.network.noPublicPorts': '无公开端口',
    'container.detail.network.ports': '端口映射',
    'container.detail.network.primaryIp': '主 IP',
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
    'container.detail.storage.destination': '挂载点',
    'container.detail.storage.mode': '模式',
    'container.detail.storage.source': '来源',
    'container.detail.storage.type': '类型',
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
    routeState.route.params.id = 'container-1';
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
    expect(messageMocks.success).toHaveBeenCalledWith('内容已复制。');
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

  it('clears stale detail and reloads logs when the route id changes on the logs tab', async () => {
    routeState.route.query.tab = 'logs';
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain('graft-web');
    expect(wrapper.text()).toContain('server started');

    apiMocks.getContainer.mockResolvedValue({
      ...createContainerDetail(),
      id: 'container-2',
      short_id: 'container-2',
      name: 'graft-api',
      names: ['graft-api'],
      image: 'graft/api:latest',
    });
    apiMocks.getContainerLogs.mockResolvedValue({
      id: 'container-2',
      lines: ['api started'],
      runtime: 'docker',
      stderr: true,
      stdout: true,
      tail: 200,
      timestamps: false,
      truncated: false,
    });

    routeState.route.params.id = 'container-2';
    await wrapper.vm.$nextTick();
    expect(wrapper.text()).not.toContain('graft-web');
    expect(wrapper.text()).not.toContain('server started');
    await flushPromises();

    expect(apiMocks.getContainer).toHaveBeenLastCalledWith('container-2');
    expect(apiMocks.getContainerLogs).toHaveBeenLastCalledWith('container-2', {
      tail: 200,
      since: undefined,
      stderr: true,
      stdout: true,
      timestamps: false,
    });
    expect(wrapper.text()).toContain('graft-api');
    expect(wrapper.text()).toContain('api started');
  });

  it('clears stale detail on missing route id and load failure', async () => {
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain('graft-web');

    routeState.route.params.id = '';
    await wrapper.vm.$nextTick();
    await flushPromises();

    expect(wrapper.text()).not.toContain('graft-web');
    expect(wrapper.text()).toContain('缺少容器标识。');

    routeState.route.params.id = 'container-failed';
    apiMocks.getContainer.mockRejectedValue(new Error('boom'));
    await wrapper.vm.$nextTick();
    await flushPromises();

    expect(wrapper.text()).not.toContain('graft-web');
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
    });
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain('未配置健康检查');
    expect(wrapper.text()).not.toContain('健康未知');
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

  it('falls back to the list route when there is no browser history', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('[data-testid="detail-back"]').trigger('click');
    await flushPromises();

    expect(routerMocks.push).toHaveBeenCalledWith({ name: CONTAINER_BOOTSTRAP_ROUTE.LIST.routeName });
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
                  ...(attrs.onClick ? { onClick: attrs.onClick as () => void } : {}),
                },
                [slots.icon?.(), slots.default?.()],
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
