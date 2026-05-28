import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import { defineComponent, h } from 'vue';
import { createI18n } from 'vue-i18n';

import AuditOverviewPage from './index.vue';

const passthroughStub = defineComponent({
  name: 'PassthroughStub',
  props: {
    title: {
      type: String,
      default: '',
    },
    subtitle: {
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
      h('div', [
        props.title,
        props.subtitle,
        props.description,
        slots.default?.(),
        slots.actions?.(),
        slots.action?.(),
      ]);
  },
});

const tagStub = defineComponent({
  name: 'TTagStub',
  setup(_, { slots }) {
    return () => h('span', slots.default?.());
  },
});

const i18n = createI18n({
  legacy: false,
  locale: 'en-US',
  messages: {
    'en-US': {
      menu: {
        audit: {
          overview: {
            title: 'Overview',
          },
        },
      },
      audit: {
        overview: {
          title: 'Audit Overview',
          description: 'Dashboard description',
          contractTag: 'Canonical Contract',
          refresh: 'Refresh',
          timelineTitle: 'Recent Events',
          timelineSubtitle: 'Timeline subtitle',
          statusSuccess: 'Succeeded',
          statusFailed: 'Failed',
          surfaceTitle: 'Focus Surfaces',
          surfaceSubtitle: 'Surface subtitle',
          guidanceTitle: 'Guidance',
          guidanceSubtitle: 'Guidance subtitle',
          cards: {
            today: { title: 'Today', subtitle: 'subtitle', meta: 'meta' },
            failed: { title: 'Failed', subtitle: 'subtitle', meta: 'meta' },
            actors: { title: 'Actors', subtitle: 'subtitle', meta: 'meta' },
            latency: { title: 'Latency', subtitle: 'subtitle', meta: 'meta' },
          },
          timeline: {
            items: {
              roleExport: { title: 'Role export', description: 'desc' },
              schedulerStop: { title: 'Scheduler stop', description: 'desc' },
              permissionReplace: { title: 'Permission replace', description: 'desc' },
            },
          },
          surfaces: {
            rbac: { title: 'RBAC', description: 'desc', value: 'Attention' },
            sessions: { title: 'Sessions', description: 'desc', value: 'Stable' },
            plugins: { title: 'Plugins', description: 'desc', value: 'Observed' },
          },
          guidance: {
            items: {
              scope: { title: 'Scope', description: 'desc' },
              trace: { title: 'Trace', description: 'desc' },
              next: { title: 'Next', description: 'desc' },
            },
          },
        },
      },
    },
  },
});

describe('AuditOverviewPage', () => {
  it('renders the overview dashboard information hierarchy', () => {
    const wrapper = mount(AuditOverviewPage, {
      global: {
        plugins: [i18n],
        stubs: {
          'management-page-content': passthroughStub,
          'management-page-header': passthroughStub,
          't-button': passthroughStub,
          't-card': passthroughStub,
          't-list': passthroughStub,
          't-list-item': passthroughStub,
          't-list-item-meta': passthroughStub,
          't-space': passthroughStub,
          't-statistic': passthroughStub,
          't-tag': tagStub,
        },
      },
    });

    expect(wrapper.attributes('data-page-type')).toBe('overview-dashboard');
    expect(wrapper.text()).toContain('Audit Overview');
    expect(wrapper.text()).toContain('Recent Events');
    expect(wrapper.text()).toContain('Focus Surfaces');
    expect(wrapper.text()).toContain('Guidance');
  });
});
