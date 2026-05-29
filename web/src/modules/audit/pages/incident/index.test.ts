import { flushPromises, mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';
import { createI18n } from 'vue-i18n';

import IncidentPage from './index.vue';

const routerMocks = vi.hoisted(() => ({
  push: vi.fn(),
}));

const routeMocks = vi.hoisted(() => ({
  params: { eventId: '42' },
  query: {
    monitorView: 'overview',
    monitorTrendRange: '10m',
    monitorAnomalyKey: 'resource_cpu_pressure',
    monitorScopeRef: 'runtime:cpu',
  },
}));

const auditApiMocks = vi.hoisted(() => ({
  getAuditIncident: vi.fn(async () => ({
    seed_event: {
      id: 42,
      action: 'auth.failed',
      request_id: 'req-42',
      created_at: '2026-05-27T08:12:00Z',
      result: 'FAILED',
      risk_level: 'HIGH',
      source: 'SECURITY_EVENT',
      actor_display_name: 'ops-admin',
      resource_name: 'console',
      metadata: {},
    },
    incident: {
      incident_key: 'incident-auth-failed-42',
      title: 'Authentication failures around console access',
      summary: 'Correlated failed sign-ins within the same bounded investigation window.',
      risk_level: 'HIGH',
      started_at: '2026-05-27T08:00:00Z',
      ended_at: '2026-05-27T08:12:00Z',
      correlation_reason: 'Repeated authentication failures on the same entry point.',
    },
    related_events: [
      {
        id: 43,
        action: 'auth.failed',
        request_id: 'req-43',
        created_at: '2026-05-27T08:10:00Z',
        result: 'FAILED',
        risk_level: 'HIGH',
        source: 'SECURITY_EVENT',
        actor_display_name: 'ops-admin',
        resource_name: 'console',
        metadata: {},
      },
    ],
    related_actors: [{ actor_display_name: 'ops-admin', event_count: 2 }],
    related_resources: [{ resource_type: 'AUTH', resource_id: 'console', resource_name: 'console', event_count: 2 }],
    related_requests: [
      {
        request_id: 'req-42',
        event_count: 1,
        started_at: '2026-05-27T08:12:00Z',
        ended_at: '2026-05-27T08:12:30Z',
      },
    ],
    monitor_context: {
      state: 'partial',
      reason: 'Monitor retention still covers the tail of this incident window.',
    },
  })),
}));

vi.mock('../../api/audit', () => ({
  getAuditIncident: auditApiMocks.getAuditIncident,
}));

vi.mock('@/modules/shared/localized-api-error', () => ({
  resolveLocalizedErrorMessage: () => 'load failed',
}));

vi.mock('@/utils/logger', () => ({
  createLogger: () => ({
    error: vi.fn(),
  }),
}));

vi.mock('vue-router', () => ({
  useRoute: () => routeMocks,
  useRouter: () => ({
    push: routerMocks.push,
  }),
}));

const passthroughStub = defineComponent({
  name: 'PassthroughStub',
  props: ['title', 'description', 'column', 'label', 'gutter', 'xs', 'xl'],
  emits: ['click'],
  setup(props, { slots, emit }) {
    return () =>
      h(
        'div',
        {
          onClick: () => emit('click'),
        },
        [props.title, props.description, props.label, slots.default?.(), slots.actions?.()],
      );
  },
});

const listStub = defineComponent({
  name: 'TListStub',
  setup(_, { slots }) {
    return () => h('div', slots.default?.());
  },
});

const listItemStub = defineComponent({
  name: 'TListItemStub',
  setup(_, { slots }) {
    return () => h('div', slots.default?.());
  },
});

const buttonStub = defineComponent({
  name: 'TButtonStub',
  emits: ['click'],
  setup(_, { emit, slots }) {
    return () => h('button', { onClick: () => emit('click') }, slots.default?.());
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
      menu: { audit: { title: 'Security Audit' } },
      audit: {
        common: {
          unknownActor: 'Unknown actor',
          risk: { HIGH: 'High' },
        },
        incident: {
          title: 'Security Incident Drilldown',
          description: 'Audit-owned incident context for one canonical security timeline seed event.',
          errorTitle: 'Audit incident drilldown is temporarily unavailable',
          loadFailed: 'Failed to load audit incident drilldown',
          invalidEventId: 'The incident seed event id is invalid.',
          actions: {
            refresh: 'Refresh',
            retry: 'Retry',
            openRequest: 'Open seed request',
            backToMonitor: 'Back to monitor',
            openMonitorContext: 'Open monitor context',
            openRelatedRequest: 'Open related request',
            openActorEvents: 'Open actor events',
            openResourceEvents: 'Open resource events',
          },
          sections: {
            summary: 'Incident Summary',
            monitorContext: 'Monitor Context',
            relatedEvents: 'Related Audit Events',
            relatedActors: 'Related Actors',
            relatedResources: 'Related Resources',
            relatedRequests: 'Related Requests',
          },
          fields: {
            riskLevel: 'Risk level',
            window: 'Incident window',
            reason: 'Correlation reason',
            seedAction: 'Seed action',
            seedResource: 'Seed resource',
          },
          monitorState: {
            available: 'Available',
            partial: 'Partial',
            unavailable: 'Unavailable',
          },
          eventCount: '{count} related events',
        },
      },
    },
  },
});

describe('AuditIncidentPage', () => {
  it('loads the canonical incident drilldown and renders the related context panels', async () => {
    routerMocks.push.mockReset();
    const wrapper = mount(IncidentPage, {
      global: {
        plugins: [i18n],
        stubs: {
          'management-page-content': passthroughStub,
          'management-page-header': passthroughStub,
          'management-empty-state': passthroughStub,
          't-button': buttonStub,
          't-space': passthroughStub,
          't-row': passthroughStub,
          't-col': passthroughStub,
          't-card': passthroughStub,
          't-descriptions': passthroughStub,
          't-descriptions-item': passthroughStub,
          't-tag': tagStub,
          't-list': listStub,
          't-list-item': listItemStub,
        },
      },
    });

    await flushPromises();

    expect(auditApiMocks.getAuditIncident).toHaveBeenCalledWith(42);
    expect(wrapper.text()).toContain('Authentication failures around console access');
    expect(wrapper.text()).toContain('Correlated failed sign-ins within the same bounded investigation window.');
    expect(wrapper.text()).toContain('Incident Summary');
    expect(wrapper.text()).toContain('Monitor Context');
    expect(wrapper.text()).toContain('Related Audit Events');
    expect(wrapper.text()).toContain('Related Actors');
    expect(wrapper.text()).toContain('Related Resources');
    expect(wrapper.text()).toContain('Related Requests');
    expect(wrapper.text()).toContain('Monitor retention still covers the tail of this incident window.');

    const buttons = wrapper.findAll('button');

    await buttons[0]!.trigger('click');
    expect(routerMocks.push).toHaveBeenCalledWith({
      path: '/server/overview',
      query: {
        monitorView: 'overview',
        monitorTrendRange: '10m',
        monitorAnomalyKey: 'resource_cpu_pressure',
        monitorScopeRef: 'runtime:cpu',
      },
    });

    await buttons[1]!.trigger('click');
    expect(routerMocks.push).toHaveBeenCalledWith({
      path: '/audit/logs',
      query: {
        requestId: 'req-42',
        monitorView: 'overview',
        monitorTrendRange: '10m',
        monitorAnomalyKey: 'resource_cpu_pressure',
        monitorScopeRef: 'runtime:cpu',
      },
    });
  });
});
