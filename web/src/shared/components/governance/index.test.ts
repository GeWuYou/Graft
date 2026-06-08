import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';

import GovernanceActionPanel from './GovernanceActionPanel.vue';
import GovernanceDashboardShell from './GovernanceDashboardShell.vue';
import GovernanceSection from './GovernanceSection.vue';
import GovernanceSummaryCard from './GovernanceSummaryCard.vue';

describe('governance dashboard primitives', () => {
  it('renders shell with domain-aware hero and slots', () => {
    const wrapper = mount(GovernanceDashboardShell, {
      props: {
        domain: 'audit',
        eyebrow: 'Audit',
        title: 'Audit Overview',
        description: 'Investigation-first dashboard',
      },
      slots: {
        actions: '<button>Refresh</button>',
        summary: '<div data-testid="summary">Summary</div>',
        default: '<section data-testid="content">Body</section>',
      },
    });

    expect(wrapper.attributes('data-governance-domain')).toBe('audit');
    expect(wrapper.text()).toContain('Audit Overview');
    expect(wrapper.find('[data-testid="summary"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="content"]').exists()).toBe(true);
  });

  it('renders summary card badge and value aside', () => {
    const wrapper = mount(GovernanceSummaryCard, {
      props: {
        title: 'Failed auth',
        value: '12',
        valueAside: '24h',
        description: 'Compared with previous window',
      },
      slots: {
        badge: '<span data-testid="badge">Alert</span>',
      },
    });

    expect(wrapper.text()).toContain('Failed auth');
    expect(wrapper.text()).toContain('12');
    expect(wrapper.text()).toContain('24h');
    expect(wrapper.find('[data-testid="badge"]').exists()).toBe(true);
  });

  it('renders summary card extra content after the description', () => {
    const wrapper = mount(GovernanceSummaryCard, {
      props: {
        title: 'CPU',
        value: '0%',
        description: 'Latest sample',
      },
      slots: {
        default: '<div data-testid="usage">Usage bar</div>',
      },
    });

    expect(wrapper.text()).toContain('Latest sample');
    expect(wrapper.find('[data-testid="usage"]').exists()).toBe(true);
  });

  it('keeps section and action panel as structural wrappers', () => {
    const section = mount(GovernanceSection, {
      props: {
        title: 'Trend layer',
        description: 'Shared dashboard grammar',
        minHeight: 320,
      },
      slots: {
        actions: '<button>Filter</button>',
        default: '<div data-testid="section-body">Metrics</div>',
      },
    });

    expect(section.attributes('data-section-kind')).toBe('default');
    expect(section.text()).toContain('Trend layer');
    expect(section.find('[data-testid="section-body"]').exists()).toBe(true);

    const panel = mount(GovernanceActionPanel, {
      props: {
        title: 'Investigation entry',
      },
      slots: {
        default: '<button data-testid="action">Open logs</button>',
      },
    });

    expect(panel.text()).toContain('Investigation entry');
    expect(panel.find('[data-testid="action"]').exists()).toBe(true);
  });
});
