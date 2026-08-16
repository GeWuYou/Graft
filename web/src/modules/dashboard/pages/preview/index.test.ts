import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h, ref } from 'vue';

import DashboardWorkbenchPreviewPage from './index.vue';

const routerMocks = vi.hoisted(() => ({ push: vi.fn() }));

vi.mock('vue-router', () => ({ useRouter: () => routerMocks }));

vi.mock('@/locales', () => ({
  currentLocale: ref('en-US'),
  t: (key: string, params?: Record<string, unknown>) => {
    if (key === 'dashboard.workbench.operational.needsReview') {
      return `${params?.count ?? 0} items need confirmation`;
    }
    if (key === 'dashboard.workbench.operational.distribution') {
      return `${params?.warning ?? 0} warning, ${params?.unknown ?? 0} unknown`;
    }
    return key;
  },
}));

vi.mock('@/shared/observability', () => ({
  MEDIUM_DATE_TIME_FORMAT_OPTIONS: {},
  formatLocaleDateTime: (value: string) => `formatted:${value}`,
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

function mountPreview() {
  return mount(DashboardWorkbenchPreviewPage, {
    global: {
      stubs: {
        TButton: buttonStub,
        TCard: surfaceStub,
        TDrawer: surfaceStub,
        TList: surfaceStub,
        TListItem: surfaceStub,
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
    expect(wrapper.text()).toContain('2 items need confirmation');
    expect(wrapper.text()).toContain('1 warning, 1 unknown');
  });

  it('keeps healthy dependencies quiet and represents missing resources as a neutral note', () => {
    const wrapper = mountPreview();

    expect(wrapper.get('[data-health-id="health-postgresql"]').attributes('data-status')).toBe('healthy');
    expect(wrapper.get('[data-health-id="health-redis"]').attributes('data-status')).toBe('healthy');
    expect(wrapper.get('.resource-note[data-evidence="missing"]').attributes('data-evidence')).toBe('missing');
    expect(wrapper.find('.t-tag').exists()).toBe(false);
    expect(wrapper.find('.health-row--emphasized').exists()).toBe(false);
  });

  it('uses shared menu icon keys for the four primary quick actions', () => {
    const wrapper = mountPreview();
    const iconKeys = wrapper.findAll('[data-icon-key]').map((icon) => icon.attributes('data-icon-key'));

    expect(iconKeys.slice(0, 4)).toEqual(['build', 'observability-overview', 'audit-trail', 'image-artifact']);
    expect(iconKeys).toContain('access-log');
  });
});
