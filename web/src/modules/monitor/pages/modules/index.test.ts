import { flushPromises, mount } from '@vue/test-utils';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import ModulesPage from './index.vue';

const moduleRuntimeApiMocks = vi.hoisted(() => ({
  getModuleRuntimeSnapshot: vi.fn(),
}));

const translations = vi.hoisted(
  (): Record<string, string> => ({
    'monitor.sectionTitle': 'Service Management',
    'monitor.moduleRuntime.title': 'Module Runtime',
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
    'monitor.moduleRuntime.columns.moduleKey': 'Module key',
    'monitor.moduleRuntime.columns.enabled': 'Enabled',
    'monitor.moduleRuntime.columns.registered': 'Registered',
    'monitor.moduleRuntime.columns.health': 'Health',
    'monitor.moduleRuntime.columns.dependencies': 'Dependencies',
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
  getModuleRuntimeSnapshot: moduleRuntimeApiMocks.getModuleRuntimeSnapshot,
}));

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) => {
      const template = translations[key] ?? key;
      return Object.entries(params ?? {}).reduce(
        (message, [name, value]) => message.replace(`{${name}}`, String(value)),
        template,
      );
    },
  }),
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
  },
  setup(props, { slots }) {
    return () =>
      h('section', { 'data-page-type': 'overview-dashboard' }, [
        h('header', [props.eyebrow, props.title, props.description, slots.toolbar?.(), slots.summary?.()]),
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
  },
  setup(props, { slots }) {
    return () =>
      h('div', { 'data-table-columns': JSON.stringify(props.columns) }, [
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
      ]);
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

function mountModulesPage() {
  return mount(ModulesPage, {
    global: {
      stubs: {
        ServerStatusPageShell: shellStub,
        SectionCard: passthroughStub,
        't-alert': passthroughStub,
        't-button': buttonStub,
        't-descriptions': passthroughStub,
        't-descriptions-item': passthroughStub,
        't-drawer': drawerStub,
        't-empty': passthroughStub,
        't-statistic': passthroughStub,
        't-table': tableStub,
        't-tag': passthroughStub,
        RefreshIcon: true,
      },
    },
  });
}

afterEach(() => {
  vi.clearAllMocks();
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
    moduleRuntimeApiMocks.getModuleRuntimeSnapshot.mockResolvedValue(createSnapshot());

    const wrapper = mountModulesPage();
    await flushPromises();

    expect(wrapper.attributes('data-page-type')).toBe('overview-dashboard');
    expect(wrapper.text()).toContain('Module Runtime');
    expect(wrapper.text()).toContain('Needs attention');
    expect(wrapper.text()).toContain('Runtime Module List');
    expect(wrapper.text()).toContain('audit');
    expect(wrapper.text()).toContain('scheduler');
    expect(wrapper.text()).toContain('1 / 1 satisfied');
    expect(wrapper.text()).toContain('1 dirs');
    expect(wrapper.text()).toContain('Not required');

    const columns = JSON.parse(wrapper.find('[data-table-columns]').attributes('data-table-columns') ?? '[]');
    expect(columns.map((column: { colKey: string }) => column.colKey)).toEqual([
      'module_key',
      'enabled',
      'registered',
      'health',
      'dependencies',
      'migration',
      'schema',
      'config',
      'operation',
    ]);

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
    expect(errorWrapper.text()).toContain('network down');
  });
});
