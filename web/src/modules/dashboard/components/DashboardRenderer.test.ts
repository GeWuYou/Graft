import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import type { DashboardWidget } from '../types/dashboard';
import DashboardRenderer from './DashboardRenderer.vue';

vi.mock('@/locales', () => ({
  t: (key: string) => {
    const translations: Record<string, string> = {
      'dashboard.actions.retry': 'Retry',
      'dashboard.widget.disabledDescription': 'Disabled widget',
      'dashboard.widget.empty': 'No widgets',
      'dashboard.widget.errorFallback': 'Failed',
      'dashboard.widget.errorTitle': 'Widget failed',
      'dashboard.widget.status.disabled': 'Disabled',
      'dashboard.widget.status.error': 'Error',
      'dashboard.widget.status.normal': 'Normal',
      'dashboard.widget.status.warning': 'Warning',
    };
    return translations[key] ?? key;
  },
}));

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: vi.fn(),
  }),
}));

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
  },
  setup(props, { slots }) {
    return () =>
      h('div', [props.title, props.message, props.description, slots.title?.(), slots.default?.(), slots.actions?.()]);
  },
});

const buttonStub = defineComponent({
  name: 'TButtonStub',
  emits: ['click'],
  setup(_props, { emit, slots }) {
    return () => h('button', { onClick: (event: MouseEvent) => emit('click', event) }, slots.default?.());
  },
});

function baseWidget(partial: Partial<DashboardWidget>): DashboardWidget {
  return {
    id: 'core.module-runtime-health',
    module_key: 'core',
    order: 10,
    payload: {
      summary: {
        status: 'healthy',
      },
      items: [],
    },
    size: 'medium',
    title: 'Module Health',
    type: 'health',
    ...partial,
  };
}

function mountRenderer(widgets: DashboardWidget[]) {
  return mount(DashboardRenderer, {
    props: {
      widgets,
    },
    global: {
      stubs: {
        TAlert: passthroughStub,
        TButton: buttonStub,
        TCard: passthroughStub,
        TEmpty: passthroughStub,
        TList: passthroughStub,
        TListItem: passthroughStub,
        TTag: passthroughStub,
        TTimeline: passthroughStub,
        TTimelineItem: passthroughStub,
      },
    },
  });
}

describe('DashboardRenderer', () => {
  it('sorts widgets by order and renders by widget type only', () => {
    const wrapper = mountRenderer([
      baseWidget({
        id: 'audit.recent-events',
        module_key: 'audit',
        order: 20,
        payload: { items: [], empty: 'No events' },
        title: 'Audit Events',
        type: 'timeline',
      }),
      baseWidget({
        id: 'core.module-runtime-health',
        module_key: 'core',
        order: 5,
        title: 'Module Health',
        type: 'health',
      }),
    ]);

    const text = wrapper.text();
    expect(text.indexOf('Module Health')).toBeLessThan(text.indexOf('Audit Events'));
    expect(text).toContain('No events');
  });

  it('keeps an error widget visible and emits a focused refresh request', async () => {
    const wrapper = mountRenderer([
      baseWidget({
        error: {
          code: 'LOAD_FAILED',
          message: 'Widget unavailable',
        },
        status: 'error',
      }),
    ]);

    expect(wrapper.text()).toContain('Widget failed');
    expect(wrapper.text()).toContain('Widget unavailable');

    await wrapper.find('button').trigger('click');

    expect(wrapper.emitted('refresh-widget')?.[0]).toEqual(['core.module-runtime-health']);
  });

  it('renders the empty state when no widgets are visible', () => {
    const wrapper = mountRenderer([]);

    expect(wrapper.text()).toContain('No widgets');
  });
});
