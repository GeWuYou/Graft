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
});
