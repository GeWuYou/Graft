import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';

import MetricUsageBar from './MetricUsageBar.vue';

describe('MetricUsageBar', () => {
  it('exposes percent-based ARIA meter values', () => {
    const wrapper = mount(MetricUsageBar, {
      props: {
        label: 'CPU',
        max: 200,
        value: 50,
      },
    });

    expect(wrapper.attributes('role')).toBe('meter');
    expect(wrapper.attributes('aria-valuemin')).toBe('0');
    expect(wrapper.attributes('aria-valuemax')).toBe('100');
    expect(wrapper.attributes('aria-valuenow')).toBe('25');
  });
});
