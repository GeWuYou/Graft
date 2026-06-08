import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, nextTick } from 'vue';

import ScheduledTaskListPage from './index.vue';

const apiMocks = vi.hoisted(() => ({
  createScheduledTask: vi.fn(),
  executeScheduledTaskAction: vi.fn(),
  getScheduledTask: vi.fn(),
  getScheduledTaskJobDefinition: vi.fn(),
  getScheduledTaskJobDefinitions: vi.fn(),
  getScheduledTaskRuns: vi.fn(),
  getScheduledTasks: vi.fn(),
  updateScheduledTask: vi.fn(),
}));

const translations = vi.hoisted(
  (): Record<string, string> => ({
    'scheduledTask.accessLogRetention.description': '删除超过配置保留窗口的访问日志。',
    'scheduledTask.accessLogRetention.title': '访问日志保留清理',
    'scheduledTask.accessLogRetention.config.batchSize.description': '单次清理最多删除的访问日志行数。',
    'scheduledTask.accessLogRetention.config.batchSize.title': '批量大小',
    'scheduledTask.accessLogRetention.config.retentionDays.description': '删除早于该保留天数的访问日志。',
    'scheduledTask.accessLogRetention.config.retentionDays.title': '日志保留时间',
    'scheduledTask.action.dryRun.description': '预览本次执行结果',
    'scheduledTask.action.dryRun.title': '试运行',
    'scheduledTask.appLogRetention.config.batchSize.description': '单次清理最多删除的应用日志行数。',
    'scheduledTask.appLogRetention.config.batchSize.title': '批量大小',
    'scheduledTask.appLogRetention.config.dryRun.description': '只预览清理结果，不删除应用日志。',
    'scheduledTask.appLogRetention.config.dryRun.title': '试运行',
    'scheduledTask.appLogRetention.config.retentionDays.description': '删除早于该保留天数的应用日志。',
    'scheduledTask.appLogRetention.config.retentionDays.title': '日志保留时间',
    'scheduledTask.appLogRetention.description': '删除超过配置保留窗口的应用日志。',
    'scheduledTask.appLogRetention.title': '应用日志保留清理',
    'scheduledTask.auditLogRetention.config.batchSize.description': '单次清理最多删除的审计日志行数。',
    'scheduledTask.auditLogRetention.config.batchSize.title': '批量大小',
    'scheduledTask.auditLogRetention.config.dryRun.description': '只预览清理结果，不删除审计日志。',
    'scheduledTask.auditLogRetention.config.dryRun.title': '试运行',
    'scheduledTask.auditLogRetention.config.retentionDays.description': '删除早于该保留天数的审计日志。',
    'scheduledTask.auditLogRetention.config.retentionDays.title': '日志保留时间',
    'scheduledTask.auditLogRetention.description': '删除超过配置保留窗口的审计日志。',
    'scheduledTask.auditLogRetention.title': '审计日志保留清理',
    'scheduledTask.cronDescription.daily': '每天 {hour}:00 执行一次。',
    'scheduledTask.cronDescription.everyNMinutes': '每 {interval} 分钟执行一次。',
    'scheduledTask.cronValidation.fieldCount':
      'Cron 表达式必须是 {unixFields} 字段 Unix Cron 或 {secondsFields} 字段秒级 Cron。',
    'scheduledTask.cronValidation.fieldRange': 'Cron {field} 字段必须是 * 或 {min} 到 {max} 之间的数字。',
    'scheduledTask.cronValidation.required': '请填写 Cron 表达式。',
    'scheduledTask.cronValidation.stepRange': 'Cron {field} 步长必须介于 {min} 到 {max} 之间。',
    'scheduledTask.list.columnSettings': '列设置',
    'scheduledTask.list.columns.cron': 'Cron',
    'scheduledTask.list.columns.jobType': 'Job 类型',
    'scheduledTask.list.columns.operation': '操作',
    'scheduledTask.list.columns.recentResult': '最近结果',
    'scheduledTask.list.columns.recentRun': '最近运行',
    'scheduledTask.list.columns.status': '状态',
    'scheduledTask.list.columns.successRate': '成功率',
    'scheduledTask.list.columns.task': '任务',
    'scheduledTask.list.columns.taskName': '任务名称',
    'scheduledTask.list.create': '新建任务',
    'scheduledTask.list.cancel': '取消',
    'scheduledTask.list.delete': '删除',
    'scheduledTask.list.description': '管理绑定到 Job Definition 的定时任务。',
    'scheduledTask.list.detail.none': '无',
    'scheduledTask.list.detail.noError': '未记录错误',
    'scheduledTask.list.detail.behavior': '任务行为',
    'scheduledTask.list.detail.effectiveConfig': '最终配置',
    'scheduledTask.list.detail.details': '详情',
    'scheduledTask.list.detail.stage': '阶段',
    'scheduledTask.list.detail.affectedResource': '影响资源',
    'scheduledTask.list.detail.metrics': '指标',
    'scheduledTask.list.detail.rawResultJson': '原始结果 JSON',
    'scheduledTask.list.detail.warnings': '警告',
    'scheduledTask.list.disable': '停用',
    'scheduledTask.list.edit': '编辑',
    'scheduledTask.list.enable': '启用',
    'scheduledTask.list.eyebrow': '服务管理',
    'scheduledTask.list.filters.allJobTypes': '全部 Job 类型',
    'scheduledTask.list.filters.allStatuses': '全部状态',
    'scheduledTask.list.filters.jobType': 'Job 类型',
    'scheduledTask.list.filters.searchPlaceholder': '搜索任务',
    'scheduledTask.list.filters.status': '状态',
    'scheduledTask.cron.nextRun': '下次执行：{time}',
    'scheduledTask.cron.nextRunUnavailable': '无法计算',
    'scheduledTask.cron.expression': 'Cron 表达式',
    'scheduledTask.cron.description': '规则说明',
    'scheduledTask.cron.advancedExpression': '高级 Cron 表达式',
    'scheduledTask.cron.timezone': '时区',
    'scheduledTask.list.form.cronExpression': 'Cron 表达式',
    'scheduledTask.list.form.configHint': '根据 Job Definition 的 JSON Schema 填写配置。',
    'scheduledTask.list.form.configJsonInvalidHint': '配置必须是合法 JSON。',
    'scheduledTask.list.form.configJsonPlaceholder': '{\n  "batchSize": 1000\n}',
    'scheduledTask.list.form.configNumberPlaceholder': '请输入数值',
    'scheduledTask.list.form.configSelectPlaceholder': '请选择',
    'scheduledTask.list.form.configStringPlaceholder': '请输入内容',
    'scheduledTask.list.form.formatJson': '格式化 JSON',
    'scheduledTask.list.form.noConfigFields': '此 Job Definition 暂无可配置项。',
    'scheduledTask.list.form.cronRequiredHint': '请填写 Cron 表达式。',
    'scheduledTask.list.form.sectionBasicConfig': '基础配置',
    'scheduledTask.list.form.sectionConfig': '任务配置',
    'scheduledTask.list.validation.additionalProperty': '{field}不是允许的配置项。',
    'scheduledTask.list.validation.aboveMaximum': '{field}不能超过 {maximum}。',
    'scheduledTask.list.validation.belowMinimum': '{field}不能小于 {minimum}。',
    'scheduledTask.list.validation.enum': '{field}必须是以下值之一：{values}。',
    'scheduledTask.list.validation.required': '{field}为必填项。',
    'scheduledTask.list.validation.tooLong': '{field}长度不能超过 {maximum}。',
    'scheduledTask.list.validation.tooShort': '{field}长度不能少于 {minimum}。',
    'scheduledTask.list.validation.typeMismatch': '{field}必须是{expected}。',
    'scheduledTask.list.validation.types.boolean': '布尔值',
    'scheduledTask.list.validation.types.integer': '整数',
    'scheduledTask.list.validation.types.number': '数字',
    'scheduledTask.list.validation.types.string': '文本',
    'scheduledTask.list.configDialog.confirm': '保存配置',
    'scheduledTask.list.configDialog.customRetentionDays': '自定义',
    'scheduledTask.list.configDialog.doneJson': '完成',
    'scheduledTask.list.configDialog.editJson': '编辑 JSON',
    'scheduledTask.list.configDialog.jsonPreview': 'JSON 预览',
    'scheduledTask.list.configDialog.open': '配置',
    'scheduledTask.list.configDialog.retentionDaysOption': '{days} 天',
    'scheduledTask.list.configDialog.schemaDebug': 'Schema 调试信息',
    'scheduledTask.list.configDialog.title': '配置任务',
    'scheduledTask.list.action.affectedResource': '影响资源',
    'scheduledTask.list.action.behavior': '行为',
    'scheduledTask.list.action.confirm': '执行操作',
    'scheduledTask.list.action.confirmTitle': '确认操作',
    'scheduledTask.list.action.currentConfig': '当前配置',
    'scheduledTask.list.action.previewWarning': '本次操作不会修改任务配置。',
    'scheduledTask.list.action.sectionHint': '预览或检查 Job Definition 操作，不改变任务持久配置。',
    'scheduledTask.list.action.sectionTitle': 'Job 操作',
    'scheduledTask.list.action.taskName': '任务名称',
    'scheduledTask.list.actionResult.confirm': '关闭',
    'scheduledTask.list.actionResult.title': '操作结果',
    'scheduledTask.list.loadError': '定时任务数据加载失败。',
    'scheduledTask.list.metric.enabled': '已启用',
    'scheduledTask.list.metric.enabledDescription': '参与调度',
    'scheduledTask.list.metric.failures24h': '24 小时失败',
    'scheduledTask.list.metric.failures24hDescription': '最近 24 小时失败次数',
    'scheduledTask.list.metric.runs24h': '24 小时运行',
    'scheduledTask.list.metric.runs24hDescription': '最近 24 小时执行次数',
    'scheduledTask.list.metric.total': '任务总数',
    'scheduledTask.list.metric.totalDescription': '已注册任务',
    'scheduledTask.list.more': '更多',
    'scheduledTask.list.refresh': '刷新',
    'scheduledTask.list.run': '立即执行',
    'scheduledTask.list.runDialog.affectedResource': '影响资源',
    'scheduledTask.list.runDialog.batchSize': '批量大小',
    'scheduledTask.list.runDialog.cleanupDescription': '{behavior} 请确认清理范围后执行。',
    'scheduledTask.list.runDialog.cutoffPolicy': '截止策略/时间',
    'scheduledTask.list.runDialog.expectedBehavior': '预期行为',
    'scheduledTask.list.runDialog.retentionDays': '保留天数',
    'scheduledTask.list.resource.accessLog': '访问日志',
    'scheduledTask.list.save': '保存',
    'scheduledTask.list.status.failed': '失败',
    'scheduledTask.list.status.idle': '空闲',
    'scheduledTask.list.status.success': '成功',
    'scheduledTask.list.tableHint': '当前筛选显示 {count} 个任务。',
    'scheduledTask.list.tableTitle': '任务列表',
    'scheduledTask.list.title': '定时任务',
    'scheduledTask.list.viewDetail': '查看',
  }),
);

vi.mock('../../api/scheduled-task', () => ({
  createScheduledTask: apiMocks.createScheduledTask,
  deleteScheduledTask: vi.fn(),
  disableScheduledTask: vi.fn(),
  enableScheduledTask: vi.fn(),
  executeScheduledTaskAction: apiMocks.executeScheduledTaskAction,
  getScheduledTask: apiMocks.getScheduledTask,
  getScheduledTaskJobDefinition: apiMocks.getScheduledTaskJobDefinition,
  getScheduledTaskJobDefinitions: apiMocks.getScheduledTaskJobDefinitions,
  getScheduledTaskRun: vi.fn(),
  getScheduledTaskRuns: apiMocks.getScheduledTaskRuns,
  getScheduledTasks: apiMocks.getScheduledTasks,
  runScheduledTask: vi.fn(),
  updateScheduledTask: apiMocks.updateScheduledTask,
}));

vi.mock('@/utils/logger', () => ({
  createLogger: () => ({
    error: vi.fn(),
    warn: vi.fn(),
  }),
}));

vi.mock('tdesign-vue-next', async () => {
  const { defineComponent, h } = await import('vue');

  return {
    MessagePlugin: {
      error: vi.fn(),
      success: vi.fn(),
      warning: vi.fn(),
    },
    Tag: defineComponent({
      name: 'TTag',
      setup(_props, { slots }) {
        return () => h('span', slots.default?.());
      },
    }),
  };
});

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    locale: { value: 'zh-CN' },
    t: (key: string, params?: Record<string, unknown>) =>
      (translations[key] ?? key).replace(/\{(\w+)\}/g, (_, name) => String(params?.[name] ?? `{${name}}`)),
    te: (key: string) => Object.prototype.hasOwnProperty.call(translations, key),
  }),
}));

vi.spyOn(Intl.DateTimeFormat.prototype, 'resolvedOptions').mockReturnValue({
  calendar: 'gregory',
  locale: 'zh-CN',
  numberingSystem: 'latn',
  timeZone: 'Asia/Shanghai',
} as Intl.ResolvedDateTimeFormatOptions);

vi.setSystemTime(new Date('2026-06-06T08:00:00+08:00'));

function scheduledTasksResponse() {
  return {
    items: [
      {
        key: 'httpx.access-log-retention-cleanup',
        job_key: 'httpx.access-log-retention-cleanup',
        schedule_type: 'cron',
        display_name_key: 'scheduledTask.accessLogRetention.title',
        description_key: 'scheduledTask.accessLogRetention.description',
        owner: 'core.httpx',
        module: 'core.httpx',
        enabled: true,
        builtin: true,
        title: 'Access log retention cleanup',
        description: 'Deletes access logs beyond the configured retention window.',
        schedule: '*/5 * * * *',
        status: 'idle',
        running: false,
        config_json: '{"retentionDays":30,"batchSize":500}',
        effective_config: '{"retentionDays":30,"batchSize":500}',
        last_run: {
          id: 101,
          trigger_type: 'cron',
          status: 'success',
          started_at: '2026-06-05T00:00:00Z',
          finished_at: '2026-06-05T00:00:05Z',
          duration_ms: 5000,
          error_summary: '',
          result_json:
            '{"summary":"Deleted 3 access log rows.","stage":"completed","affected_resource":"access_log","metrics":{"deletedRows":3}}',
        },
      },
      {
        key: 'logger.app-log-retention-cleanup',
        job_key: 'logger.app-log-retention-cleanup',
        schedule_type: 'cron',
        display_name_key: 'scheduledTask.appLogRetention.title',
        description_key: 'scheduledTask.appLogRetention.description',
        owner: 'core.logger',
        module: 'core.logger',
        enabled: true,
        builtin: true,
        title: 'App log retention cleanup',
        description: 'Deletes app logs beyond the configured retention window.',
        schedule: '*/5 * * * *',
        status: 'idle',
        running: false,
        config_json: '{}',
        last_run: {
          id: 102,
          trigger_type: 'cron',
          status: 'failed',
          started_at: '2026-06-05T00:10:00Z',
          finished_at: '2026-06-05T00:10:01Z',
          duration_ms: 1000,
          error_summary: 'retention window is invalid',
          result_json: '{"stage":"failed","affected_resource":"app_log","warnings":["retention window is invalid"]}',
        },
      },
      {
        key: 'audit.audit-log-retention-cleanup',
        job_key: 'audit.audit-log-retention-cleanup',
        schedule_type: 'cron',
        display_name_key: 'scheduledTask.auditLogRetention.title',
        description_key: 'scheduledTask.auditLogRetention.description',
        owner: 'audit',
        module: 'audit',
        enabled: true,
        builtin: true,
        title: 'Audit log retention cleanup',
        description: 'Deletes audit logs beyond the configured retention window.',
        schedule: '*/5 * * * *',
        status: 'idle',
        running: false,
        config_json: '{}',
      },
      {
        key: 'custom.task',
        job_key: 'audit.audit-log-retention-cleanup',
        schedule_type: 'cron',
        display_name_key: 'scheduledTask.auditLogRetention.title',
        description_key: 'scheduledTask.auditLogRetention.description',
        owner: 'audit',
        module: 'audit',
        enabled: true,
        builtin: false,
        title: 'Custom cleanup',
        description: 'Custom description',
        schedule: '0 17 * * *',
        status: 'idle',
        running: false,
        config_json: '{}',
      },
    ],
    total: 4,
  };
}

function jobDefinitionsResponse() {
  const configSchemaJson = (prefix: string, resource: string) =>
    JSON.stringify({
      type: 'object',
      properties: {
        retentionDays: {
          type: 'integer',
          minimum: 1,
          maximum: 365,
          'x-i18n': {
            titleKey: `scheduledTask.${prefix}.config.retentionDays.title`,
            descriptionKey: `scheduledTask.${prefix}.config.retentionDays.description`,
          },
          title: 'Retention days',
          description: `Delete ${resource} older than this number of days.`,
          default: 30,
        },
        batchSize: {
          type: 'integer',
          minimum: 1,
          maximum: 10000,
          'x-i18n': {
            titleKey: `scheduledTask.${prefix}.config.batchSize.title`,
            descriptionKey: `scheduledTask.${prefix}.config.batchSize.description`,
          },
          title: 'Batch size',
          description: `Maximum ${resource} rows to delete in one cleanup batch.`,
          default: 1000,
        },
      },
      additionalProperties: false,
    });

  return {
    items: [
      {
        key: 'httpx.access-log-retention-cleanup',
        owner: 'core.httpx',
        module: 'core.httpx',
        display_name_key: 'scheduledTask.accessLogRetention.title',
        description_key: 'scheduledTask.accessLogRetention.description',
        title: 'Access log retention cleanup',
        description: 'Deletes access logs beyond the configured retention window.',
        config_schema_json: configSchemaJson('accessLogRetention', 'access logs'),
        default_config_json: '{"retentionDays":30,"batchSize":1000}',
        default_cron_expression: '*/5 * * * *',
        default_enabled: true,
        actions: [
          {
            key: 'dryRun',
            title_key: 'scheduledTask.action.dryRun.title',
            title: '试运行',
            description_key: 'scheduledTask.action.dryRun.description',
            description: '预览本次执行结果',
            affected_resource: 'access_logs',
            confirm_required: true,
            theme: 'primary',
          },
        ],
      },
      {
        key: 'logger.app-log-retention-cleanup',
        owner: 'core.logger',
        module: 'core.logger',
        display_name_key: 'scheduledTask.appLogRetention.title',
        description_key: 'scheduledTask.appLogRetention.description',
        title: 'App log retention cleanup',
        description: 'Deletes app logs beyond the configured retention window.',
        config_schema_json: configSchemaJson('appLogRetention', 'app logs'),
        default_config_json: '{"retentionDays":30,"batchSize":1000}',
        default_cron_expression: '*/5 * * * *',
        default_enabled: true,
        actions: [
          {
            key: 'dryRun',
            title_key: 'scheduledTask.action.dryRun.title',
            title: '试运行',
            description_key: 'scheduledTask.action.dryRun.description',
            description: 'Preview cleanup without deleting app logs.',
            behavior: '本次将预演清理应用日志，不会真正删除数据。',
            affected_resource: 'app_logs',
            confirm_required: true,
            theme: 'primary',
          },
        ],
      },
      {
        key: 'audit.audit-log-retention-cleanup',
        owner: 'audit',
        module: 'audit',
        display_name_key: 'scheduledTask.auditLogRetention.title',
        description_key: 'scheduledTask.auditLogRetention.description',
        title: 'Audit log retention cleanup',
        description: 'Deletes audit logs beyond the configured retention window.',
        config_schema_json: configSchemaJson('auditLogRetention', 'audit logs'),
        default_config_json: '{"retentionDays":30,"batchSize":1000}',
        default_cron_expression: '*/5 * * * *',
        default_enabled: true,
        actions: [
          {
            key: 'dryRun',
            title_key: 'scheduledTask.action.dryRun.title',
            title: '试运行',
            description_key: 'scheduledTask.action.dryRun.description',
            description: 'Preview cleanup without deleting audit logs.',
            behavior: '本次将预演清理审计日志，不会真正删除数据。',
            affected_resource: 'audit_logs',
            confirm_required: true,
            theme: 'primary',
          },
        ],
      },
    ],
    total: 3,
  };
}

const AdvancedQueryListPageStub = defineComponent({
  name: 'AdvancedQueryListPage',
  setup(_props, { slots }) {
    return () =>
      h('section', [
        h('header', [slots.eyebrow?.(), slots.actions?.()]),
        slots['feedback-extra']?.(),
        slots.filters?.(),
        slots.table?.(),
        slots.detail?.(),
      ]);
  },
});

const ColumnDrawerStub = defineComponent({
  name: 'AdvancedQueryColumnDrawer',
  props: ['columns', 'selectedKeys', 'title', 'visible'],
  emits: ['update:selectedKeys', 'update:visible'],
  setup(props, { emit }) {
    return () =>
      h('aside', { 'data-testid': 'column-drawer' }, [
        h('strong', props.title),
        h(
          'button',
          {
            'data-testid': 'hide-recent-run',
            onClick: () => emit('update:selectedKeys', ['task', 'job_key', 'status', 'schedule', 'recent_result']),
          },
          'hide recent run',
        ),
        h('span', JSON.stringify(props.columns)),
      ]);
  },
});

const TableStub = defineComponent({
  name: 'TTable',
  props: ['columns', 'data'],
  setup(props, { slots }) {
    return () =>
      h('table', [
        h(
          'thead',
          h(
            'tr',
            props.columns.map((column: { colKey: string; title: string }) =>
              h('th', { 'data-col': column.colKey }, column.title),
            ),
          ),
        ),
        h(
          'tbody',
          props.data.map((row: Record<string, unknown>) =>
            h(
              'tr',
              props.columns.map((column: { colKey: string }) =>
                h(
                  'td',
                  { 'data-col': column.colKey },
                  slots[column.colKey]?.({ row }) ?? String(row[column.colKey] ?? ''),
                ),
              ),
            ),
          ),
        ),
      ]);
  },
});

const ButtonStub = defineComponent({
  name: 'TButton',
  emits: ['click'],
  setup(_props, { emit, slots }) {
    return () => h('button', { onClick: (event: MouseEvent) => emit('click', event) }, slots.default?.());
  },
});

const InputStub = defineComponent({
  name: 'TInput',
  props: ['modelValue', 'placeholder'],
  emits: ['change', 'update:modelValue', 'input'],
  setup(props, { emit }) {
    return () =>
      h('input', {
        placeholder: props.placeholder,
        value: props.modelValue,
        onInput: (event: Event) => {
          const value = (event.target as HTMLInputElement).value;
          emit('update:modelValue', value);
          emit('input', value);
          emit('change', value);
        },
      });
  },
});

const InputNumberStub = defineComponent({
  name: 'TInputNumber',
  props: ['max', 'min', 'modelValue', 'placeholder'],
  emits: ['change', 'update:modelValue', 'input'],
  setup(props, { emit }) {
    return () =>
      h('input', {
        'data-max': props.max,
        'data-min': props.min,
        'data-testid':
          props.placeholder === translations['scheduledTask.list.form.configNumberPlaceholder']
            ? 'config-number-input'
            : undefined,
        placeholder: props.placeholder,
        type: 'number',
        value: props.modelValue,
        onInput: (event: Event) => {
          const value = Number((event.target as HTMLInputElement).value);
          emit('update:modelValue', value);
          emit('input', value);
          emit('change', value, { type: 'input' });
        },
      });
  },
});

const TextareaStub = defineComponent({
  name: 'TTextarea',
  props: ['modelValue', 'placeholder'],
  emits: ['change', 'update:modelValue', 'input'],
  setup(props, { emit }) {
    return () =>
      h('textarea', {
        'data-testid':
          props.placeholder === translations['scheduledTask.list.form.configJsonPlaceholder']
            ? 'config-json-textarea'
            : undefined,
        placeholder: props.placeholder,
        value: props.modelValue,
        onInput: (event: Event) => {
          const value = (event.target as HTMLTextAreaElement).value;
          emit('update:modelValue', value);
          emit('input', value);
          emit('change', value);
        },
      });
  },
});

const DialogStub = defineComponent({
  name: 'TDialog',
  props: ['cancelBtn', 'confirmBtn', 'header', 'visible'],
  emits: ['confirm', 'close', 'update:visible'],
  setup(props, { attrs, emit, slots }) {
    return () => {
      if (props.visible === false) {
        return null;
      }
      return h('div', attrs, [
        props.header,
        h(
          'button',
          {
            'data-testid': 'dialog-close',
            onClick: () => {
              emit('update:visible', false);
              emit('close', { trigger: 'close-btn' });
            },
          },
          'close',
        ),
        slots.default?.(),
        props.confirmBtn === null
          ? null
          : h(
              'button',
              {
                onClick: () => emit('confirm'),
              },
              props.confirmBtn,
            ),
      ]);
    };
  },
});

const CronExpressionFieldStub = defineComponent({
  name: 'CronExpressionField',
  props: ['modelValue', 'error'],
  emits: ['update:modelValue', 'validate'],
  setup(props, { emit }) {
    return () =>
      h('label', { class: 'cron-editor-stub' }, [
        h('span', 'Cron 表达式'),
        h('input', {
          'data-testid': 'cron-editor-input',
          value: props.modelValue,
          onInput: (event: Event) => {
            const value = (event.target as HTMLInputElement).value;
            const normalized = value.trim().split(/\s+/).length === 5 ? `0 ${value.trim()}` : value.trim();
            emit('update:modelValue', normalized);
            emit('validate', { valid: true, normalizedExpression: normalized });
          },
        }),
        props.error ? h('span', { 'data-testid': 'cron-editor-error' }, String(props.error)) : null,
      ]);
  },
});

const PassthroughStub = defineComponent({
  name: 'PassthroughStub',
  props: ['header', 'help', 'label', 'tips'],
  setup(props, { attrs, slots }) {
    return () =>
      h('div', attrs, [
        props.header,
        props.label,
        props.help,
        props.tips,
        slots.content?.(),
        slots.default?.(),
        slots.footer?.(),
        slots.action?.(),
      ]);
  },
});

function mountPage() {
  return mount(ScheduledTaskListPage, {
    global: {
      directives: {
        permission: () => undefined,
      },
      stubs: {
        AddIcon: true,
        AdvancedQueryColumnDrawer: ColumnDrawerStub,
        AdvancedQueryListPage: AdvancedQueryListPageStub,
        BrowseIcon: true,
        DeleteIcon: true,
        EditIcon: true,
        EllipsisIcon: true,
        PauseIcon: true,
        PlayIcon: true,
        SearchIcon: true,
        CronExpressionField: CronExpressionFieldStub,
        TButton: ButtonStub,
        TCard: PassthroughStub,
        TCollapse: PassthroughStub,
        TCollapsePanel: PassthroughStub,
        TDescriptions: PassthroughStub,
        TDescriptionsItem: PassthroughStub,
        TDialog: DialogStub,
        TDropdown: PassthroughStub,
        TDropdownItem: PassthroughStub,
        TDropdownMenu: PassthroughStub,
        TDrawer: PassthroughStub,
        TEmpty: PassthroughStub,
        TForm: PassthroughStub,
        TFormItem: PassthroughStub,
        TInput: InputStub,
        TInputNumber: InputNumberStub,
        TOption: PassthroughStub,
        TOptionGroup: PassthroughStub,
        TRadioButton: PassthroughStub,
        TRadioGroup: PassthroughStub,
        TSelect: PassthroughStub,
        TSpace: PassthroughStub,
        TSwitch: PassthroughStub,
        TAlert: PassthroughStub,
        TTable: TableStub,
        TTag: PassthroughStub,
        TTextarea: TextareaStub,
        TTooltip: PassthroughStub,
      },
    },
  });
}

function findButtonByText(wrapper: ReturnType<typeof mountPage>, text: string) {
  return wrapper.findAll('button').find((button) => button.text() === text);
}

async function openFirstTaskEditDrawer(wrapper: ReturnType<typeof mountPage>) {
  const editTrigger = wrapper.findAll('*').find((node) => node.text() === '编辑');
  expect(editTrigger).toBeTruthy();
  await editTrigger!.trigger('click');
  await flushPromises();
}

async function openTaskEditDrawerByRow(wrapper: ReturnType<typeof mountPage>, rowIndex: number) {
  const operationCell = wrapper.find(`tbody tr:nth-child(${rowIndex}) td[data-col="operation"]`);
  const editTrigger = operationCell.findAll('*').find((node) => node.text() === '编辑');
  expect(editTrigger).toBeTruthy();
  await editTrigger!.trigger('click');
  await flushPromises();
}

async function openConfigDialog(wrapper: ReturnType<typeof mountPage>) {
  const configTrigger = findButtonByText(wrapper, '配置');
  expect(configTrigger).toBeTruthy();
  await configTrigger!.trigger('click');
  await flushPromises();
}

describe('ScheduledTaskListPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    apiMocks.getScheduledTasks.mockResolvedValue(scheduledTasksResponse());
    apiMocks.getScheduledTask.mockImplementation(async (taskKey: string) => {
      const task = scheduledTasksResponse().items.find((item) => item.key === taskKey);
      if (!task) {
        throw new Error('not found');
      }
      return task;
    });
    apiMocks.getScheduledTaskJobDefinitions.mockResolvedValue(jobDefinitionsResponse());
    apiMocks.getScheduledTaskJobDefinition.mockImplementation(async (jobKey: string) => {
      const job = jobDefinitionsResponse().items.find((item) => item.key === jobKey);
      if (!job) {
        throw new Error('not found');
      }
      return job;
    });
    apiMocks.getScheduledTaskRuns.mockResolvedValue({ items: [], total: 0, limit: 20, offset: 0 });
    apiMocks.updateScheduledTask.mockImplementation(async (taskKey: string, payload: Record<string, unknown>) => ({
      ...scheduledTasksResponse().items[0],
      key: taskKey,
      schedule: payload.cron_expression,
      enabled: payload.enabled,
      config_json: String(payload.config_json ?? '{}'),
    }));
    apiMocks.executeScheduledTaskAction.mockResolvedValue({
      action_key: 'dryRun',
      result_json:
        '{"summary":"预计可清理 128 条访问日志","stage":"estimated","affected_resource":"access_logs","metrics":{"estimatedScanCount":128,"estimatedDeleteCount":128,"estimatedRetainCount":32},"details":{"retentionDays":30,"batchSize":500},"warnings":[]}',
    });
  });

  it('localizes builtin task and job definition title keys before rendering', async () => {
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain('访问日志保留清理');
    expect(wrapper.text()).toContain('应用日志保留清理');
    expect(wrapper.text()).toContain('审计日志保留清理');
    expect(wrapper.text()).not.toContain('Access log retention cleanup');
    expect(wrapper.text()).toContain('Custom cleanup');
  });

  it('keeps operation column visible while column settings hide optional columns', async () => {
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.find('th[data-col="recent_run"]').exists()).toBe(true);
    expect(wrapper.find('th[data-col="operation"]').exists()).toBe(true);

    await wrapper.find('[data-testid="hide-recent-run"]').trigger('click');
    await nextTick();

    expect(wrapper.find('th[data-col="recent_run"]').exists()).toBe(false);
    expect(wrapper.find('th[data-col="operation"]').exists()).toBe(true);
  });

  it('keeps high-frequency actions visible and folds management actions into more menu', async () => {
    const wrapper = mountPage();
    await flushPromises();

    const firstOperationCell = wrapper.find('tbody tr:first-child td[data-col="operation"]');
    expect(firstOperationCell.exists()).toBe(true);
    expect(firstOperationCell.text()).toContain('查看');
    expect(firstOperationCell.text()).toContain('立即执行');
    expect(firstOperationCell.text()).toContain('更多');
    expect(firstOperationCell.text()).toContain('编辑');
    expect(firstOperationCell.text()).toContain('停用');
    expect(firstOperationCell.text()).toContain('删除');
  });

  it('renders raw cron expressions, next run diagnostics, and recent result summaries in list cells', async () => {
    const wrapper = mountPage();
    await flushPromises();

    const firstScheduleCell = wrapper.find('tbody tr:first-child td[data-col="schedule"]');
    expect(firstScheduleCell.text()).toContain('*/5 * * * *');
    expect(firstScheduleCell.text()).toContain('下次执行：2026-06-06 08:05');
    expect(firstScheduleCell.find('.scheduled-task-schedule').text()).not.toContain('规则说明');
    expect(firstScheduleCell.text()).toContain('规则说明');
    expect(firstScheduleCell.text()).toContain('每隔 5 分钟');
    expect(firstScheduleCell.text()).toContain('时区');

    const customScheduleCell = wrapper.find('tbody tr:nth-child(4) td[data-col="schedule"]');
    expect(customScheduleCell.text()).toContain('0 17 * * *');
    expect(customScheduleCell.text()).toContain('下次执行：2026-06-06 17:00');
    expect(customScheduleCell.find('.scheduled-task-schedule').text()).not.toContain('规则说明');
    expect(customScheduleCell.text()).toContain('每天 17:00 执行');
    expect(customScheduleCell.text()).not.toContain('在17:00, 每天');

    const firstResultCell = wrapper.find('tbody tr:first-child td[data-col="recent_result"]');
    expect(firstResultCell.text()).toContain('成功');
    expect(firstResultCell.text()).toContain('Deleted 3 access log rows.');
    expect(firstResultCell.text()).not.toContain('成功无');

    const secondResultCell = wrapper.find('tbody tr:nth-child(2) td[data-col="recent_result"]');
    expect(secondResultCell.text()).toContain('失败');
    expect(secondResultCell.text()).toContain('retention window is invalid');
  });

  it('normalizes cron editor values before submitting an update payload', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await openFirstTaskEditDrawer(wrapper);

    await wrapper.get('[data-testid="cron-editor-input"]').setValue('*/10 * * * *');
    await wrapper
      .findAll('button')
      .find((button) => button.text() === '保存')!
      .trigger('click');
    await flushPromises();

    expect(apiMocks.updateScheduledTask).toHaveBeenCalledWith(
      'httpx.access-log-retention-cleanup',
      expect.objectContaining({
        cron_expression: '0 */10 * * * *',
      }),
    );
    expect(apiMocks.updateScheduledTask).toHaveBeenCalledWith(
      'httpx.access-log-retention-cleanup',
      expect.not.objectContaining({
        config_json: expect.any(String),
      }),
    );
  });

  it('keeps persistent config fields in the edit drawer and renders backend action buttons', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await openFirstTaskEditDrawer(wrapper);

    expect(wrapper.text()).toContain('任务配置');
    expect(wrapper.text()).toContain('批量大小');
    expect(wrapper.text()).toContain('单次清理最多删除的访问日志行数。');
    expect(wrapper.text()).not.toContain('此 Job Definition 暂无可配置项。');
    expect(wrapper.text()).toContain('Job 操作');
    expect(wrapper.text()).toContain('试运行');

    const configSectionText = wrapper.findAll('.scheduled-task-form-section')[3]?.text() ?? '';
    expect(configSectionText).toContain('任务配置');
    expect(configSectionText).toContain('日志保留时间');
    expect(configSectionText).toContain('批量大小');
    expect(configSectionText).not.toContain('试运行');
  });

  it('uses Job Definition default config when editing a task without task config', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await openTaskEditDrawerByRow(wrapper, 2);
    await openConfigDialog(wrapper);

    expect(wrapper.text()).toContain('"retentionDays": 30');
    expect(wrapper.text()).toContain('"batchSize": 1000');
  });

  it('does not persist Job Definition defaults when task config is unchanged', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await openTaskEditDrawerByRow(wrapper, 2);
    await findButtonByText(wrapper, '保存')!.trigger('click');
    await flushPromises();

    expect(apiMocks.updateScheduledTask).toHaveBeenCalledWith(
      'logger.app-log-retention-cleanup',
      expect.not.objectContaining({
        config_json: '{"retentionDays":30,"batchSize":1000}',
      }),
    );
    expect(apiMocks.updateScheduledTask).toHaveBeenCalledWith(
      'logger.app-log-retention-cleanup',
      expect.not.objectContaining({
        config_json: expect.any(String),
      }),
    );
  });

  it('renders schema config labels from x-i18n in the form surface', async () => {
    const wrapper = mountPage();
    await flushPromises();

    const viewTrigger = wrapper.findAll('*').find((node) => node.text() === '查看');
    expect(viewTrigger).toBeTruthy();
    await viewTrigger!.trigger('click');
    await flushPromises();

    expect(wrapper.text()).toContain('日志保留时间');
    expect(wrapper.text()).toContain('批量大小');
    expect(wrapper.text()).toContain('单次清理最多删除的访问日志行数。');
  });

  it('executes dry-run actions through the action endpoint without persisting dryRun', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await openFirstTaskEditDrawer(wrapper);

    const actionTrigger = wrapper.findAll('button').find((button) => button.text() === '试运行');
    expect(actionTrigger).toBeTruthy();
    await actionTrigger!.trigger('click');
    await flushPromises();

    expect(wrapper.text()).toContain('确认操作');
    expect(wrapper.text()).toContain('预览本次执行结果');
    expect(wrapper.text()).toContain('access_logs');

    await wrapper
      .findAll('button')
      .find((button) => button.text() === '执行操作')!
      .trigger('click');
    await flushPromises();

    expect(apiMocks.executeScheduledTaskAction).toHaveBeenCalledWith('httpx.access-log-retention-cleanup', 'dryRun', {
      config_json: { retentionDays: 30, batchSize: 500 },
    });
    expect(apiMocks.updateScheduledTask).not.toHaveBeenCalled();
    expect(wrapper.text()).toContain('操作结果');
    expect(wrapper.text()).toContain('预计可清理 128 条访问日志');
    expect(wrapper.text()).toContain('estimated');
  });

  it('blocks dry-run action execution when schema maximum is exceeded', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await openFirstTaskEditDrawer(wrapper);
    await openConfigDialog(wrapper);
    await findButtonByText(wrapper, '编辑 JSON')!.trigger('click');
    await nextTick();
    await wrapper.get('[data-testid="config-json-textarea"]').setValue('{"retentionDays":30,"batchSize":100000}');

    const actionTrigger = wrapper.findAll('button').find((button) => button.text() === '试运行');
    expect(actionTrigger).toBeTruthy();
    await actionTrigger!.trigger('click');
    await flushPromises();
    await wrapper
      .findAll('button')
      .find((button) => button.text() === '执行操作')!
      .trigger('click');
    await flushPromises();

    expect(apiMocks.executeScheduledTaskAction).not.toHaveBeenCalled();
    expect(wrapper.text()).toContain('批量大小不能超过 10000');
  });

  it('opens the config dialog in JSON preview mode without showing the editor textarea', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await openFirstTaskEditDrawer(wrapper);
    await openConfigDialog(wrapper);

    expect(wrapper.text()).toContain('JSON 预览');
    expect(wrapper.text()).toContain('编辑 JSON');
    expect(wrapper.text()).not.toContain('格式化 JSON');
    expect(wrapper.find('[data-testid="config-json-textarea"]').exists()).toBe(false);
    expect(wrapper.text()).toContain('"retentionDays": 30');
    expect(wrapper.text()).toContain('"batchSize": 500');
  });

  it('switches advanced config JSON into edit mode and formats the textarea', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await openFirstTaskEditDrawer(wrapper);
    await openConfigDialog(wrapper);

    await findButtonByText(wrapper, '编辑 JSON')!.trigger('click');
    await nextTick();

    expect(wrapper.text()).toContain('完成');
    const textarea = wrapper.get('[data-testid="config-json-textarea"]');
    await textarea.setValue('{"retentionDays":45,"batchSize":250}');

    await findButtonByText(wrapper, '格式化 JSON')!.trigger('click');
    await nextTick();

    expect(wrapper.get('[data-testid="config-json-textarea"]').element).toHaveProperty(
      'value',
      '{\n  "retentionDays": 45,\n  "batchSize": 250\n}',
    );

    await findButtonByText(wrapper, '完成')!.trigger('click');
    await nextTick();

    expect(wrapper.find('[data-testid="config-json-textarea"]').exists()).toBe(false);
    expect(wrapper.text()).toContain('"retentionDays": 45');
  });

  it('sanitizes config save payload to the selected Job Definition schema', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await openFirstTaskEditDrawer(wrapper);
    await openConfigDialog(wrapper);
    await findButtonByText(wrapper, '编辑 JSON')!.trigger('click');
    await nextTick();

    await wrapper
      .get('[data-testid="config-json-textarea"]')
      .setValue('{"retentionDays":45,"batchSize":250,"dryRun":true,"unknown":"stale"}');

    await findButtonByText(wrapper, '保存配置')!.trigger('click');
    await flushPromises();

    expect(apiMocks.updateScheduledTask).toHaveBeenLastCalledWith('httpx.access-log-retention-cleanup', {
      config_json: '{"retentionDays":45,"batchSize":250}',
    });
  });

  it('blocks config save when schema maximum is exceeded', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await openFirstTaskEditDrawer(wrapper);
    await openConfigDialog(wrapper);
    await findButtonByText(wrapper, '编辑 JSON')!.trigger('click');
    await nextTick();

    await wrapper.get('[data-testid="config-json-textarea"]').setValue('{"retentionDays":45,"batchSize":100000}');
    await findButtonByText(wrapper, '保存配置')!.trigger('click');
    await flushPromises();

    expect(apiMocks.updateScheduledTask).not.toHaveBeenCalled();
    expect(wrapper.text()).toContain('批量大小不能超过 10000');
  });

  it('blocks config save when a basic numeric config input exceeds the schema maximum', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await openFirstTaskEditDrawer(wrapper);
    await openConfigDialog(wrapper);

    const numberInputs = wrapper.findAll('[data-testid="config-number-input"]');
    const batchSizeInput = numberInputs.find((input) => input.attributes('data-max') === '10000');
    expect(batchSizeInput).toBeTruthy();

    await batchSizeInput!.setValue('2000000');
    await findButtonByText(wrapper, '保存配置')!.trigger('click');
    await flushPromises();

    expect(apiMocks.updateScheduledTask).not.toHaveBeenCalled();
    expect(wrapper.text()).toContain('批量大小不能超过 10000');
  });

  it('blocks config save when a range-valid numeric enum value is not allowed', async () => {
    const definitions = jobDefinitionsResponse();
    const firstDefinition = definitions.items[0];
    const schema = JSON.parse(firstDefinition.config_schema_json) as {
      properties: Record<string, { enum?: number[] }>;
    };
    schema.properties.batchSize.enum = [1000, 2000];
    firstDefinition.config_schema_json = JSON.stringify(schema);
    apiMocks.getScheduledTaskJobDefinitions.mockResolvedValueOnce(definitions);
    apiMocks.getScheduledTaskJobDefinition.mockResolvedValueOnce(firstDefinition);

    const wrapper = mountPage();
    await flushPromises();

    await openFirstTaskEditDrawer(wrapper);
    await openConfigDialog(wrapper);
    await findButtonByText(wrapper, '编辑 JSON')!.trigger('click');
    await nextTick();

    await wrapper.get('[data-testid="config-json-textarea"]').setValue('{"retentionDays":45,"batchSize":500}');
    await findButtonByText(wrapper, '保存配置')!.trigger('click');
    await flushPromises();

    expect(apiMocks.updateScheduledTask).not.toHaveBeenCalled();
    expect(wrapper.text()).toContain('批量大小必须是以下值之一：1000, 2000');
  });

  it('preserves in-drawer cron edits after saving config only', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await openFirstTaskEditDrawer(wrapper);
    await wrapper.get('[data-testid="cron-editor-input"]').setValue('*/10 * * * *');
    await openConfigDialog(wrapper);
    await findButtonByText(wrapper, '编辑 JSON')!.trigger('click');
    await nextTick();
    await wrapper.get('[data-testid="config-json-textarea"]').setValue('{"retentionDays":45,"batchSize":250}');

    await findButtonByText(wrapper, '保存配置')!.trigger('click');
    await flushPromises();
    await findButtonByText(wrapper, '保存')!.trigger('click');
    await flushPromises();

    expect(apiMocks.updateScheduledTask).toHaveBeenNthCalledWith(1, 'httpx.access-log-retention-cleanup', {
      config_json: '{"retentionDays":45,"batchSize":250}',
    });
    expect(apiMocks.updateScheduledTask).toHaveBeenLastCalledWith(
      'httpx.access-log-retention-cleanup',
      expect.objectContaining({
        cron_expression: '0 */10 * * * *',
      }),
    );
    expect(apiMocks.updateScheduledTask).toHaveBeenLastCalledWith(
      'httpx.access-log-retention-cleanup',
      expect.not.objectContaining({
        config_json: expect.any(String),
      }),
    );
  });

  it('closes the action result dialog from the visible confirm button and clears the result state', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await openFirstTaskEditDrawer(wrapper);

    const actionTrigger = findButtonByText(wrapper, '试运行');
    expect(actionTrigger).toBeTruthy();
    await actionTrigger!.trigger('click');
    await flushPromises();
    await findButtonByText(wrapper, '执行操作')!.trigger('click');
    await flushPromises();

    expect(wrapper.text()).toContain('操作结果');
    expect(wrapper.text()).toContain('预计可清理 128 条访问日志');

    await findButtonByText(wrapper, '关闭')!.trigger('click');
    await flushPromises();

    expect(wrapper.text()).not.toContain('操作结果');
    expect(wrapper.text()).not.toContain('预计可清理 128 条访问日志');
  });
});
