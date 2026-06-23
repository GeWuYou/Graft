import { flushPromises, mount } from '@vue/test-utils';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import ModulesPage from './index.vue';

const moduleRuntimeApiMocks = vi.hoisted(() => ({
  getModuleRuntimeDetail: vi.fn(),
  getModuleRuntimeSnapshot: vi.fn(),
}));

const loggerMocks = vi.hoisted(() => ({
  error: vi.fn(),
}));

const routeMocks = vi.hoisted(() => ({
  route: {
    path: '/server/modules',
    fullPath: '/server/modules',
  },
}));

const tabsRouterStoreMock = vi.hoisted(() => ({
  activeTabKey: '/server/modules',
  tabRouters: [
    {
      path: '/server/modules',
      fullPath: '/server/modules',
      tabKey: '/server/modules',
    },
  ],
}));

const translations = vi.hoisted(
  (): Record<string, string> => ({
    'app.refreshControl.labels.interval': '自动刷新：',
    'app.refreshControl.labels.trendWindow': '趋势窗口：',
    'app.refreshControl.status.running': '自动刷新：{interval}',
    'app.refreshControl.status.paused': '自动刷新已暂停',
    'app.refreshControl.status.off': '自动刷新关闭',
    'app.refreshControl.countdown': '{countdown} 后刷新',
    'app.refreshControl.pending': '等待下次刷新',
    'app.refreshControl.actions.refresh': 'Refresh',
    'app.refreshControl.actions.pause': 'Pause auto refresh',
    'app.refreshControl.actions.resume': 'Resume auto refresh',
    'app.refreshControl.actions.enable': 'Enable auto refresh',
    'app.refreshControl.actions.pauseCompact': 'Pause',
    'app.refreshControl.actions.resumeCompact': 'Resume',
    'app.refreshControl.actions.enableCompact': 'Enable',
    'monitor.serverStatus.refreshInterval5Seconds': 'Every 5 sec',
    'monitor.serverStatus.refreshInterval10Seconds': 'Every 10 sec',
    'monitor.serverStatus.refreshInterval30Seconds': 'Every 30 sec',
    'monitor.serverStatus.refreshInterval1Minute': 'Every 1 min',
    'monitor.sectionTitle': 'Service Management',
    'monitor.moduleRuntime.title': 'Modules',
    'monitor.moduleRuntime.subtitle': 'Review compile-time module status.',
    'monitor.moduleRuntime.errorTitle': 'Module snapshot request failed',
    'monitor.moduleRuntime.errorFallback': 'Failed to load module runtime snapshot',
    'monitor.moduleRuntime.empty': 'No module runtime data',
    'monitor.moduleRuntime.status.ready': 'Ready',
    'monitor.moduleRuntime.status.attention': 'Needs attention',
    'monitor.moduleRuntime.status.unknown': 'Unknown',
    'monitor.moduleRuntime.actions.refresh': 'Refresh',
    'monitor.moduleRuntime.actions.detail': 'Detail',
    'monitor.moduleRuntime.summary.total': 'Total',
    'monitor.moduleRuntime.summary.totalDescription': 'Modules known to the runtime registry',
    'monitor.moduleRuntime.summary.enabled': 'Enabled',
    'monitor.moduleRuntime.summary.enabledDescription': 'Modules enabled for this process',
    'monitor.moduleRuntime.summary.healthy': 'Healthy',
    'monitor.moduleRuntime.summary.healthyDescription': 'Modules reporting healthy runtime status',
    'monitor.moduleRuntime.summary.degradedUnknown': 'Degraded / unknown',
    'monitor.moduleRuntime.summary.degradedUnknownDescription': 'Modules needing operator attention',
    'monitor.moduleRuntime.table.title': 'Runtime Module List',
    'monitor.moduleRuntime.table.description': 'Read-only view of the current process module registry.',
    'monitor.moduleRuntime.table.note': 'Read-only view. Module write operations are unavailable.',
    'monitor.moduleRuntime.table.columnSettings': 'Column settings',
    'monitor.moduleRuntime.table.resetColumns': 'Restore default columns',
    'monitor.moduleRuntime.table.compactDensity': 'Compact density',
    'monitor.moduleRuntime.table.defaultDensity': 'Default density',
    'monitor.moduleRuntime.columns.moduleKey': 'Module key',
    'monitor.moduleRuntime.columns.enabled': 'Enabled',
    'monitor.moduleRuntime.columns.registered': 'Registered',
    'monitor.moduleRuntime.columns.health': 'Health',
    'monitor.moduleRuntime.columns.dependencies': 'Dependencies',
    'monitor.moduleRuntime.columns.resourceStatus': 'Resource status',
    'monitor.moduleRuntime.columns.migration': 'Migrations',
    'monitor.moduleRuntime.columns.schema': 'Schema',
    'monitor.moduleRuntime.columns.config': 'Config',
    'monitor.moduleRuntime.columns.action': 'Action',
    'monitor.moduleRuntime.detail.title': 'Module runtime detail',
    'monitor.moduleRuntime.detail.titleWithKey': '{key} runtime detail',
    'monitor.moduleRuntime.detail.basicInfo': 'Basic information',
    'monitor.moduleRuntime.detail.moduleKey': 'Module key',
    'monitor.moduleRuntime.detail.enabled': 'Enabled',
    'monitor.moduleRuntime.detail.registered': 'Registered',
    'monitor.moduleRuntime.detail.health': 'Health',
    'monitor.moduleRuntime.detail.runtimeStatus': 'Runtime status',
    'monitor.moduleRuntime.detail.enablementSource': 'Enablement source',
    'monitor.moduleRuntime.detail.dependencies': 'Dependencies',
    'monitor.moduleRuntime.detail.declaredDependencies': 'Declared dependencies',
    'monitor.moduleRuntime.detail.dependencySatisfaction': 'Satisfaction',
    'monitor.moduleRuntime.detail.migration': 'Migration',
    'monitor.moduleRuntime.detail.migrationDir': 'Migration Dir',
    'monitor.moduleRuntime.detail.migrationStatus': 'Status',
    'monitor.moduleRuntime.detail.schema': 'Schema',
    'monitor.moduleRuntime.detail.schemaOwner': 'Owner',
    'monitor.moduleRuntime.detail.schemaStatus': 'Status',
    'monitor.moduleRuntime.detail.config': 'Config',
    'monitor.moduleRuntime.detail.configStatus': 'Status',
    'monitor.moduleRuntime.detail.configDescription': 'Description',
    'monitor.moduleRuntime.detail.diagnostics': 'Diagnostics',
    'monitor.moduleRuntime.detail.rawJson': 'Raw runtime JSON',
    'monitor.moduleRuntime.values.yes': 'Yes',
    'monitor.moduleRuntime.values.no': 'No',
    'monitor.moduleRuntime.values.none': 'None',
    'monitor.moduleRuntime.values.emptyDependencies': 'No dependencies',
    'monitor.moduleRuntime.values.emptyMigrationDir': 'No migration directory',
    'monitor.moduleRuntime.values.emptySchema': 'Schema not declared',
    'monitor.moduleRuntime.values.unknownConfig': 'Unknown config status',
    'monitor.moduleRuntime.values.noDiagnostics': 'No diagnostics',
    'monitor.moduleRuntime.values.notReported': 'Not reported',
    'monitor.moduleRuntime.values.dependencySummary': '{satisfied} / {total} satisfied',
    'monitor.moduleRuntime.values.migrationDirCount': '{count} dirs',
    'monitor.moduleRuntime.values.moduleOwnedSchema': 'Module-owned',
    'monitor.moduleRuntime.values.notRequiredConfig': 'No config required',
    'monitor.moduleRuntime.health.healthy': 'Healthy',
    'monitor.moduleRuntime.health.degraded': 'Degraded',
    'monitor.moduleRuntime.health.unknown': 'Unknown',
    'monitor.moduleRuntime.health.disabled': 'Disabled',
    'monitor.moduleRuntime.runtimeStatus.registered': 'Registered',
    'monitor.moduleRuntime.runtimeStatus.disabled': 'Disabled',
    'monitor.moduleRuntime.runtimeStatus.degraded': 'Degraded',
    'monitor.moduleRuntime.runtimeStatus.unknown': 'Unknown',
    'monitor.moduleRuntime.migrationStatus.declared': 'Declared',
    'monitor.moduleRuntime.migrationStatus.not_declared': 'Not declared',
    'monitor.moduleRuntime.schemaStatus.declared': 'Declared',
    'monitor.moduleRuntime.schemaStatus.unknown': 'Unknown',
    'monitor.moduleRuntime.configStatus.not_required': 'Not required',
    'monitor.moduleRuntime.configStatus.unknown': 'Unknown',
    'monitor.moduleRuntime.dependencyStatus.satisfied': 'Satisfied',
    'monitor.moduleRuntime.dependencyStatus.missing': 'Missing',
    'monitor.moduleRuntime.dependencyStatus.disabled': 'Disabled',
    'monitor.moduleRuntime.enablementSource.all': 'All modules',
    'monitor.moduleRuntime.enablementSource.allowlist': 'Allowlist',
  }),
);

vi.mock('../../api/module-runtime', () => ({
  getModuleRuntimeDetail: moduleRuntimeApiMocks.getModuleRuntimeDetail,
  getModuleRuntimeSnapshot: moduleRuntimeApiMocks.getModuleRuntimeSnapshot,
}));

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n');
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => {
        const template = translations[key] ?? key;
        return Object.entries(params ?? {}).reduce(
          (message, [name, value]) => message.replace(`{${name}}`, String(value)),
          template,
        );
      },
    }),
  };
});

vi.mock('@/utils/logger', () => ({
  createLogger: () => loggerMocks,
}));

vi.mock('vue-router', () => ({
  useRoute: () => routeMocks.route,
}));

vi.mock('@/store', () => ({
  useTabsRouterStore: () => tabsRouterStoreMock,
}));

const shellStub = defineComponent({
  name: 'ServerStatusPageShellStub',
  props: {
    eyebrow: {
      type: String,
      default: '',
    },
    title: {
      type: String,
      default: '',
    },
    description: {
      type: String,
      default: '',
    },
    titleKey: {
      type: String,
      default: '',
    },
    descriptionKey: {
      type: String,
      default: '',
    },
  },
  setup(props, { slots }) {
    const resolveText = (key: string, fallback: string) => {
      const translated = translations[key] ?? key;
      return translated && translated !== key ? translated : fallback;
    };

    return () =>
      h('section', { 'data-page-type': 'overview-dashboard' }, [
        h('header', [
          props.eyebrow,
          resolveText(props.titleKey, props.title),
          resolveText(props.descriptionKey, props.description),
          slots.toolbar?.(),
          slots.summary?.(),
        ]),
        slots.feedback?.(),
        slots.default?.(),
      ]);
  },
});

const passthroughStub = defineComponent({
  name: 'PassthroughStub',
  props: {
    title: {
      type: String,
      default: '',
    },
    message: {
      type: String,
      default: '',
    },
    description: {
      type: String,
      default: '',
    },
    label: {
      type: String,
      default: '',
    },
    value: {
      type: [Number, String],
      default: '',
    },
  },
  setup(props, { slots }) {
    return () =>
      h('div', [props.title, props.message, props.description, props.label, String(props.value), slots.default?.()]);
  },
});

const buttonStub = defineComponent({
  name: 'TButtonStub',
  emits: ['click'],
  setup(_props, { attrs, emit, slots }) {
    return () => h('button', { ...attrs, onClick: (event: MouseEvent) => emit('click', event) }, slots.default?.());
  },
});

const selectStub = defineComponent({
  name: 'TSelectStub',
  props: {
    modelValue: {
      type: [Number, String],
      default: '',
    },
    options: {
      type: Array,
      default: () => [],
    },
  },
  emits: ['update:modelValue'],
  setup(props, { attrs, emit }) {
    return () =>
      h(
        'select',
        {
          ...attrs,
          value: String(props.modelValue),
          onChange: (event: Event) => emit('update:modelValue', (event.target as HTMLSelectElement).value),
        },
        (props.options as Array<{ label: string; value: number | string }>).map((option) =>
          h('option', { value: String(option.value) }, option.label),
        ),
      );
  },
});

const tableStub = defineComponent({
  name: 'TTableStub',
  props: {
    data: {
      type: Array,
      default: () => [],
    },
    columns: {
      type: Array,
      default: () => [],
    },
    empty: {
      type: String,
      default: '',
    },
    size: {
      type: String,
      default: 'medium',
    },
    tableContentWidth: {
      type: String,
      default: '',
    },
  },
  setup(props, { slots }) {
    return () =>
      h(
        'div',
        {
          'data-table-columns': JSON.stringify(props.columns),
          'data-table-content-width': props.tableContentWidth,
          'data-table-size': props.size,
        },
        [
          (props.data as Array<Record<string, unknown>>).map((row) =>
            h(
              'div',
              { class: 'table-row' },
              (props.columns as Array<{ colKey: string }>).map((column) =>
                h(
                  'div',
                  { class: `table-cell-${column.colKey}` },
                  slots[column.colKey]?.({ row }) ?? String(row[column.colKey] ?? ''),
                ),
              ),
            ),
          ),
          !(props.data as unknown[]).length ? props.empty : '',
        ],
      );
  },
});

const drawerStub = defineComponent({
  name: 'TDrawerStub',
  props: {
    visible: {
      type: Boolean,
      default: false,
    },
    header: {
      type: String,
      default: '',
    },
  },
  setup(props, { slots }) {
    return () => (props.visible ? h('aside', { 'data-drawer': 'true' }, [props.header, slots.default?.()]) : null);
  },
});

const columnDrawerStub = defineComponent({
  name: 'AdvancedQueryColumnDrawerStub',
  props: {
    columns: {
      type: Array,
      default: () => [],
    },
    defaultSelectedKeys: {
      type: Array,
      default: () => [],
    },
    disabledKeys: {
      type: Array,
      default: () => [],
    },
    visible: {
      type: Boolean,
      default: false,
    },
    title: {
      type: String,
      default: '',
    },
  },
  setup(props) {
    return () =>
      props.visible
        ? h(
            'aside',
            {
              'data-column-drawer': 'true',
              'data-column-options': JSON.stringify(props.columns),
              'data-default-selected-keys': JSON.stringify(props.defaultSelectedKeys),
              'data-disabled-keys': JSON.stringify(props.disabledKeys),
            },
            props.title,
          )
        : null;
  },
});

const popupStub = defineComponent({
  name: 'TPopupStub',
  setup(_props, { slots }) {
    return () => h('span', [slots.default?.(), slots.content?.()]);
  },
});

const tooltipStub = defineComponent({
  name: 'TTooltipStub',
  setup(_props, { slots }) {
    return () => h('span', slots.default?.());
  },
});

const collapseStub = defineComponent({
  name: 'TCollapseStub',
  setup(_props, { slots }) {
    return () => h('div', slots.default?.());
  },
});

const collapsePanelStub = defineComponent({
  name: 'TCollapsePanelStub',
  props: {
    header: {
      type: String,
      default: '',
    },
  },
  setup(props, { slots }) {
    return () => h('section', { 'data-collapse-panel': 'true' }, [props.header, slots.default?.()]);
  },
});

function mountModulesPage() {
  return mount(ModulesPage, {
    global: {
      stubs: {
        ServerStatusPageShell: shellStub,
        AdvancedQueryColumnDrawer: columnDrawerStub,
        SectionCard: passthroughStub,
        't-alert': passthroughStub,
        't-button': buttonStub,
        't-collapse': collapseStub,
        't-collapse-panel': collapsePanelStub,
        't-descriptions': passthroughStub,
        't-descriptions-item': passthroughStub,
        't-drawer': drawerStub,
        't-empty': passthroughStub,
        't-popup': popupStub,
        't-select': selectStub,
        't-statistic': passthroughStub,
        't-table': tableStub,
        't-tag': passthroughStub,
        't-tooltip': tooltipStub,
      },
    },
  });
}

afterEach(() => {
  moduleRuntimeApiMocks.getModuleRuntimeSnapshot.mockReset();
  moduleRuntimeApiMocks.getModuleRuntimeDetail.mockReset();
  loggerMocks.error.mockReset();
});

function createSnapshot() {
  return {
    summary: {
      total_modules: 3,
      enabled_modules: 2,
      registered_modules: 3,
      healthy_modules: 1,
      degraded_modules: 1,
      unknown_modules: 1,
    },
    items: [
      {
        module_key: 'audit',
        registered: true,
        enabled: true,
        enablement_source: 'all',
        runtime_status: 'registered',
        health: 'healthy',
        dependencies: [{ module_key: 'user', required: true, present: true, enabled: true, status: 'satisfied' }],
        migration_status: { declared_dirs: ['server/modules/audit/migrations'], status: 'declared' },
        schema_status: { status: 'declared' },
        config_status: { status: 'not_required' },
        diagnostics: { boot: 'ok' },
      },
      {
        module_key: 'scheduler',
        registered: true,
        enabled: false,
        enablement_source: 'allowlist',
        runtime_status: 'disabled',
        health: 'disabled',
        dependencies: [],
        migration_status: { declared_dirs: [], status: 'not_declared' },
        schema_status: { status: 'unknown' },
        config_status: { status: 'unknown' },
        diagnostics: {},
      },
    ],
  };
}

describe('monitor module runtime page', () => {
  it('loads module runtime snapshot and renders summary, table state, and detail drawer', async () => {
    const snapshot = createSnapshot();
    moduleRuntimeApiMocks.getModuleRuntimeSnapshot.mockResolvedValue(snapshot);
    moduleRuntimeApiMocks.getModuleRuntimeDetail.mockResolvedValue(snapshot.items[0]);

    const wrapper = mountModulesPage();
    await flushPromises();

    expect(wrapper.attributes('data-page-type')).toBe('overview-dashboard');
    expect(wrapper.text()).toContain('Modules');
    expect(wrapper.text()).toContain('Needs attention');
    expect(wrapper.text()).toContain('Every 5 sec');
    expect(wrapper.text()).toContain('5s 后刷新');
    expect(wrapper.text()).toContain('Pause auto refresh');
    expect(wrapper.text()).toContain('Runtime Module List');
    expect(wrapper.text()).toContain('Column settings');
    expect(wrapper.find('button[aria-label="Compact density"]').exists()).toBe(true);
    expect(wrapper.text()).toContain('audit');
    expect(wrapper.text()).toContain('scheduler');
    expect(wrapper.text()).toContain('1 / 1 satisfied');
    expect(wrapper.find('.table-cell-resource_status').text()).toContain('Migrations');
    expect(wrapper.find('.table-cell-resource_status').text()).toContain('Schema');
    expect(wrapper.find('.table-cell-resource_status').text()).toContain('Config');
    expect(wrapper.text()).toContain('Not required');

    const table = wrapper.find('[data-table-columns]');
    const columns = JSON.parse(table.attributes('data-table-columns') ?? '[]');
    expect(columns.map((column: { colKey: string }) => column.colKey)).toEqual([
      'module_key',
      'enabled',
      'registered',
      'health',
      'dependencies',
      'resource_status',
      'operation',
    ]);
    expect(table.attributes('data-table-size')).toBe('medium');
    expect(table.attributes('data-table-content-width')).toBe('');
    expect(columns.find((column: { colKey: string }) => column.colKey === 'module_key').minWidth).toBe(180);
    expect(columns.find((column: { colKey: string }) => column.colKey === 'resource_status').minWidth).toBe(260);
    expect(columns.find((column: { colKey: string }) => column.colKey === 'operation').fixed).toBe('right');

    await wrapper
      .findAll('button')
      .find((button) => button.text().includes('Detail'))
      ?.trigger('click');

    expect(wrapper.find('[data-drawer="true"]').text()).toContain('audit runtime detail');
    expect(wrapper.find('[data-drawer="true"]').text()).toContain('Basic information');
    expect(wrapper.find('[data-drawer="true"]').text()).toContain('Dependencies');
    expect(wrapper.find('[data-drawer="true"]').text()).toContain('Migration');
    expect(wrapper.find('[data-drawer="true"]').text()).toContain('Schema');
    expect(wrapper.find('[data-drawer="true"]').text()).toContain('Config');
    expect(wrapper.find('[data-drawer="true"]').text()).toContain('All modules');
    expect(wrapper.find('[data-drawer="true"]').text()).toContain('server/modules/audit/migrations');
    expect(wrapper.find('[data-drawer="true"]').text()).toContain('Module-owned');
    expect(wrapper.find('[data-drawer="true"]').text()).toContain('boot');
    expect(wrapper.find('[data-drawer="true"]').text()).toContain('ok');
    expect(wrapper.find('[data-drawer="true"]').text()).toContain('Raw runtime JSON');
    expect(wrapper.find('[data-drawer="true"]').text()).toContain('"module_key": "audit"');
    expect(moduleRuntimeApiMocks.getModuleRuntimeDetail).toHaveBeenCalledWith('audit');
  });

  it('opens column settings with locked critical columns and toggles table density', async () => {
    moduleRuntimeApiMocks.getModuleRuntimeSnapshot.mockResolvedValue(createSnapshot());

    const wrapper = mountModulesPage();
    await flushPromises();

    await wrapper
      .findAll('button')
      .find((button) => button.text().includes('Column settings'))
      ?.trigger('click');

    const drawer = wrapper.find('[data-column-drawer="true"]');
    expect(drawer.exists()).toBe(true);
    expect(JSON.parse(drawer.attributes('data-disabled-keys') ?? '[]')).toEqual(['module_key', 'health', 'operation']);
    expect(JSON.parse(drawer.attributes('data-default-selected-keys') ?? '[]')).toEqual([
      'module_key',
      'enabled',
      'registered',
      'health',
      'dependencies',
      'resource_status',
      'operation',
    ]);
    expect(JSON.parse(drawer.attributes('data-column-options') ?? '[]')).toEqual(
      expect.arrayContaining([{ label: 'Resource status', value: 'resource_status' }]),
    );

    await wrapper.find('button[aria-label="Compact density"]').trigger('click');

    expect(wrapper.find('[data-table-columns]').attributes('data-table-size')).toBe('small');
    expect(wrapper.find('button[aria-label="Default density"]').exists()).toBe(true);
  });

  it('refreshes the snapshot on demand', async () => {
    moduleRuntimeApiMocks.getModuleRuntimeSnapshot.mockResolvedValue(createSnapshot());

    const wrapper = mountModulesPage();
    await flushPromises();
    await wrapper
      .findAll('button')
      .find((button) => button.text().includes('Refresh'))
      ?.trigger('click');
    await flushPromises();

    expect(moduleRuntimeApiMocks.getModuleRuntimeSnapshot).toHaveBeenCalledTimes(2);
  });

  it('renders empty and error states', async () => {
    moduleRuntimeApiMocks.getModuleRuntimeSnapshot.mockResolvedValueOnce({
      summary: {
        total_modules: 0,
        enabled_modules: 0,
        registered_modules: 0,
        healthy_modules: 0,
        degraded_modules: 0,
        unknown_modules: 0,
      },
      items: [],
    });

    const emptyWrapper = mountModulesPage();
    await flushPromises();

    expect(emptyWrapper.text()).toContain('No module runtime data');

    moduleRuntimeApiMocks.getModuleRuntimeSnapshot.mockRejectedValueOnce(new Error('network down'));

    const errorWrapper = mountModulesPage();
    await flushPromises();

    expect(errorWrapper.text()).toContain('Module snapshot request failed');
    expect(errorWrapper.text()).toContain('Failed to load module runtime snapshot');
    expect(errorWrapper.text()).not.toContain('network down');
    expect(loggerMocks.error).toHaveBeenCalledWith(expect.any(Error), {
      operation: 'module_runtime_snapshot',
    });
  });

  it('shows the localized load failure message when detail loading fails', async () => {
    moduleRuntimeApiMocks.getModuleRuntimeSnapshot.mockResolvedValue(createSnapshot());
    moduleRuntimeApiMocks.getModuleRuntimeDetail.mockRejectedValue(new Error('detail unavailable'));

    const wrapper = mountModulesPage();
    await flushPromises();
    await wrapper
      .findAll('button')
      .find((button) => button.text().includes('Detail'))
      ?.trigger('click');
    await flushPromises();

    expect(wrapper.text()).toContain('Failed to load module runtime snapshot');
    expect(wrapper.text()).not.toContain('detail unavailable');
    expect(wrapper.find('[data-drawer="true"]').exists()).toBe(false);
    expect(loggerMocks.error).toHaveBeenCalledWith(expect.any(Error), {
      moduleKey: 'audit',
      operation: 'module_runtime_detail',
    });
  });
});
