import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, nextTick } from 'vue';

import ScheduledTaskListPage from './index.vue';

const apiMocks = vi.hoisted(() => ({
  createScheduledTask: vi.fn(),
  getScheduledTask: vi.fn(),
  getScheduledTaskJobs: vi.fn(),
  getScheduledTaskRuns: vi.fn(),
  getScheduledTasks: vi.fn(),
  updateScheduledTask: vi.fn(),
}));

const translations = vi.hoisted(
  (): Record<string, string> => ({
    'scheduledTask.accessLogRetention.description': '删除超过配置保留窗口的访问日志。',
    'scheduledTask.accessLogRetention.title': '访问日志保留清理',
    'scheduledTask.appLogRetention.description': '删除超过配置保留窗口的应用日志。',
    'scheduledTask.appLogRetention.title': '应用日志保留清理',
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
    'scheduledTask.list.form.formatJson': '格式化 JSON',
    'scheduledTask.list.form.cronRequiredHint': '请填写 Cron 表达式。',
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
  getScheduledTask: apiMocks.getScheduledTask,
  getScheduledTaskJobs: apiMocks.getScheduledTaskJobs,
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
        params_json: '{}',
        last_run: {
          id: 101,
          trigger_type: 'cron',
          status: 'success',
          started_at: '2026-06-05T00:00:00Z',
          finished_at: '2026-06-05T00:00:05Z',
          duration_ms: 5000,
          error_summary: '',
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
        params_json: '{}',
        last_run: {
          id: 102,
          trigger_type: 'cron',
          status: 'failed',
          started_at: '2026-06-05T00:10:00Z',
          finished_at: '2026-06-05T00:10:01Z',
          duration_ms: 1000,
          error_summary: 'retention window is invalid',
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
        params_json: '{}',
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
        params_json: '{}',
      },
    ],
    total: 4,
  };
}

function jobDefinitionsResponse() {
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
        params_schema_json: '{}',
        default_params_json: '{}',
        default_cron_expression: '*/5 * * * *',
        default_enabled: true,
      },
      {
        key: 'logger.app-log-retention-cleanup',
        owner: 'core.logger',
        module: 'core.logger',
        display_name_key: 'scheduledTask.appLogRetention.title',
        description_key: 'scheduledTask.appLogRetention.description',
        title: 'App log retention cleanup',
        description: 'Deletes app logs beyond the configured retention window.',
        params_schema_json: '{}',
        default_params_json: '{}',
        default_cron_expression: '*/5 * * * *',
        default_enabled: true,
      },
      {
        key: 'audit.audit-log-retention-cleanup',
        owner: 'audit',
        module: 'audit',
        display_name_key: 'scheduledTask.auditLogRetention.title',
        description_key: 'scheduledTask.auditLogRetention.description',
        title: 'Audit log retention cleanup',
        description: 'Deletes audit logs beyond the configured retention window.',
        params_schema_json: '{}',
        default_params_json: '{}',
        default_cron_expression: '*/5 * * * *',
        default_enabled: true,
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
  emits: ['update:modelValue', 'input'],
  setup(props, { emit }) {
    return () =>
      h('input', {
        placeholder: props.placeholder,
        value: props.modelValue,
        onInput: (event: Event) => {
          const value = (event.target as HTMLInputElement).value;
          emit('update:modelValue', value);
          emit('input', value);
        },
      });
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
  props: ['header', 'label'],
  setup(props, { attrs, slots }) {
    return () =>
      h('div', attrs, [
        props.header,
        props.label,
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
        TDialog: PassthroughStub,
        TDropdown: PassthroughStub,
        TDropdownItem: PassthroughStub,
        TDropdownMenu: PassthroughStub,
        TDrawer: PassthroughStub,
        TEmpty: PassthroughStub,
        TForm: PassthroughStub,
        TFormItem: PassthroughStub,
        TInput: InputStub,
        TOption: PassthroughStub,
        TOptionGroup: PassthroughStub,
        TRadioButton: PassthroughStub,
        TRadioGroup: PassthroughStub,
        TSelect: PassthroughStub,
        TSpace: PassthroughStub,
        TSwitch: PassthroughStub,
        TTable: TableStub,
        TTag: PassthroughStub,
        TTextarea: InputStub,
        TTooltip: PassthroughStub,
      },
    },
  });
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
    apiMocks.getScheduledTaskJobs.mockResolvedValue(jobDefinitionsResponse());
    apiMocks.getScheduledTaskRuns.mockResolvedValue({ items: [], total: 0, limit: 20, offset: 0 });
    apiMocks.updateScheduledTask.mockImplementation(async (taskKey: string, payload: Record<string, unknown>) => ({
      ...scheduledTasksResponse().items[0],
      key: taskKey,
      schedule: payload.cron_expression,
      enabled: payload.enabled,
    }));
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
    expect(firstResultCell.text()).toContain('未记录错误');
    expect(firstResultCell.text()).not.toContain('成功无');

    const secondResultCell = wrapper.find('tbody tr:nth-child(2) td[data-col="recent_result"]');
    expect(secondResultCell.text()).toContain('失败');
    expect(secondResultCell.text()).toContain('retention window is invalid');
  });

  it('normalizes cron editor values before submitting an update payload', async () => {
    const wrapper = mountPage();
    await flushPromises();

    const editTrigger = wrapper.findAll('*').find((node) => node.text() === '编辑');
    expect(editTrigger).toBeTruthy();
    await editTrigger!.trigger('click');
    await flushPromises();

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
  });
});
