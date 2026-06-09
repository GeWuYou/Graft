// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import { formatCompactDateTime } from '@/shared/components/management';

import SystemConfigListPage from './index.vue';

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
    'systemConfig.groups.coreHttpxLogRetention': '访问日志保留配置',
    'systemConfig.items.accessLogRetentionCleanup.description': '访问日志保留清理任务的默认配置。',
    'systemConfig.items.accessLogRetentionCleanup.title': '访问日志保留清理',
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
    'systemConfig.list.groupConfigCount': '{count} 个配置项',
    'systemConfig.list.groupLabel': '{module} / {group}',
    'systemConfig.list.loadError': '系统配置加载失败。',
    'systemConfig.list.masked': '已隐藏',
    'systemConfig.list.noDescription': '暂无描述。',
    'systemConfig.list.overrideCount': '{count} 个覆盖值',
    'systemConfig.list.previewTitle': '配置预览',
    'systemConfig.list.refresh': '刷新',
    'systemConfig.list.reset': '重置',
    'systemConfig.list.resetConfirm': '确认删除该用户覆盖值并回到模块默认值？',
    'systemConfig.list.save': '保存',
    'systemConfig.list.saveError': '系统配置保存失败。',
    'systemConfig.list.saveSuccess': '系统配置已保存。',
    'systemConfig.list.schema.invalidJson': 'JSON 格式不正确',
    'systemConfig.list.schema.jsonPlaceholder': '请输入 JSON',
    'systemConfig.list.schema.numberPlaceholder': '请输入数字',
    'systemConfig.list.schema.selectPlaceholder': '请选择',
    'systemConfig.list.schema.stringPlaceholder': '请输入配置值',
    'systemConfig.list.schema.value': '配置值',
    'systemConfig.list.status.default': '默认',
    'systemConfig.list.status.defaultDescription': '使用默认配置',
    'systemConfig.list.status.modified': '已修改',
    'systemConfig.list.status.modifiedDescription': '存在用户覆盖',
    'systemConfig.list.status.title': '配置状态',
    'systemConfig.list.lastModified.none': '暂无修改记录',
    'systemConfig.list.lastModified.title': '最后修改',
    'systemConfig.list.lastModified.unknownUser': '未知用户',
    'systemConfig.list.lastModified.userId': '用户 {id}',
    'systemConfig.list.lastModified.value': '{user} / {time}',
    'systemConfig.list.tags.override': '已修改',
    'systemConfig.list.tags.restartRequired': '需重启',
    'systemConfig.list.tags.sensitive': '敏感',
    'systemConfig.list.technicalId': '技术标识',
    'systemConfig.list.title': '系统配置',
    'systemConfig.list.values.default': '默认值',
    'systemConfig.list.values.effective': '生效值',
    'systemConfig.list.viewJson': '查看 JSON',
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
  EditIcon: defineComponent({ name: 'EditIcon', setup: () => () => h('span') }),
  RefreshIcon: defineComponent({ name: 'RefreshIcon', setup: () => () => h('span') }),
  RollbackIcon: defineComponent({ name: 'RollbackIcon', setup: () => () => h('span') }),
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
    apiMocks.getSystemConfigs.mockResolvedValue({
      items: [systemConfigItem()],
      total: 1,
    });
  });

  it('renders backend-provided config metadata through key-first localization', async () => {
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.find('.page-header').exists()).toBe(true);
    expect(wrapper.find('.page-header').text()).toContain('服务管理');
    expect(wrapper.find('.page-header').text()).toContain('系统配置');
    expect(wrapper.text()).toContain('访问日志保留配置');
    expect(wrapper.text()).toContain('core.httpx / log.retention');
    expect(wrapper.text()).toContain('访问日志保留清理');
    expect(wrapper.text()).toContain('访问日志保留清理任务的默认配置。');
    expect(wrapper.text()).toContain('配置状态');
    expect(wrapper.text()).toContain('默认');
    expect(wrapper.text()).toContain('使用默认配置');
    expect(wrapper.text()).toContain('最后修改');
    expect(wrapper.text()).toContain('暂无修改记录');
    expect(wrapper.text()).toContain('默认值');
    expect(wrapper.text()).toContain('日志保留时间');
    expect(wrapper.text()).toContain('30 天');
    expect(wrapper.text()).toContain('批量大小');
    expect(wrapper.text()).toContain('1000 行');
    expect(wrapper.text()).toContain('查看 JSON');
    expect(wrapper.text()).toContain('技术标识');
    expect(wrapper.text()).toContain('httpx.access-log-retention-cleanup');
    expect(wrapper.text()).not.toContain('1 项');
    expect(wrapper.text()).not.toContain('无覆盖值');
    expect(wrapper.text()).not.toContain('httpxlog.retention');
    expect(wrapper.text()).not.toContain('Access log retention cleanup');
    expect(wrapper.text()).not.toContain('Default cleanup configuration for access-log retention jobs.');

    await wrapper.find('button[data-test-id="edit-button"]').trigger('click');
    await flushPromises();

    expect(wrapper.text()).toContain('编辑：访问日志保留清理');
    expect(wrapper.text()).toContain('日志保留时间');
    expect(wrapper.text()).toContain('删除早于指定天数的日志。');
    expect(wrapper.text()).toContain('批量大小');
    expect(wrapper.text()).toContain('单次清理最多删除的日志行数。');
    expect(wrapper.text()).toContain('配置预览');
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
    expect(wrapper.text()).toContain('存在用户覆盖');
    expect(wrapper.text()).toContain(`alice / ${formatCompactDateTime('2026-05-24T10:00:00Z')}`);
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
    await wrapper.find('button[data-test-id="dialog-confirm"]').trigger('click');
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
            return () =>
              h(
                'button',
                {
                  ...attrs,
                  'data-test-id': slots.default?.()?.some((node) => String(node.children).includes('编辑'))
                    ? 'edit-button'
                    : undefined,
                },
                slots.default?.(),
              );
          },
        }),
        TDialog: defineComponent({
          name: 'TDialog',
          props: ['visible', 'header'],
          emits: ['confirm'],
          setup(props, { emit, slots }) {
            return () =>
              props.visible
                ? h('section', [
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
        TCollapse: textStub('section'),
        TCollapsePanel: defineComponent({
          name: 'TCollapsePanel',
          props: ['header'],
          setup(props, { slots }) {
            return () => h('section', [h('button', props.header as string), slots.default?.()]);
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
        TInput: textStub('input'),
        TInputNumber: defineComponent({
          name: 'TInputNumber',
          props: ['suffix'],
          setup(props) {
            return () => h('span', props.suffix as string);
          },
        }),
        TLoading: textStub('div'),
        TOption: textStub('option'),
        TPopconfirm: textStub('div'),
        TSelect: textStub('select'),
        TSpace: textStub('span'),
        TSwitch: textStub('span'),
        TTag: textStub('span'),
        TTextarea: textStub('textarea'),
      },
    },
  });
}

function textStub(tag: string) {
  return defineComponent({
    setup(_props, { slots }) {
      return () => h(tag, slots.default?.());
    },
  });
}

function systemConfigItem() {
  return {
    key: 'httpx.access-log-retention-cleanup',
    module: 'core.httpx',
    group: 'log.retention',
    group_key: 'systemConfig.groups.coreHttpxLogRetention',
    group_label: 'core.httpx / log.retention',
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
