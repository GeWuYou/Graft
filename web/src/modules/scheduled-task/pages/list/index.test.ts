import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, nextTick } from 'vue';

import ScheduledTaskListPage from './index.vue';

const apiMocks = vi.hoisted(() => ({
  getScheduledTaskJobs: vi.fn(),
  getScheduledTaskRuns: vi.fn(),
  getScheduledTasks: vi.fn(),
}));

const translations = vi.hoisted(
  (): Record<string, string> => ({
    'scheduledTask.accessLogRetention.description': '删除超过配置保留窗口的访问日志。',
    'scheduledTask.accessLogRetention.title': '访问日志保留清理',
    'scheduledTask.auditLogRetention.description': '删除超过配置保留窗口的审计日志。',
    'scheduledTask.auditLogRetention.title': '审计日志保留清理',
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
    'scheduledTask.list.cronDescription.every5Minutes': '表达式：*/5 * * * *',
    'scheduledTask.list.cronHint.every5Minutes': '每 5 分钟执行一次。',
    'scheduledTask.list.cronMode.custom': '自定义',
    'scheduledTask.list.cronMode.daily': '每天',
    'scheduledTask.list.cronMode.every5Minutes': '每 5 分钟',
    'scheduledTask.list.cronMode.everyMinute': '每分钟',
    'scheduledTask.list.cronMode.hourly': '每小时',
    'scheduledTask.list.cronMode.monthly': '每月',
    'scheduledTask.list.cronMode.weekly': '每周',
    'scheduledTask.list.create': '新建任务',
    'scheduledTask.list.description': '管理绑定到 Job Definition 的定时任务。',
    'scheduledTask.list.detail.none': '无',
    'scheduledTask.list.eyebrow': '服务管理',
    'scheduledTask.list.filters.allJobTypes': '全部 Job 类型',
    'scheduledTask.list.filters.allStatuses': '全部状态',
    'scheduledTask.list.filters.jobType': 'Job 类型',
    'scheduledTask.list.filters.searchPlaceholder': '搜索任务',
    'scheduledTask.list.filters.status': '状态',
    'scheduledTask.list.form.cronExpression': 'Cron Expression',
    'scheduledTask.list.form.cronExpressionPlaceholder': '例如 */5 * * * *',
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
    'scheduledTask.list.refresh': '刷新',
    'scheduledTask.list.status.idle': '空闲',
    'scheduledTask.list.tableHint': '当前筛选显示 {count} 个任务。',
    'scheduledTask.list.tableTitle': '任务列表',
    'scheduledTask.list.title': '定时任务',
    'scheduledTask.list.viewDetail': '查看',
  }),
);

vi.mock('../../api/scheduled-task', () => ({
  createScheduledTask: vi.fn(),
  deleteScheduledTask: vi.fn(),
  disableScheduledTask: vi.fn(),
  enableScheduledTask: vi.fn(),
  getScheduledTask: vi.fn(),
  getScheduledTaskJobs: apiMocks.getScheduledTaskJobs,
  getScheduledTaskRun: vi.fn(),
  getScheduledTaskRuns: apiMocks.getScheduledTaskRuns,
  getScheduledTasks: apiMocks.getScheduledTasks,
  runScheduledTask: vi.fn(),
  updateScheduledTask: vi.fn(),
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
        title: 'scheduledTask.accessLogRetention.title',
        description: 'scheduledTask.accessLogRetention.description',
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
        schedule: '0 * * * *',
        status: 'idle',
        running: false,
        params_json: '{}',
      },
    ],
    total: 2,
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
        title: 'scheduledTask.accessLogRetention.title',
        description: 'scheduledTask.accessLogRetention.description',
        params_schema_json: '{}',
        default_params_json: '{}',
        default_cron_expression: '*/5 * * * *',
        default_enabled: true,
      },
    ],
    total: 1,
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

const PassthroughStub = defineComponent({
  name: 'PassthroughStub',
  props: ['header', 'label'],
  setup(props, { slots }) {
    return () => h('div', [props.header, props.label, slots.default?.(), slots.footer?.(), slots.action?.()]);
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
        PlayIcon: true,
        SearchIcon: true,
        TButton: ButtonStub,
        TCard: PassthroughStub,
        TCollapse: PassthroughStub,
        TCollapsePanel: PassthroughStub,
        TDescriptions: PassthroughStub,
        TDescriptionsItem: PassthroughStub,
        TDialog: PassthroughStub,
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
      },
    },
  });
}

describe('ScheduledTaskListPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    apiMocks.getScheduledTasks.mockResolvedValue(scheduledTasksResponse());
    apiMocks.getScheduledTaskJobs.mockResolvedValue(jobDefinitionsResponse());
    apiMocks.getScheduledTaskRuns.mockResolvedValue({ items: [], total: 0, limit: 20, offset: 0 });
  });

  it('localizes builtin task and job definition title keys before rendering', async () => {
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain('访问日志保留清理');
    expect(wrapper.text()).not.toContain('scheduledTask.accessLogRetention.title');
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

  it('renders cron expression hint below the input inside a vertical container', async () => {
    const wrapper = mountPage();
    await flushPromises();

    const container = wrapper.find('.scheduled-task-cron-expression');
    expect(container.exists()).toBe(true);
    expect(container.find('input[placeholder="例如 */5 * * * *"]').exists()).toBe(true);
    expect(container.text()).toContain('表达式：*/5 * * * *');
  });
});
