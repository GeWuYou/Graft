import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h, inject, type InjectionKey, nextTick, type PropType, provide, ref } from 'vue';

import enUS from '../../locales/en-US.json';
import zhCN from '../../locales/zh-CN.json';
import { DASHBOARD_PREVIEW_PRESENTATION } from '../../presentation/preview-workbench';
import { projectWorkbenchScenario, type WorkbenchPresentation } from '../../presentation/workbench';
import DashboardWorkbench from './DashboardWorkbench.vue';
import componentSource from './DashboardWorkbench.vue?raw';
import contextLinkSource from './WorkbenchContextLinkList.vue?raw';
import presentationListSource from './WorkbenchPresentationList.vue?raw';

vi.mock('@/locales', () => ({
  currentLocale: ref('en-US'),
  t: (key: string, params?: Record<string, unknown>) => {
    const labels: Record<string, string> = {
      'dashboard.workbench.expand.attention': `More attention (${params?.count ?? 0})`,
      'dashboard.workbench.expand.health': `More health (${params?.count ?? 0})`,
      'dashboard.workbench.expand.contextLinks': `More actions (${params?.count ?? 0})`,
    };
    return labels[key] ?? `translated:${key}`;
  },
}));

vi.mock('@/shared/components/page', () => ({
  PageHeader: defineComponent({
    name: 'PageHeaderStub',
    setup(_props, { slots }) {
      return () => h('header', [slots.default?.(), slots.actions?.()]);
    },
  }),
}));

vi.mock('@/shared/components/responsive/ResponsiveContent.vue', () => ({
  default: defineComponent({
    name: 'ResponsiveContentStub',
    setup(_props, { attrs, slots }) {
      return () => h('div', attrs, slots.default?.());
    },
  }),
}));

vi.mock('@/shared/components/responsive/ResponsiveDialog.vue', () => ({
  default: defineComponent({
    name: 'ResponsiveDialogStub',
    setup(_props, { slots }) {
      return () => h('aside', slots.default?.());
    },
  }),
}));

vi.mock('@/shared/icons/MenuIcon.vue', () => ({
  default: defineComponent({
    name: 'GraftMenuIconStub',
    setup() {
      return () => h('i');
    },
  }),
}));

type CollapseContext = {
  isOpen: (value: string) => boolean;
  toggle: (value: string) => void;
};

const COLLAPSE_CONTEXT: InjectionKey<CollapseContext> = Symbol('collapse-context');

const surfaceStub = defineComponent({
  name: 'SurfaceStub',
  inheritAttrs: false,
  setup(_props, { attrs, slots }) {
    return () => h('section', attrs, [slots.header?.(), slots.default?.(), slots.action?.(), slots.footer?.()]);
  },
});

const buttonStub = defineComponent({
  name: 'ButtonStub',
  inheritAttrs: false,
  emits: ['click'],
  setup(_props, { attrs, emit, slots }) {
    return () =>
      h('button', { ...attrs, onClick: (event: MouseEvent) => emit('click', event) }, [
        slots.icon?.(),
        slots.default?.(),
        slots.suffix?.(),
      ]);
  },
});

const alertStub = defineComponent({
  name: 'AlertStub',
  props: { message: { type: String, default: '' } },
  setup(props, { slots }) {
    return () => h('section', { class: 'alert-stub' }, [props.message, slots.operation?.()]);
  },
});

const collapseStub = defineComponent({
  name: 'CollapseStub',
  props: { modelValue: { type: Array as PropType<string[]>, default: () => [] } },
  emits: ['update:modelValue'],
  setup(props, { emit, slots }) {
    provide(COLLAPSE_CONTEXT, {
      isOpen: (value) => props.modelValue.includes(value),
      toggle: (value) => {
        const next = props.modelValue.includes(value)
          ? props.modelValue.filter((item) => item !== value)
          : [...props.modelValue, value];
        emit('update:modelValue', next);
      },
    });
    return () => h('div', { class: 'collapse-stub' }, slots.default?.());
  },
});

const collapsePanelStub = defineComponent({
  name: 'CollapsePanelStub',
  props: {
    header: { type: String, default: '' },
    value: { type: String, required: true },
  },
  setup(props, { slots }) {
    const collapse = inject(COLLAPSE_CONTEXT);
    const toggle = () => collapse?.toggle(props.value);
    return () =>
      h('section', [
        h(
          'button',
          {
            class: 'collapse-trigger',
            'data-collapse-value': props.value,
            onClick: toggle,
            onKeydown: (event: KeyboardEvent) => {
              if (event.key === 'Enter' || event.key === ' ') {
                event.preventDefault();
                toggle();
              }
            },
          },
          props.header,
        ),
        collapse?.isOpen(props.value) ? h('div', { class: 'collapse-content' }, slots.default?.()) : null,
      ]);
  },
});

const quickActionStub = defineComponent({
  name: 'WorkbenchQuickActionItem',
  props: { action: { type: Object, required: true } },
  emits: ['activate'],
  setup(props, { emit }) {
    return () =>
      h(
        'button',
        { class: 'quick-action-stub', onClick: () => emit('activate', props.action) },
        String((props.action as { id?: string }).id ?? ''),
      );
  },
});

const statusStub = defineComponent({
  name: 'WorkbenchStatusIndicator',
  setup(_props, { attrs }) {
    return () => h('span', { ...attrs, class: 'workbench-status-stub' });
  },
});

const emptyPresentation = projectWorkbenchScenario({
  generatedAt: '2026-08-17T03:20:00.000Z',
  operational: { enabledModules: 0, failedTasks: 0, highRiskEvents: 0 },
  items: [],
  quickActions: [],
});

function mountWorkbench(overrides: Partial<InstanceType<typeof DashboardWorkbench>['$props']> = {}) {
  return mount(DashboardWorkbench, {
    props: {
      generatedAt: DASHBOARD_PREVIEW_PRESENTATION.generatedAt,
      navigationLinks: [],
      presentation: DASHBOARD_PREVIEW_PRESENTATION as WorkbenchPresentation,
      preview: true,
      ...overrides,
    },
    global: {
      stubs: {
        TAlert: alertStub,
        TButton: buttonStub,
        TCard: surfaceStub,
        TCollapse: collapseStub,
        TCollapsePanel: collapsePanelStub,
        TInput: surfaceStub,
        TList: surfaceStub,
        TListItem: surfaceStub,
        TSkeleton: surfaceStub,
        TTag: surfaceStub,
        WorkbenchQuickActionItem: quickActionStub,
        WorkbenchStatusIndicator: statusStub,
      },
    },
  });
}

function flattenKeys(value: unknown, prefix = ''): string[] {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return [prefix];
  }
  return Object.entries(value).flatMap(([key, child]) => flattenKeys(child, prefix ? `${prefix}.${key}` : key));
}

describe('DashboardWorkbench', () => {
  it('keeps the first screen in operational priority order and applies the summary budgets', () => {
    const wrapper = mountWorkbench();

    expect(
      wrapper.findAll('[data-first-screen-region]').map((node) => node.attributes('data-first-screen-region')),
    ).toEqual(['operational-status', 'attention', 'health', 'module-coverage']);
    expect(wrapper.findAll('[data-attention-id]')).toHaveLength(5);
    expect(wrapper.findAll('[data-health-id]')).toHaveLength(3);
    expect(wrapper.findAll('[data-context-link-key]')).toHaveLength(6);
    expect(wrapper.find('[data-secondary-region="activity"]').exists()).toBe(false);
  });

  it('reveals attention, health, and contextual overflow through keyboard-accessible collapse triggers', async () => {
    const wrapper = mountWorkbench();

    await wrapper.get('[data-collapse-value="attention-more"]').trigger('keydown', { key: 'Enter' });
    await nextTick();
    expect(wrapper.findAll('[data-attention-id]')).toHaveLength(6);

    await wrapper.get('[data-collapse-value="health-more"]').trigger('keydown', { key: ' ' });
    await nextTick();
    expect(wrapper.findAll('[data-health-id]')).toHaveLength(5);

    const contextTrigger = wrapper
      .findAll('.collapse-trigger')
      .find((node) => node.attributes('data-collapse-value')?.startsWith('context-more:'));
    expect(contextTrigger).toBeDefined();
    await contextTrigger!.trigger('keydown', { key: 'Enter' });
    await nextTick();
    expect(wrapper.findAll('[data-context-link-key]')).toHaveLength(8);
  });

  it('keeps contextual routes separate from quick-entry ranking events', async () => {
    const wrapper = mountWorkbench();

    await wrapper.get('[data-context-link-key="build-jobs"]').trigger('click');
    await wrapper.get('[data-metric-key="running"]').trigger('click');
    await wrapper.get('.quick-action-stub').trigger('click');
    await wrapper.get('.workbench-surface--resources .workbench-surface__heading button').trigger('click');

    const navigations = wrapper.emitted('navigate') ?? [];
    expect(navigations).toContainEqual(['/build/jobs', 'contextual-action']);
    expect(navigations).toContainEqual(['/platform/scheduled-tasks', 'contextual-action']);
    expect(navigations).toContainEqual(['/build/jobs/create', 'quick-entry']);
    expect(navigations.at(-1)?.[1]).toBe('contextual-action');
    expect(navigations.at(-1)?.[0]).not.toBe('/infrastructure/docker/containers/resources');
  });

  it('renders detailed loaded resources without inventing timeline activity', () => {
    const wrapper = mountWorkbench();

    expect(wrapper.get('[data-resource-state="loaded"]').attributes('data-resource-state')).toBe('loaded');
    expect(wrapper.findAll('[data-resource-group]')).toHaveLength(3);
    expect(wrapper.find('[data-secondary-region="activity"]').exists()).toBe(false);
    expect(wrapper.text()).toContain('graft-api');
    expect(wrapper.text()).toContain('graft-postgres');
    expect(wrapper.text()).toContain('graft-worker');
  });

  it('preserves loading, error, and empty feedback boundaries', () => {
    const loading = mountWorkbench({ loading: true, ready: false, presentation: emptyPresentation });
    expect(loading.find('.workbench-preview__loading').exists()).toBe(true);
    expect(loading.find('[data-first-screen-region]').exists()).toBe(false);

    const failed = mountWorkbench({
      errorMessage: 'summary unavailable',
      ready: false,
      presentation: emptyPresentation,
    });
    expect(failed.text()).toContain('summary unavailable');

    const empty = mountWorkbench({ presentation: emptyPresentation });
    expect(empty.find('[data-first-screen-region="attention"]').text()).toContain(
      'translated:dashboard.workbench.attention.empty',
    );
    expect(empty.find('.workbench-surface--resources').exists()).toBe(false);
  });

  it('keeps new locale keys in parity and uses only token-driven component styling', () => {
    expect(flattenKeys(enUS.dashboard.workbench).sort()).toEqual(flattenKeys(zhCN.dashboard.workbench).sort());
    expect(flattenKeys(enUS.dashboard.widget).sort()).toEqual(flattenKeys(zhCN.dashboard.widget).sort());
    expect(enUS.dashboard.widget.monitorSystemHealth.hostResources).toBe('Host Resources');
    expect(zhCN.dashboard.widget.auditRiskEvents.sensitiveOperations.title).toBe('敏感操作');
    expect(enUS.dashboard.widget.announcementTimeline.title).toBe('Latest Announcements');
    expect(zhCN.dashboard.widget.backupHealth.available.label).toBe('备份可用');
    expect(enUS.dashboard.widget.runtimeTargetSummary.unavailable).toBe('Unavailable Targets');
    const workbenchSources = [componentSource, contextLinkSource, presentationListSource].join('\n');
    expect(workbenchSources).not.toMatch(/#[\dA-Fa-f]{3,8}\b/);
    expect(componentSource).not.toContain('/infrastructure/docker/containers/resources');
    expect(workbenchSources).toContain('var(--td-');
  });
});
