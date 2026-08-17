import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';

import WorkbenchStatusIndicator from './WorkbenchStatusIndicator.vue';

describe('WorkbenchStatusIndicator', () => {
  it('uses visible text as the accessible name without announcing a duplicate live status', () => {
    const wrapper = mount(WorkbenchStatusIndicator, {
      props: { status: 'warning', label: 'Warning' },
    });

    expect(wrapper.attributes('role')).toBeUndefined();
    expect(wrapper.attributes('aria-label')).toBeUndefined();
    expect(wrapper.text()).toBe('Warning');
  });

  it('exposes an accessible image label when the visible status text is hidden', () => {
    const wrapper = mount(WorkbenchStatusIndicator, {
      props: { status: 'healthy', label: 'Healthy', showLabel: false },
    });

    expect(wrapper.attributes('role')).toBe('img');
    expect(wrapper.attributes('aria-label')).toBe('Healthy');
    expect(wrapper.text()).toBe('');
  });
});
