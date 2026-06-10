// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { readFileSync } from 'node:fs';
import { join } from 'node:path';

import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, ref, type VNode } from 'vue';

import { formatCompactDateTime } from '@/shared/components/management';

import SystemConfigListPage from './index.vue';

const sourceText = readFileSync(join(process.cwd(), 'src/modules/system-config/pages/list/index.vue'), 'utf8');

const apiMocks = vi.hoisted(() => ({
  getSystemConfigs: vi.fn(),
  resetSystemConfig: vi.fn(),
  updateSystemConfig: vi.fn(),
}));

const translations = vi.hoisted(
  (): Record<string, string> => ({
    'menu.server.title': '服务管理',
    'systemConfig.fields.batchSize.description': '单次清理最多删除的日志行数。',
    'systemConfig.fields.batchSize.title': '批量大小',
    'systemConfig.fields.retentionDays.description': '删除早于指定天数的日志。',
    'systemConfig.fields.retentionDays.title': '日志保留时间',
    'systemConfig.domains.dashboard': '工作台配置',
    'systemConfig.domains.logs': '日志配置',
    'systemConfig.domains.notification': '站内通知',
    'systemConfig.groupDescriptions.dashboardQuickActions': '管理首页快捷入口的显示与排序策略。',
    'systemConfig.groupDescriptions.coreLoggerLogRetention': '管理应用日志清理的保留周期与批量策略。',
    'systemConfig.groupDescriptions.coreHttpxLogRetention': '管理访问日志清理的保留周期与批量策略。',
    'systemConfig.groups.coreLoggerLogRetention': '应用日志保留',
    'systemConfig.groups.coreHttpxLogRetention': '访问日志保留配置',
    'systemConfig.groups.dashboardQuickActions': '工作台快捷入口',
    'systemConfig.groups.notification.general': '通用',
    'systemConfig.groups.notification.general.description': '控制通知中心的基础行为。',
    'systemConfig.items.appLogRetentionCleanup.description': '应用日志保留清理任务的默认配置。',
    'systemConfig.items.appLogRetentionCleanup.title': '应用日志保留清理',
    'systemConfig.items.accessLogRetentionCleanup.description': '访问日志保留清理任务的默认配置。',
    'systemConfig.items.accessLogRetentionCleanup.title': '访问日志保留清理',
    'systemConfig.items.dashboardQuickActions.description': '工作台首页快捷入口的显示与排序默认配置。',
    'systemConfig.items.dashboardQuickActions.title': '工作台快捷入口',
    'systemConfig.fields.dashboardQuickActions.enabled.description': '控制工作台首页是否展示个性化快捷入口。',
    'systemConfig.fields.dashboardQuickActions.enabled.title': '是否启用',
    'systemConfig.fields.dashboardQuickActions.maxItems.description': '工作台首页默认展示的个性化入口数量。',
    'systemConfig.fields.dashboardQuickActions.maxItems.title': '最大数量',
    'systemConfig.fields.dashboardQuickActions.strategy.description': '个性化快捷入口的推荐排序策略。',
    'systemConfig.fields.dashboardQuickActions.strategy.title': '排序策略',
    'systemConfig.notification.notification.enabled.description': '是否启用站内通知功能。',
    'systemConfig.notification.notification.enabled.title': '启用通知',
    'systemConfig.notification.notification.retention_days.description': '通知记录的默认保留天数。',
    'systemConfig.notification.notification.retention_days.title': '通知保留天数',
    'systemConfig.list.boolean.disabled': '已禁用',
    'systemConfig.list.boolean.enabled': '已启用',
    'systemConfig.list.boolean.false': '否',
    'systemConfig.list.boolean.true': '是',
    'systemConfig.list.cancel': '取消',
    'systemConfig.list.description': '管理模块注册的系统级默认配置与用户覆盖值。',
    'systemConfig.list.edit': '编辑',
    'systemConfig.list.editorTitle': '编辑：{title}',
    'systemConfig.list.emptyDescription': '模块注册的 ConfigDefinition 会显示在这里。',
    'systemConfig.list.emptyTitle': '暂无系统配置',
    'systemConfig.list.emptyValue': '无数据',
    'systemConfig.list.eyebrow': '服务管理',
    'systemConfig.list.defaultGroup': '默认分组',
    'systemConfig.list.groupConfigCount': '{count} 个配置项',
    'systemConfig.list.groupLabel': '{module} / {group}',
    'systemConfig.list.loadError': '系统配置加载失败。',
    'systemConfig.list.masked': '已隐藏',
    'systemConfig.list.noDescription': '暂无描述。',
    'systemConfig.list.overrideCount': '{count} 个覆盖值',
    'systemConfig.list.previewTitle': '配置预览 JSON',
    'systemConfig.list.refresh': '刷新',
    'systemConfig.list.reset': '重置',
    'systemConfig.list.resetConfirm': '确认删除该用户覆盖值并回到模块默认值？',
    'systemConfig.list.save': '保存',
    'systemConfig.list.saveError': '系统配置保存失败。',
    'systemConfig.list.saveSuccess': '系统配置已保存。',
    'systemConfig.list.searchEmpty': '未找到匹配的配置组',
    'systemConfig.list.searchPlaceholder': '搜索配置组、配置项或技术标识',
    'systemConfig.list.schema.invalidJson': 'JSON 格式不正确',
    'systemConfig.list.schema.jsonPlaceholder': '请输入 JSON',
    'systemConfig.list.schema.numberPlaceholder': '请输入数字',
    'systemConfig.list.schema.selectPlaceholder': '请选择',
    'systemConfig.list.schema.stringPlaceholder': '请输入配置值',
    'systemConfig.list.schema.value': '配置值',
    'systemConfig.list.schema.invalidValue': '配置值不符合字段规则',
    'systemConfig.list.schema.invalidEnum': '请选择允许的配置值',
    'systemConfig.list.schema.belowMinimum': '配置值不能小于 {minimum}',
    'systemConfig.list.schema.aboveMaximum': '配置值不能大于 {maximum}',
    'systemConfig.list.status.default': '使用默认值',
    'systemConfig.list.status.defaultDescription': '使用默认配置',
    'systemConfig.list.status.modified': '已修改',
    'systemConfig.list.status.modifiedDescription': '存在用户覆盖',
    'systemConfig.list.status.title': '配置状态',
    'systemConfig.list.lastModified.none': '无覆盖值',
    'systemConfig.list.lastModified.title': '最后修改',
    'systemConfig.list.lastModified.unknownUser': '未知用户',
    'systemConfig.list.lastModified.userId': '用户 {id}',
    'systemConfig.list.lastModified.value': '{user} / {time}',
    'systemConfig.list.tags.override': '已修改',
    'systemConfig.list.tags.restartRequired': '需重启',
    'systemConfig.list.tags.sensitive': '敏感',
    'systemConfig.list.technicalId': '技术标识',
    'systemConfig.list.title': '系统配置',
    'systemConfig.list.valueDescription': '配置值说明',
    'systemConfig.list.values.current': '当前值',
    'systemConfig.list.values.default': '默认值',
    'systemConfig.list.values.effective': '生效值',
    'systemConfig.list.values.moreFields': '展开 {count} 个次要字段',
    'systemConfig.list.advanced.title': '高级信息',
    'systemConfig.list.advanced.copyKey': '复制 key',
    'systemConfig.list.advanced.copySuccess': '配置 key 已复制。',
    'systemConfig.list.advanced.copyError': '配置 key 复制失败。',
    'systemConfig.list.advanced.currentJson': '当前 JSON',
    'systemConfig.list.advanced.defaultJson': '默认 JSON',
    'systemConfig.list.advanced.schemaSummary': 'Schema 摘要',
    'systemConfig.list.advanced.schemaType': '类型：{type}',
    'systemConfig.list.advanced.schemaFieldCount': '字段数：{count}',
    'systemConfig.list.advanced.schemaRequiredCount': '必填字段：{count}',
    'systemConfig.list.advanced.schemaNoAdditionalProperties': '不允许额外字段',
    'systemConfig.options.dashboardQuickActionStrategy.hybrid': '综合推荐',
    'systemConfig.options.dashboardQuickActionStrategy.mostUsed': '最常使用',
    'systemConfig.options.dashboardQuickActionStrategy.recent': '最近访问',
    'systemConfig.options.dashboardQuickActionStrategyDescriptions.hybrid':
      '根据最近访问、使用频率和系统推荐结果综合排序。',
    'systemConfig.options.dashboardQuickActionStrategyDescriptions.mostUsed': '优先展示使用频率最高的快捷入口。',
    'systemConfig.options.dashboardQuickActionStrategyDescriptions.recent': '优先展示最近访问过的快捷入口。',
    'systemConfig.units.days': '天',
    'systemConfig.units.rows': '行',
  }),
);

vi.mock('../../api/system-config', () => ({
  getSystemConfigs: apiMocks.getSystemConfigs,
  resetSystemConfig: apiMocks.resetSystemConfig,
  updateSystemConfig: apiMocks.updateSystemConfig,
}));

vi.mock('tdesign-vue-next', () => ({
  MessagePlugin: {
    error: vi.fn(),
    success: vi.fn(),
  },
}));

vi.mock('tdesign-icons-vue-next', () => ({
  CopyIcon: defineComponent({ name: 'CopyIcon', setup: () => () => h('span') }),
  EditIcon: defineComponent({ name: 'EditIcon', setup: () => () => h('span') }),
  InfoCircleIcon: defineComponent({ name: 'InfoCircleIcon', setup: () => () => h('span', 'i') }),
  RefreshIcon: defineComponent({ name: 'RefreshIcon', setup: () => () => h('span') }),
  RollbackIcon: defineComponent({ name: 'RollbackIcon', setup: () => () => h('span') }),
  SearchIcon: defineComponent({ name: 'SearchIcon', setup: () => () => h('span') }),
}));

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) =>
      (translations[key] ?? key).replace(/\{(\w+)\}/g, (_, name) => String(params?.[name] ?? `{${name}}`)),
    te: (key: string) => Object.prototype.hasOwnProperty.call(translations, key),
  }),
}));

describe('system config list page', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    const items = dashboardQuickActionItems();
    apiMocks.getSystemConfigs.mockResolvedValue({
      items,
      total: items.length,
    });
  });

  it('renders backend-provided config metadata through key-first localization', async () => {
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.find('.page-header').exists()).toBe(true);
    expect(wrapper.find('.page-header').text()).toContain('服务管理');
    expect(wrapper.find('.page-header').text()).toContain('系统配置');
    expect(wrapper.text()).toContain('工作台配置');
    expect(wrapper.text()).toContain('工作台快捷入口');
    expect(wrapper.text()).toContain('管理首页快捷入口的显示与排序策略。');
    expect(wrapper.text()).toContain('1 个配置项');
    expect(wrapper.text()).toContain('0 个覆盖值');
    expect(wrapper.text()).toContain('是否启用');
    expect(wrapper.text()).toContain('最大数量');
    expect(wrapper.text()).toContain('排序策略');
    expect(wrapper.text()).toContain('当前值');
    expect(wrapper.text()).toContain('已启用');
    expect(wrapper.text()).toContain('综合推荐');
    expect(wrapper.text()).not.toContain('Personalized quick action ranking strategy.');
    expect(wrapper.text()).not.toContain('Maximum quick actions');
    expect(wrapper.find('[data-tooltip-content="根据最近访问、使用频率和系统推荐结果综合排序。"]').exists()).toBe(true);
    expect(
      wrapper
        .findAll('.system-config-value__rows small')
        .some((node) => node.text() === '根据最近访问、使用频率和系统推荐结果综合排序。'),
    ).toBe(false);
    expect(wrapper.text()).toContain('使用默认值');
    expect(wrapper.text()).toContain('默认值');
    expect(wrapper.text()).toContain('最后修改');
    expect(wrapper.text()).toContain('无覆盖值');
    expect(wrapper.text()).toContain('高级信息');
    expect(wrapper.text()).not.toContain('技术标识');
    expect(wrapper.text()).not.toContain('dashboard.quick_actions');
    expect(wrapper.findAll('button').some((button) => button.text() === '重置')).toBe(false);
    expect(wrapper.text()).not.toContain('生效值');
    expect(wrapper.text()).not.toContain('Dashboard Quick Actions');
    expect(wrapper.text()).not.toContain('core / dashboard.quick_actions');

    await toggleFirstCollapsePanel(wrapper, '高级信息');
    expect(wrapper.text()).toContain('技术标识');
    expect(wrapper.text()).toContain('dashboard.quick_actions');
    expect(wrapper.text()).toContain('当前 JSON');
    expect(wrapper.text()).toContain('默认 JSON');
    expect(wrapper.text()).toContain('Schema 摘要');

    await wrapper.find('button[data-test-id="edit-button"]').trigger('click');
    await flushPromises();

    expect(wrapper.text()).toContain('编辑：工作台快捷入口');
    expect(wrapper.text()).toContain('是否启用');
    expect(wrapper.text()).toContain('配置预览 JSON');
  });

  it('renders modified config as a user override with username and timestamp', async () => {
    apiMocks.getSystemConfigs.mockResolvedValue({
      items: [
        {
          ...systemConfigItem(),
          status: 'modified',
          has_override: true,
          updated_at: '2026-05-24T10:00:00Z',
          updated_by_user_id: 7,
          updated_by_username: 'alice',
        },
      ],
      total: 1,
    });

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain('已修改');
    expect(wrapper.text()).toContain('默认值');
    expect(wrapper.text()).toContain('当前值');
    expect(wrapper.text()).toContain(`alice / ${formatCompactDateTime('2026-05-24T10:00:00Z')}`);
    expect(wrapper.findAll('button').some((button) => button.text() === '重置')).toBe(true);
  });

  it('groups access-log and app-log retention under one logs domain', async () => {
    apiMocks.getSystemConfigs.mockResolvedValue({
      items: [
        systemConfigItem(),
        {
          ...systemConfigItem(),
          key: 'logger.app-log-retention-cleanup',
          module: 'core.logger',
          group_key: 'systemConfig.groups.coreLoggerLogRetention',
          group_label: 'App log retention',
          group_description_key: 'systemConfig.groupDescriptions.coreLoggerLogRetention',
          group_description: 'Manage application log cleanup retention and batch policy.',
          title_key: 'systemConfig.items.appLogRetentionCleanup.title',
          title: 'App log retention cleanup',
          description_key: 'systemConfig.items.appLogRetentionCleanup.description',
          description: 'Default cleanup configuration for app-log retention jobs.',
          tags: ['logger', 'log.retention'],
          order: 220,
        },
      ],
      total: 2,
    });

    const wrapper = mountPage();
    await flushPromises();

    expect(
      wrapper.findAll('[data-tree-node="domain"]').filter((node) => node.text().includes('日志配置')),
    ).toHaveLength(1);
    expect(wrapper.text()).toContain('访问日志保留配置');
    expect(wrapper.text()).toContain('应用日志保留');
  });

  it('renders notification config metadata from web catalog entries instead of backend fallback English', async () => {
    apiMocks.getSystemConfigs.mockResolvedValue({
      items: [
        notificationConfigItem({
          key: 'notification.enabled',
          titleKey: 'systemConfig.notification.notification.enabled.title',
          title: 'Notification enabled',
          descriptionKey: 'systemConfig.notification.notification.enabled.description',
          description: 'Whether in-app notifications are enabled.',
          type: 'boolean',
          configSchema: {
            type: 'boolean',
            'x-i18n': {
              titleKey: 'systemConfig.notification.notification.enabled.title',
              descriptionKey: 'systemConfig.notification.notification.enabled.description',
            },
          },
          defaultValue: 'true',
          effectiveValue: 'true',
          order: 5100,
        }),
        notificationConfigItem({
          key: 'notification.retention_days',
          titleKey: 'systemConfig.notification.notification.retention_days.title',
          title: 'Notification retention days',
          descriptionKey: 'systemConfig.notification.notification.retention_days.description',
          description: 'Number of days notification records should be retained.',
          type: 'integer',
          configSchema: {
            type: 'integer',
            minimum: 1,
            default: 30,
            title: 'Notification retention days',
            description: 'Number of days notification records should be retained.',
            'x-i18n': {
              titleKey: 'systemConfig.notification.notification.retention_days.title',
              descriptionKey: 'systemConfig.notification.notification.retention_days.description',
            },
          },
          defaultValue: '30',
          effectiveValue: '30',
          order: 5101,
        }),
      ],
      total: 2,
    });

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain('站内通知');
    expect(wrapper.text()).toContain('通用');
    expect(wrapper.text()).toContain('控制通知中心的基础行为。');
    expect(wrapper.text()).toContain('启用通知');
    expect(wrapper.text()).toContain('是否启用站内通知功能。');
    expect(wrapper.text()).toContain('通知保留天数');
    expect(wrapper.text()).not.toContain('Notification enabled');
    expect(wrapper.text()).not.toContain('Number of days notification records should be retained.');
  });

  it('uses item type fallback to render notification boolean config without schema as a switch', async () => {
    apiMocks.getSystemConfigs.mockResolvedValue({
      items: [
        notificationConfigItem({
          key: 'notification.enabled',
          titleKey: 'systemConfig.notification.notification.enabled.title',
          title: 'Notification enabled',
          descriptionKey: 'systemConfig.notification.notification.enabled.description',
          description: 'Whether in-app notifications are enabled.',
          type: 'boolean',
          configSchema: {},
          defaultValue: 'true',
          effectiveValue: 'true',
          order: 5100,
        }),
      ],
      total: 1,
    });

    const wrapper = mountPage();
    await flushPromises();

    await wrapper.find('button[data-test-id="edit-button"]').trigger('click');
    await flushPromises();

    expect(wrapper.find('[data-test-id="schema-switch"]').exists()).toBe(true);
    expect(wrapper.find('[data-test-id="schema-input"]').exists()).toBe(false);
    expect(wrapper.text()).toContain('编辑：启用通知');
    expect(wrapper.text()).toContain('是否启用站内通知功能。');
    expect(wrapper.text()).not.toContain('Notification enabled');
  });

  it('uses schema enum before item type fallback when rendering scalar editors', async () => {
    apiMocks.getSystemConfigs.mockResolvedValue({
      items: [
        notificationConfigItem({
          key: 'notification.delivery.mode',
          titleKey: '',
          title: 'Delivery mode',
          descriptionKey: '',
          description: 'Delivery mode.',
          type: 'boolean',
          configSchema: {
            type: 'string',
            enum: ['in_app', 'disabled'],
          },
          defaultValue: '"in_app"',
          effectiveValue: '"in_app"',
          order: 5102,
        }),
      ],
      total: 1,
    });

    const wrapper = mountPage();
    await flushPromises();

    await wrapper.find('button[data-test-id="edit-button"]').trigger('click');
    await flushPromises();

    expect(wrapper.find('[data-test-id="schema-select"]').exists()).toBe(true);
    expect(wrapper.find('[data-test-id="schema-switch"]').exists()).toBe(false);
  });

  it('localizes root scalar schema labels before falling back to backend schema copy', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.find('button[data-test-id="edit-button"]').trigger('click');
    await flushPromises();

    expect(wrapper.find('[data-testid="config-editor-dialog"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="config-editor-drawer"]').exists()).toBe(false);
    expect(wrapper.find('[data-test-id="schema-number"]').exists()).toBe(true);
    expect(wrapper.text()).toContain('编辑：工作台快捷入口');
    expect(wrapper.text()).toContain('最大数量');
    expect(wrapper.text()).toContain('工作台首页默认展示的个性化入口数量。');
    expect(wrapper.text()).not.toContain('Maximum quick actions');
    expect(wrapper.text()).not.toContain('Maximum personalized entries shown on the dashboard home page.');
  });

  it('filters the group tree by localized labels and technical keys', async () => {
    const items = [systemConfigItem(), ...dashboardQuickActionItems()];
    apiMocks.getSystemConfigs.mockResolvedValue({
      items,
      total: items.length,
    });

    const wrapper = mountPage();
    await flushPromises();

    await wrapper.find('[data-test-id="group-search"]').setValue('retention');
    await flushPromises();

    expect(wrapper.findAll('[data-tree-node="group"]').map((node) => node.text())).toEqual([
      '访问日志保留配置1 个配置项',
    ]);
    expect(wrapper.find('.system-config-content__head').text()).toContain('访问日志保留配置');

    await wrapper.find('[data-test-id="group-search"]').setValue('快捷');
    await flushPromises();

    expect(wrapper.findAll('[data-tree-node="group"]').map((node) => node.text())).toEqual([
      '工作台快捷入口1 个配置项',
    ]);
    expect(wrapper.find('.system-config-content__head').text()).toContain('工作台快捷入口');
  });

  it('keeps search text from every item in the same group', async () => {
    apiMocks.getSystemConfigs.mockResolvedValue({
      items: [
        dashboardQuickActionItem({
          key: 'dashboard.quick_actions.alpha_entry',
          titleKey: '',
          title: 'Alpha Entry',
          descriptionKey: '',
          description: 'Only the first grouped item mentions alpha.',
          type: 'string',
          configSchema: { type: 'string' },
          defaultValue: '"alpha"',
          effectiveValue: '"alpha"',
          order: 120,
        }),
        dashboardQuickActionItem({
          key: 'dashboard.quick_actions.beta_entry',
          titleKey: '',
          title: 'Beta Entry',
          descriptionKey: '',
          description: 'The final grouped item mentions beta.',
          type: 'string',
          configSchema: { type: 'string' },
          defaultValue: '"beta"',
          effectiveValue: '"beta"',
          order: 121,
        }),
      ],
      total: 2,
    });

    const wrapper = mountPage();
    await flushPromises();

    await wrapper.find('[data-test-id="group-search"]').setValue('alpha');
    await flushPromises();

    expect(wrapper.findAll('[data-tree-node="group"]').map((node) => node.text())).toEqual([
      '工作台快捷入口2 个配置项',
    ]);
    expect(wrapper.find('.system-config-content__head').text()).toContain('工作台快捷入口');
  });

  it('keeps settings navigation and content as independent scroll panes', async () => {
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.find('.system-config-page').exists()).toBe(true);
    expect(wrapper.find('.system-config-workspace').exists()).toBe(true);
    expect(wrapper.find('.system-config-layout').exists()).toBe(true);
    expect(wrapper.find('.system-config-groups__search').exists()).toBe(true);
    expect(wrapper.find('.system-config-groups.system-config-scrollbar').exists()).toBe(true);
    expect(wrapper.find('.system-config-content.system-config-scrollbar').exists()).toBe(true);
    expect(wrapper.find('.system-config-list').exists()).toBe(true);
    expect(wrapper.find('.system-config-content__head').exists()).toBe(true);
    expect(wrapper.find('.system-config-content__head').text()).toContain('工作台快捷入口');
  });

  it('declares the full left and right scroll container height chain', () => {
    expect(cssBlock('.system-config-page')).toContain('overflow: hidden;');
    expect(cssBlock('.system-config-workspace,')).toContain('height: 100%;');
    expect(cssBlock('.system-config-workspace,')).toContain('min-height: 0;');
    expect(cssBlock('.system-config-layout')).toContain('height: 100%;');
    expect(cssBlock('.system-config-layout')).toContain('min-height: 0;');
    expect(cssBlock('.system-config-layout')).toContain('overflow: hidden;');
    expect(cssBlock('.system-config-layout')).toContain('align-items: stretch;');
    expect(cssBlock('.system-config-groups')).toContain('align-self: stretch;');
    expect(cssBlock('.system-config-groups')).toContain('height: 100%;');
    expect(cssBlock('.system-config-groups')).toContain('min-height: 0;');
    expect(cssBlock('.system-config-groups')).toContain('overflow-y: auto;');
    expect(cssBlock('.system-config-content')).toContain('align-self: stretch;');
    expect(cssBlock('.system-config-content')).toContain('height: 100%;');
    expect(cssBlock('.system-config-content')).toContain('min-height: 0;');
    expect(cssBlock('.system-config-content')).toContain('overflow-y: auto;');
    expect(cssBlock('.system-config-content__head')).toContain('position: sticky;');
    expect(cssBlock('.system-config-list')).toContain('padding-bottom: var(--graft-density-gap-24);');
    expect(cssBlock('.system-config-scrollbar')).toContain('scrollbar-color: var(--td-scrollbar-color) transparent;');
    expect(cssBlock('.system-config-scrollbar::-webkit-scrollbar-thumb')).toContain(
      'background-color: var(--td-scrollbar-color);',
    );
  });

  it('falls back to user id when modified config has no username', async () => {
    apiMocks.getSystemConfigs.mockResolvedValue({
      items: [
        {
          ...systemConfigItem(),
          status: 'modified',
          has_override: true,
          updated_at: '2026-05-24T10:00:00Z',
          updated_by_user_id: 7,
          updated_by_username: '',
        },
      ],
      total: 1,
    });

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain(`用户 7 / ${formatCompactDateTime('2026-05-24T10:00:00Z')}`);
  });

  it('falls back to unknown user when modified config only has updated time', async () => {
    apiMocks.getSystemConfigs.mockResolvedValue({
      items: [
        {
          ...systemConfigItem(),
          status: 'modified',
          has_override: true,
          updated_at: '2026-05-24T10:00:00Z',
          updated_by_user_id: null,
          updated_by_username: '',
        },
      ],
      total: 1,
    });

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain(`未知用户 / ${formatCompactDateTime('2026-05-24T10:00:00Z')}`);
  });

  it('uses dialog for small flat object schemas', async () => {
    apiMocks.getSystemConfigs.mockResolvedValue({
      items: [systemConfigItem()],
      total: 1,
    });

    const wrapper = mountPage();
    await flushPromises();

    await wrapper.find('button[data-test-id="edit-button"]').trigger('click');
    await flushPromises();

    expect(wrapper.find('[data-testid="config-editor-dialog"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="config-editor-drawer"]').exists()).toBe(false);
    expect(wrapper.findAll('[data-test-id="schema-number"]')).toHaveLength(2);
  });

  it('renders schema number units outside the compact stepper input', async () => {
    apiMocks.getSystemConfigs.mockResolvedValue({
      items: [
        {
          ...systemConfigItem(),
          effective_value: '{"retentionDays":30,"batchSize":2000}',
        },
      ],
      total: 1,
    });

    const wrapper = mountPage();
    await flushPromises();

    await wrapper.find('button[data-test-id="edit-button"]').trigger('click');
    await flushPromises();

    const numberInputs = wrapper.findAll('[data-test-id="schema-number"]');
    expect(numberInputs).toHaveLength(2);
    expect(numberInputs.map((node) => node.text())).toEqual(['30', '2000']);
    expect(numberInputs.map((node) => node.attributes('data-suffix'))).toEqual(['', '']);
    expect(numberInputs.map((node) => node.attributes('data-align'))).toEqual(['center', 'center']);
    expect(numberInputs.map((node) => node.attributes('data-theme'))).toEqual(['row', 'row']);
    expect(numberInputs.map((node) => node.attributes('data-min'))).toEqual(['1', '1']);
    expect(numberInputs.map((node) => node.attributes('data-max'))).toEqual(['365', '10000']);
    expect(wrapper.findAll('.config-editor-renderer__number-unit').map((node) => node.text())).toEqual(['天', '行']);
  });

  it('uses drawer and raw JSON textarea fields for nested object or array properties', async () => {
    apiMocks.getSystemConfigs.mockResolvedValue({
      items: [
        {
          ...systemConfigItem(),
          key: 'notification.delivery-policy',
          title_key: '',
          title: 'Delivery policy',
          type: 'object',
          config_schema: {
            type: 'object',
            properties: {
              channels: {
                type: 'array',
                title: 'Channels',
              },
              metadata: {
                type: 'object',
                title: 'Metadata',
              },
            },
          },
          default_value: '{"channels":["inbox"],"metadata":{"priority":"normal"}}',
          effective_value: '{"channels":["inbox"],"metadata":{"priority":"normal"}}',
        },
      ],
      total: 1,
    });

    const wrapper = mountPage();
    await flushPromises();

    await wrapper.find('button[data-test-id="edit-button"]').trigger('click');
    await flushPromises();

    expect(wrapper.find('[data-testid="config-editor-dialog"]').exists()).toBe(false);
    expect(wrapper.find('[data-testid="config-editor-drawer"]').exists()).toBe(true);
    expect(wrapper.findAll('[data-test-id="schema-textarea"]')).toHaveLength(2);
  });

  it.each([
    ['boolean', false],
    ['string', ''],
    ['object', {}],
    ['array', []],
    ['number', null],
    ['integer', null],
  ] as const)('uses a blank %s value when editing sensitive config', async (type, expectedValue) => {
    apiMocks.getSystemConfigs.mockResolvedValue({
      items: [
        {
          ...systemConfigItem(),
          key: `sensitive.${type}`,
          title_key: '',
          title: `Sensitive ${type}`,
          type,
          config_schema: { type },
          sensitive: true,
          masked: true,
          default_value: '***',
          effective_value: '***',
        },
      ],
      total: 1,
    });
    apiMocks.updateSystemConfig.mockResolvedValue(systemConfigItem());

    const wrapper = mountPage();
    await flushPromises();

    await wrapper.find('button[data-test-id="edit-button"]').trigger('click');
    await flushPromises();
    const saveButton = wrapper.find('button[data-test-id="dialog-confirm"]').exists()
      ? wrapper.find('button[data-test-id="dialog-confirm"]')
      : wrapper.find('[data-test-id="editor-drawer-save"]');
    await saveButton.trigger('click');
    await flushPromises();

    expect(apiMocks.updateSystemConfig).toHaveBeenCalledWith(`sensitive.${type}`, {
      value: expectedValue,
    });
  });
});

function mountPage() {
  return mount(SystemConfigListPage, {
    global: {
      directives: {
        permission: {},
      },
      stubs: {
        TAlert: textStub('div'),
        TCard: textStub('article'),
        TButton: defineComponent({
          name: 'TButton',
          props: ['loading'],
          setup(_props, { attrs, slots }) {
            const isEditButton = () => slots.default?.()?.some((node) => String(node.children).includes('编辑'));
            return () =>
              h(
                'button',
                {
                  ...attrs,
                  'data-test-id': isEditButton() ? 'edit-button' : attrs['data-testid'],
                },
                slots.default?.(),
              );
          },
        }),
        TDialog: defineComponent({
          name: 'TDialog',
          props: ['visible', 'header'],
          emits: ['confirm'],
          setup(props, { attrs, emit, slots }) {
            return () =>
              props.visible
                ? h('section', attrs, [
                    h('h2', props.header as string),
                    slots.default?.(),
                    h(
                      'button',
                      {
                        'data-test-id': 'dialog-confirm',
                        onClick: () => emit('confirm'),
                      },
                      'confirm',
                    ),
                  ])
                : null;
          },
        }),
        TDrawer: defineComponent({
          name: 'TDrawer',
          props: ['visible', 'header'],
          setup(props, { attrs, slots }) {
            return () =>
              props.visible ? h('section', attrs, [h('h2', props.header as string), slots.default?.()]) : null;
          },
        }),
        TCollapse: textStub('section'),
        TCollapsePanel: defineComponent({
          name: 'TCollapsePanel',
          props: ['header'],
          setup(props, { slots }) {
            const expanded = ref(false);
            return () =>
              h('section', [
                h(
                  'button',
                  {
                    'data-test-id': 'collapse-panel-toggle',
                    onClick: () => {
                      expanded.value = !expanded.value;
                    },
                  },
                  props.header as string,
                ),
                expanded.value ? slots.default?.() : null,
              ]);
          },
        }),
        TEmpty: textStub('section'),
        TForm: textStub('form'),
        TFormItem: defineComponent({
          name: 'TFormItem',
          props: ['label', 'help'],
          setup(props, { slots }) {
            return () =>
              h('label', [props.label, props.help ? h('small', props.help as string) : null, slots.default?.()]);
          },
        }),
        TInput: defineComponent({
          name: 'TInput',
          props: ['modelValue', 'placeholder'],
          emits: ['update:modelValue'],
          setup(props, { attrs, emit }) {
            return () =>
              h('input', {
                ...attrs,
                'data-test-id': 'group-search',
                placeholder: props.placeholder as string,
                value: props.modelValue as string,
                onInput: (event: Event) => emit('update:modelValue', (event.target as HTMLInputElement).value),
              });
          },
        }),
        TInputNumber: defineComponent({
          name: 'TInputNumber',
          props: ['align', 'max', 'min', 'modelValue', 'suffix', 'theme'],
          setup(props) {
            return () =>
              h(
                'span',
                {
                  'data-align': props.align as string,
                  'data-max': String(props.max ?? ''),
                  'data-min': String(props.min ?? ''),
                  'data-suffix': String(props.suffix ?? ''),
                  'data-test-id': 'schema-number',
                  'data-theme': props.theme as string,
                },
                String(props.modelValue ?? ''),
              );
          },
        }),
        TLoading: textStub('div'),
        TOption: textStub('option'),
        TPopconfirm: textStub('div'),
        TSelect: defineComponent({
          name: 'TSelect',
          setup(_props, { slots }) {
            return () => h('select', { 'data-test-id': 'schema-select' }, slots.default?.());
          },
        }),
        TSpace: textStub('span'),
        TSwitch: defineComponent({
          name: 'TSwitch',
          props: ['modelValue'],
          setup(props) {
            return () => h('span', { 'data-test-id': 'schema-switch' }, String(props.modelValue));
          },
        }),
        TTag: textStub('span'),
        TTextarea: defineComponent({
          name: 'TTextarea',
          setup() {
            return () => h('textarea', { 'data-test-id': 'schema-textarea' });
          },
        }),
        TTooltip: defineComponent({
          name: 'TTooltip',
          props: ['content'],
          setup(props, { slots }) {
            return () => h('span', { 'data-tooltip-content': props.content as string }, slots.default?.());
          },
        }),
        TTree: defineComponent({
          name: 'TTree',
          props: ['data'],
          setup(props, { slots }) {
            const renderNode = (node: Record<string, unknown>): VNode =>
              h('li', { 'data-tree-node': node.children ? 'domain' : 'group' }, [
                slots.label?.({
                  node: {
                    data: node,
                  },
                }) ?? String(node.label ?? ''),
                Array.isArray(node.children)
                  ? h(
                      'ul',
                      node.children.map((child) => renderNode(child)),
                    )
                  : null,
              ]);
            return () => h('ul', Array.isArray(props.data) ? props.data.map((node) => renderNode(node)) : []);
          },
        }),
      },
    },
  });
}

async function toggleFirstCollapsePanel(wrapper: ReturnType<typeof mountPage>, header: string) {
  const toggle = wrapper.findAll('[data-test-id="collapse-panel-toggle"]').find((node) => node.text() === header);
  expect(toggle).toBeDefined();
  await toggle!.trigger('click');
  await flushPromises();
}

function textStub(tag: string) {
  return defineComponent({
    setup(_props, { slots }) {
      return () => h(tag, slots.default?.());
    },
  });
}

function cssBlock(selector: string) {
  const selectorStart = selector.endsWith(',') ? selector : `${selector} {`;
  const start = sourceText.indexOf(selectorStart);
  if (start < 0) {
    return '';
  }
  const openBrace = sourceText.indexOf('{', start);
  const closeBrace = sourceText.indexOf('}', openBrace);
  return sourceText.slice(openBrace + 1, closeBrace);
}

function systemConfigItem() {
  return {
    key: 'httpx.access-log-retention-cleanup',
    module: 'core.httpx',
    domain: 'logs',
    domain_key: 'systemConfig.domains.logs',
    domain_label: 'Logs',
    group: 'log.retention',
    group_key: 'systemConfig.groups.coreHttpxLogRetention',
    group_label: 'Access log retention',
    group_description_key: 'systemConfig.groupDescriptions.coreHttpxLogRetention',
    group_description: 'Manage access log cleanup retention and batch policy.',
    title_key: 'systemConfig.items.accessLogRetentionCleanup.title',
    title: 'Access log retention cleanup',
    description_key: 'systemConfig.items.accessLogRetentionCleanup.description',
    description: 'Default cleanup configuration for access-log retention jobs.',
    tags: ['httpx', 'log.retention'],
    type: 'object',
    config_schema: {
      type: 'object',
      properties: {
        retentionDays: {
          type: 'integer',
          minimum: 1,
          maximum: 365,
          default: 30,
          title: 'Log retention days',
          description: 'Delete logs older than this many days.',
          'x-i18n': {
            titleKey: 'systemConfig.fields.retentionDays.title',
            descriptionKey: 'systemConfig.fields.retentionDays.description',
            unitKey: 'systemConfig.units.days',
          },
        },
        batchSize: {
          type: 'integer',
          minimum: 1,
          maximum: 10000,
          default: 1000,
          title: 'Batch size',
          description: 'Maximum rows deleted per cleanup batch.',
          'x-i18n': {
            titleKey: 'systemConfig.fields.batchSize.title',
            descriptionKey: 'systemConfig.fields.batchSize.description',
            unitKey: 'systemConfig.units.rows',
          },
        },
      },
    },
    default_value: '{"retentionDays":30,"batchSize":1000}',
    effective_value: '{"retentionDays":30,"batchSize":1000}',
    override_value: null,
    has_override: false,
    sensitive: false,
    masked: false,
    restart_required: false,
    order: 210,
  };
}

function dashboardQuickActionItems() {
  return [
    dashboardQuickActionItem({
      key: 'dashboard.quick_actions',
      titleKey: 'systemConfig.items.dashboardQuickActions.title',
      title: 'Dashboard quick actions',
      descriptionKey: 'systemConfig.items.dashboardQuickActions.description',
      description: 'Dashboard home quick-action visibility and ranking defaults.',
      type: 'object',
      configSchema: {
        type: 'object',
        title: 'Dashboard quick actions',
        description: 'Dashboard home quick-action visibility and ranking defaults.',
        properties: {
          enabled: {
            type: 'boolean',
            default: true,
            title: 'Enabled',
            description: 'Controls whether personalized dashboard quick actions are shown.',
            'x-i18n': {
              titleKey: 'systemConfig.fields.dashboardQuickActions.enabled.title',
              descriptionKey: 'systemConfig.fields.dashboardQuickActions.enabled.description',
            },
          },
          maxItems: {
            type: 'integer',
            minimum: 1,
            maximum: 24,
            default: 4,
            title: 'Maximum quick actions',
            description: 'Maximum personalized entries shown on the dashboard home page.',
            'x-i18n': {
              titleKey: 'systemConfig.fields.dashboardQuickActions.maxItems.title',
              descriptionKey: 'systemConfig.fields.dashboardQuickActions.maxItems.description',
            },
          },
          strategy: {
            type: 'string',
            enum: ['most_used', 'recent', 'hybrid'],
            default: 'hybrid',
            title: 'Quick action strategy',
            description: 'Personalized quick action ranking strategy.',
            'x-i18n': {
              titleKey: 'systemConfig.fields.dashboardQuickActions.strategy.title',
              descriptionKey: 'systemConfig.fields.dashboardQuickActions.strategy.description',
              enumLabels: {
                most_used: {
                  labelKey: 'systemConfig.options.dashboardQuickActionStrategy.mostUsed',
                  descriptionKey: 'systemConfig.options.dashboardQuickActionStrategyDescriptions.mostUsed',
                },
                recent: {
                  labelKey: 'systemConfig.options.dashboardQuickActionStrategy.recent',
                  descriptionKey: 'systemConfig.options.dashboardQuickActionStrategyDescriptions.recent',
                },
                hybrid: {
                  labelKey: 'systemConfig.options.dashboardQuickActionStrategy.hybrid',
                  descriptionKey: 'systemConfig.options.dashboardQuickActionStrategyDescriptions.hybrid',
                },
              },
            },
          },
        },
        required: ['enabled', 'maxItems', 'strategy'],
        additionalProperties: false,
        'x-i18n': {
          titleKey: 'systemConfig.items.dashboardQuickActions.title',
          descriptionKey: 'systemConfig.items.dashboardQuickActions.description',
        },
      },
      defaultValue: '{"enabled":true,"maxItems":4,"strategy":"hybrid"}',
      effectiveValue: '{"enabled":true,"maxItems":4,"strategy":"hybrid"}',
      order: 120,
    }),
  ];
}

function dashboardQuickActionItem(input: {
  key: string;
  titleKey: string;
  title: string;
  descriptionKey: string;
  description: string;
  type: string;
  configSchema: Record<string, unknown>;
  defaultValue: string;
  effectiveValue: string;
  order: number;
}) {
  return {
    key: input.key,
    module: 'core',
    domain: 'dashboard',
    domain_key: 'systemConfig.domains.dashboard',
    domain_label: 'Dashboard',
    group: 'quick_actions',
    group_key: 'systemConfig.groups.dashboardQuickActions',
    group_label: 'Quick actions',
    group_description_key: 'systemConfig.groupDescriptions.dashboardQuickActions',
    group_description: 'Manage dashboard home quick-action visibility and ranking.',
    title_key: input.titleKey,
    title: input.title,
    description_key: input.descriptionKey,
    description: input.description,
    tags: ['dashboard', 'quick_actions'],
    type: input.type,
    config_schema: input.configSchema,
    default_value: input.defaultValue,
    effective_value: input.effectiveValue,
    override_value: null,
    has_override: false,
    sensitive: false,
    masked: false,
    restart_required: false,
    status: 'default',
    order: input.order,
  };
}

function notificationConfigItem(input: {
  key: string;
  titleKey: string;
  title: string;
  descriptionKey: string;
  description: string;
  type: string;
  configSchema: Record<string, unknown>;
  defaultValue: string;
  effectiveValue: string;
  order: number;
}) {
  return {
    key: input.key,
    module: 'notification',
    domain: 'notification',
    domain_key: 'systemConfig.domains.notification',
    domain_label: 'Notification',
    group: 'notification.general',
    group_key: 'systemConfig.groups.notification.general',
    group_label: 'Notification general',
    group_description_key: 'systemConfig.groups.notification.general.description',
    group_description: 'Control the Notification Center baseline behavior.',
    title_key: input.titleKey,
    title: input.title,
    description_key: input.descriptionKey,
    description: input.description,
    tags: ['notification', 'notification.general'],
    type: input.type,
    config_schema: input.configSchema,
    default_value: input.defaultValue,
    effective_value: input.effectiveValue,
    override_value: null,
    has_override: false,
    sensitive: false,
    masked: false,
    restart_required: false,
    status: 'default',
    order: input.order,
  };
}
