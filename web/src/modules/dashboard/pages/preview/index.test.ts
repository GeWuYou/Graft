import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h, ref } from 'vue';

import { DASHBOARD_PREVIEW_PRESENTATION } from '../../presentation/preview-workbench';
import DashboardWorkbenchPreviewPage from './index.vue';

const routerMocks = vi.hoisted(() => ({ push: vi.fn() }));
const navigationMocks = vi.hoisted(() => ({
  links: [
    {
      id: 'build-jobs',
      module_key: 'build',
      title: 'Build Jobs',
      group: 'Build',
      icon: 'build',
      route_location: '/build/jobs',
      order: 1,
    },
    {
      id: 'access-logs',
      module_key: 'access-log',
      title: 'Access Logs',
      group: 'Observability',
      icon: 'access-log',
      route_location: '/logs/access',
      order: 2,
    },
    {
      id: 'application-logs',
      module_key: 'application-log',
      title: 'Application Logs',
      group: 'Observability',
      icon: 'application-log',
      route_location: '/logs/application',
      order: 3,
    },
    {
      id: 'audit-logs',
      module_key: 'audit',
      title: 'Audit Logs',
      group: 'Security',
      icon: 'audit-trail',
      route_location: '/audit/logs',
      order: 4,
    },
  ],
}));

vi.mock('vue-router', () => ({ useRouter: () => routerMocks }));
vi.mock('@/store/modules/permission', () => ({ usePermissionStore: () => ({ routers: [] }) }));
vi.mock('../../contract/sidebar-quick-actions', () => ({
  buildDashboardQuickActionLinks: () => navigationMocks.links,
}));

vi.mock('@/locales', () => ({
  currentLocale: ref('en-US'),
  t: (key: string, params?: Record<string, unknown>) => {
    if (key === 'dashboard.workbench.operational.needsAttention') {
      return 'Needs Attention';
    }
    if (key === 'dashboard.workbench.operational.itemCount') {
      return `${params?.count ?? 0} Items`;
    }
    if (key === 'dashboard.workbench.operational.distribution') {
      return `${params?.warning ?? 0} warning, ${params?.unknown ?? 0} unknown`;
    }
    if (key === 'dashboard.workbench.attention.title') {
      return 'Attention';
    }
    return key;
  },
}));

vi.mock('@/shared/observability', () => ({
  MEDIUM_DATE_TIME_FORMAT_OPTIONS: {},
  formatBytes: (value: number | null) => (value === null ? '-' : `${value} B`),
  formatLocaleDateTime: (value: string) => `formatted:${value}`,
  formatPercent: (value: number | null) => (value === null ? '-' : `${value.toFixed(1)}%`),
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
    setup(_props, { slots }) {
      return () => h('div', { class: 'responsive-content-stub' }, slots.default?.());
    },
  }),
}));

vi.mock('@/shared/components/responsive/ResponsiveDialog.vue', () => ({
  default: defineComponent({
    name: 'ResponsiveDialog',
    props: {
      closeLabel: { type: String, default: '' },
      purpose: { type: String, default: '' },
      size: { type: String, default: '' },
      title: { type: String, default: '' },
      visible: Boolean,
    },
    setup(_props, { slots }) {
      return () => h('aside', slots.default?.());
    },
  }),
}));

vi.mock('@/shared/icons/MenuIcon.vue', () => ({
  default: defineComponent({
    name: 'GraftMenuIconStub',
    props: { iconKey: { type: String, default: '' } },
    setup(props) {
      return () => h('i', { 'data-icon-key': props.iconKey });
    },
  }),
}));

const surfaceStub = defineComponent({
  name: 'SurfaceStub',
  setup(_props, { slots }) {
    return () => h('section', [slots.header?.(), slots.default?.(), slots.action?.(), slots.footer?.()]);
  },
});

const buttonStub = defineComponent({
  name: 'ButtonStub',
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

const inputStub = defineComponent({
  name: 'InputStub',
  props: { modelValue: { type: String, default: '' }, placeholder: { type: String, default: '' } },
  emits: ['update:modelValue'],
  setup(props, { emit, slots }) {
    return () =>
      h('label', [
        slots.prefixIcon?.(),
        h('input', {
          value: props.modelValue,
          placeholder: props.placeholder,
          onInput: (event: Event) => emit('update:modelValue', (event.target as HTMLInputElement).value),
        }),
      ]);
  },
});

function mountPreview() {
  return mount(DashboardWorkbenchPreviewPage, {
    global: {
      stubs: {
        TButton: buttonStub,
        TCard: surfaceStub,
        TCollapse: surfaceStub,
        TCollapsePanel: surfaceStub,
        TDrawer: surfaceStub,
        TInput: inputStub,
        TList: surfaceStub,
        TListItem: surfaceStub,
        TTag: surfaceStub,
      },
    },
  });
}

describe('DashboardWorkbenchPreviewPage', () => {
  it('renders warning and unknown attention states without promoting either to an error', () => {
    const wrapper = mountPreview();
    const accessLog = wrapper.get('[data-attention-id="access-log-source"]');
    const outbound = wrapper.get('[data-attention-id="outbound-network"]');

    expect(accessLog.attributes('data-status')).toBe('warning');
    expect(accessLog.attributes('data-evidence')).toBe('source-failed');
    expect(accessLog.classes()).toContain('attention-row--warning');
    expect(accessLog.classes()).not.toContain('attention-row--error');

    expect(outbound.attributes('data-status')).toBe('unknown');
    expect(outbound.attributes('data-evidence')).toBe('missing');
    expect(outbound.classes()).toContain('attention-row--unknown');
    expect(outbound.classes()).not.toContain('attention-row--warning');
    expect(outbound.classes()).not.toContain('attention-row--error');
    expect(wrapper.text()).toContain('Needs Attention');
    expect(wrapper.text()).toContain('6 Items');
    expect(wrapper.text()).toContain('3 warning, 3 unknown');
    expect(wrapper.text()).toContain('Attention');
    expect(wrapper.text()).not.toContain('items need confirmation');
    expect(accessLog.findAll('.workbench-status')).toHaveLength(1);
    expect(outbound.findAll('.workbench-status')).toHaveLength(1);
  });

  it('keeps healthy dependencies quiet and renders the loaded resource detail scenario', () => {
    const wrapper = mountPreview();

    expect(wrapper.get('[data-health-id="health-postgresql"]').attributes('data-status')).toBe('healthy');
    expect(wrapper.get('[data-health-id="health-redis"]').attributes('data-status')).toBe('healthy');
    expect(wrapper.get('[data-resource-state="loaded"]').attributes('data-resource-state')).toBe('loaded');
    expect(wrapper.findAll('[data-resource-group]')).toHaveLength(3);
    expect(wrapper.find('.health-row--emphasized').exists()).toBe(false);
    expect(wrapper.find('[data-secondary-region="activity"]').exists()).toBe(false);
  });

  it('uses configurable home actions and the shared responsive overlay for all entries', () => {
    const wrapper = mountPreview();
    const iconKeys = wrapper.findAll('[data-icon-key]').map((icon) => icon.attributes('data-icon-key'));
    const homeIconKeys = DASHBOARD_PREVIEW_PRESENTATION.homeQuickActions.map((action) => action.iconKey);

    expect(iconKeys.slice(0, homeIconKeys.length)).toEqual(homeIconKeys);
    expect(iconKeys).toContain('access-log');
    expect(wrapper.findAll('.workbench-quick-action-item--default')).toHaveLength(
      DASHBOARD_PREVIEW_PRESENTATION.homeQuickActions.length,
    );
    expect(wrapper.findAll('.workbench-quick-action-item--drawer-featured')).toHaveLength(
      DASHBOARD_PREVIEW_PRESENTATION.homeQuickActions.length,
    );
    expect(wrapper.findAll('.quick-entry-drawer__navigation-item')).toHaveLength(navigationMocks.links.length);
    expect(wrapper.getComponent({ name: 'ResponsiveDialog' }).props()).toMatchObject({
      purpose: 'workspace',
      size: 'compact',
    });
    expect(wrapper.get('[role="group"]').attributes('aria-label')).toBe('dashboard.workbench.quickActions.title');
  });

  it('filters the complete navigation without treating the home item count as a drawer limit', async () => {
    const wrapper = mountPreview();

    await wrapper.get('input').setValue('logs');

    expect(wrapper.find('.quick-entry-drawer__frequent').exists()).toBe(false);
    expect(wrapper.findAll('.quick-entry-drawer__navigation-item')).toHaveLength(3);
    expect(wrapper.text()).toContain('Access Logs');
    expect(wrapper.text()).toContain('Application Logs');
    expect(wrapper.text()).toContain('Audit Logs');
    expect(wrapper.text()).not.toContain('Build Jobs');
  });
});
