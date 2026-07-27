import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import { defineComponent, h } from 'vue';

import ManagementStatisticsBar from './ManagementStatisticsBar.vue';

const DividerStub = defineComponent({
  name: 'TDividerStub',
  setup() {
    return () => h('i', { 'data-testid': 'statistics-divider' });
  },
});

describe('ManagementStatisticsBar', () => {
  it('renders inline labels, emphasized values, and dividers without statistic cards', () => {
    const wrapper = mount(ManagementStatisticsBar, {
      global: { components: { 't-divider': DividerStub } },
      props: {
        items: [
          { label: 'Total', value: 19 },
          { label: 'Running', marker: '🟢', value: 18 },
          { label: 'Stopped', marker: '🟠', value: 1 },
        ],
      },
    });

    expect(wrapper.findAll('[data-testid="statistics-divider"]')).toHaveLength(2);
    expect(wrapper.findAll('.management-statistics-bar__value')).toHaveLength(3);
    expect(wrapper.text()).toContain('🟢Running18');
    expect(wrapper.find('.t-card').exists()).toBe(false);
  });

  it('exposes the compact summary layout without changing the item structure', () => {
    const wrapper = mount(ManagementStatisticsBar, {
      global: { components: { 't-divider': DividerStub } },
      props: { items: [{ label: 'Targets', value: 1 }], layout: 'summary' },
    });

    expect(wrapper.classes()).toContain('management-statistics-bar--summary');
    expect(wrapper.findAll('.management-statistics-bar__item')).toHaveLength(1);
  });

  it('accepts compact chip items without changing the full-width statistics source', () => {
    const wrapper = mount(ManagementStatisticsBar, {
      global: { components: { 't-divider': DividerStub } },
      props: {
        compactItems: [{ label: 'Images', value: 19 }],
        items: [{ label: 'Total Images', value: 19 }],
        layout: 'chips',
      },
    });

    expect(wrapper.classes()).toContain('management-statistics-bar--chips');
    expect(wrapper.find('.management-statistics-bar__compact-content').text()).toContain('Images');
  });

  it('falls back to the regular items when chips have no compact items', () => {
    const wrapper = mount(ManagementStatisticsBar, {
      global: { components: { 't-divider': DividerStub } },
      props: { items: [{ label: 'Total Images', value: 19 }], layout: 'chips' },
    });

    expect(wrapper.find('.management-statistics-bar__compact-content').text()).toContain('Total Images');
  });
});
