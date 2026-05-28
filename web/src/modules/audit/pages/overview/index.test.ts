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
    value: {
      type: String,
      default: '',
    },
    badge: {
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
        props.value,
        props.badge,
        slots.default?.(),
        slots.actions?.(),
        slots.action?.(),
        slots.summary?.(),
        slots.headerHint?.(),
      ]);
  },
});

const tagStub = defineComponent({
  name: 'TTagStub',
  setup(_, { slots }) {
    return () => h('span', slots.default?.());
  },
});

const radioGroupStub = defineComponent({
  name: 'TRadioGroupStub',
  props: {
    modelValue: {
      type: String,
      default: '',
    },
  },
  setup(_, { slots }) {
    return () => h('div', slots.default?.());
  },
});

const radioButtonStub = defineComponent({
  name: 'TRadioButtonStub',
  setup(_, { slots }) {
    return () => h('button', slots.default?.());
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
          description: 'Investigation workbench',
          contractTag: 'Canonical Contract',
          statusSuccess: 'Succeeded',
          statusFailed: 'Failed',
          trendTitle: 'Risk Trend Layer',
          trendSubtitle: 'Track risk trends',
          anomalyTitle: 'Anomaly Highlights',
          anomalySubtitle: 'Anomaly help',
          hotspotTitle: 'Risk Hotspots',
          hotspotSubtitle: 'Hotspot help',
          entryTitle: 'Investigation Entry Points',
          entrySubtitle: 'Entry help',
          correlationTitle: 'Cross-System Correlation',
          correlationSubtitle: 'Correlation help',
          timelineTitle: 'Investigation Timeline',
          timelineSubtitle: 'Timeline help',
          timeRanges: {
            '24h': '24h',
            '7d': '7d',
            '30d': '30d',
          },
          heroSignals: {
            queue: {
              label: 'Queue',
              meta: 'Queue meta',
              values: { '24h': '08 active cases', '7d': '17 active cases', '30d': '36 active cases' },
            },
            scope: {
              label: 'Scope',
              meta: 'Scope meta',
              values: { '24h': 'RBAC privilege drift', '7d': 'Permission deny cluster', '30d': 'Cross-plugin churn' },
            },
            correlation: {
              label: 'Correlation',
              meta: 'Correlation meta',
              values: { '24h': '03 live correlations', '7d': '06 live correlations', '30d': '11 live correlations' },
            },
          },
          cards: {
            failedAuth: {
              title: 'Failed Authentication',
              aside: 'events',
              description: 'Failed auth desc',
              badge: 'Anomaly',
              values: { '24h': '12', '7d': '53', '30d': '168' },
            },
            permissionDenied: {
              title: 'Permission Denied',
              aside: 'events',
              description: 'Permission denied desc',
              badge: 'Correlate',
              values: { '24h': '09', '7d': '44', '30d': '121' },
            },
            sensitiveActions: {
              title: 'Sensitive Operations',
              aside: 'ops',
              description: 'Sensitive desc',
              badge: 'Review',
              values: { '24h': '18', '7d': '67', '30d': '204' },
            },
            escalation: {
              title: 'Escalation Candidates',
              aside: 'cases',
              description: 'Escalation desc',
              badge: 'Escalate',
              values: { '24h': '03', '7d': '11', '30d': '29' },
            },
          },
          trends: {
            failedAuth: {
              title: 'Failed auth spike',
              description: 'desc',
              compareText: 'compare',
              values: {
                '24h': '12 failed auth events',
                '7d': '53 failed auth events',
                '30d': '168 failed auth events',
              },
              changes: { '24h': '+41%', '7d': '+18%', '30d': '+09%' },
            },
            permissionDenied: {
              title: 'Permission deny concentration',
              description: 'desc',
              compareText: 'compare',
              values: { '24h': '09 denied operations', '7d': '44 denied operations', '30d': '121 denied operations' },
              changes: { '24h': '+22%', '7d': '+11%', '30d': '+04%' },
            },
            pluginLifecycle: {
              title: 'Plugin lifecycle churn',
              description: 'desc',
              compareText: 'compare',
              values: { '24h': '04 lifecycle clusters', '7d': '09 lifecycle clusters', '30d': '21 lifecycle clusters' },
              changes: { '24h': '+16%', '7d': '+08%', '30d': '+03%' },
            },
          },
          anomalies: {
            suspiciousIp: {
              title: 'Suspicious IP sequence',
              description: 'desc',
              values: { '24h': '2 IP groups', '7d': '5 IP groups', '30d': '9 IP groups' },
            },
            tokenAnomaly: {
              title: 'Token refresh anomaly',
              description: 'desc',
              values: { '24h': '1 chain', '7d': '4 chains', '30d': '7 chains' },
            },
            privilegeEscalation: {
              title: 'Privilege escalation path',
              description: 'desc',
              values: { '24h': '3 paths', '7d': '7 paths', '30d': '15 paths' },
            },
          },
          hotspots: {
            actor: {
              group: 'Actor hotspot',
              title: 'Privileged operators',
              description: 'desc',
              scores: { '24h': 'High', '7d': 'High', '30d': 'Medium' },
              meta: { one: 'one', two: 'two', three: 'three' },
            },
            resource: {
              group: 'Resource hotspot',
              title: 'RBAC resources',
              description: 'desc',
              scores: { '24h': 'High', '7d': 'Medium', '30d': 'Medium' },
              meta: { one: 'one', two: 'two', three: 'three' },
            },
            plugin: {
              group: 'Plugin hotspot',
              title: 'Lifecycle',
              description: 'desc',
              scores: { '24h': 'Medium', '7d': 'Medium', '30d': 'High' },
              meta: { one: 'one', two: 'two', three: 'three' },
            },
          },
          entries: {
            failedAuth: { title: 'Investigate failed authentication', description: 'desc', query: 'preset query' },
            rbacChanges: { title: 'Review RBAC changes', description: 'desc', query: 'preset query' },
            sensitiveOps: { title: 'Trace sensitive operations', description: 'desc', query: 'preset query' },
            pluginOps: { title: 'Correlate plugin operations', description: 'desc', query: 'preset query' },
          },
          correlations: {
            pluginReload: {
              title: 'Plugin reload followed by deny spike',
              description: 'desc',
              nextStep: 'next',
              status: 'Needs follow-up',
            },
            authSpike: {
              title: 'Failed auth preceding role mutation',
              description: 'desc',
              nextStep: 'next',
              status: 'Escalation candidate',
            },
            monitorLink: {
              title: 'Runtime signal linked to audit churn',
              description: 'desc',
              nextStep: 'next',
              status: 'Linked',
            },
          },
          timeline: {
            items: {
              failedSignin: { title: 'Failed sign-in burst detected', description: 'desc', context: 'context' },
              roleModify: { title: 'Role membership changed', description: 'desc', context: 'context' },
              pluginReload: { title: 'Plugin reloaded during review window', description: 'desc', context: 'context' },
              permissionDenied: { title: 'Permission denied after mutation', description: 'desc', context: 'context' },
            },
          },
        },
      },
    },
  },
});

describe('AuditOverviewPage', () => {
  it('renders the investigation-first dashboard hierarchy', () => {
    const wrapper = mount(AuditOverviewPage, {
      global: {
        plugins: [i18n],
        stubs: {
          'governance-dashboard-shell': passthroughStub,
          'governance-summary-card': passthroughStub,
          'governance-section': passthroughStub,
          'governance-action-panel': passthroughStub,
          't-space': passthroughStub,
          't-tag': tagStub,
          't-radio-group': radioGroupStub,
          't-radio-button': radioButtonStub,
        },
      },
    });

    expect(wrapper.attributes('data-page-type')).toBe('overview-dashboard');
    expect(wrapper.text()).toContain('Audit Overview');
    expect(wrapper.text()).toContain('Risk Trend Layer');
    expect(wrapper.text()).toContain('Anomaly Highlights');
    expect(wrapper.text()).toContain('Risk Hotspots');
    expect(wrapper.text()).toContain('Investigation Entry Points');
    expect(wrapper.text()).toContain('Cross-System Correlation');
    expect(wrapper.text()).toContain('Investigation Timeline');
    expect(wrapper.text()).toContain('Failed auth spike');
    expect(wrapper.text()).toContain('Plugin reload followed by deny spike');
  });
});
