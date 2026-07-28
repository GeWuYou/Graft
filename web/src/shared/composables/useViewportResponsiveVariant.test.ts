import { mount } from '@vue/test-utils';
import { afterEach, describe, expect, it } from 'vitest';
import { defineComponent, h } from 'vue';

import { useViewportResponsiveVariant } from './useViewportResponsiveVariant';

describe('useViewportResponsiveVariant', () => {
  const originalWidth = window.innerWidth;

  afterEach(() => {
    Object.defineProperty(window, 'innerWidth', { configurable: true, value: originalWidth });
  });

  it('uses the current viewport width for its initial render', () => {
    Object.defineProperty(window, 'innerWidth', { configurable: true, value: 1280 });
    const Probe = defineComponent({
      setup() {
        const variant = useViewportResponsiveVariant();
        return () => h('output', variant.value.density);
      },
    });

    const wrapper = mount(Probe);
    expect(wrapper.text()).toBe('spacious');
    wrapper.unmount();
  });
});
