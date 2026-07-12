import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import { nextTick } from 'vue';

import MenuIcon from './MenuIcon.vue';

describe('MenuIcon', () => {
  it('renders static Iconify SVG data', async () => {
    const wrapper = mount(MenuIcon, { props: { iconKey: 'docker' } });

    await nextTick();

    expect(wrapper.find('svg').exists()).toBe(true);
    expect(wrapper.html()).toContain('M22 12.54');
  });
});
