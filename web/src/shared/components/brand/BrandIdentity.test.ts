import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';

import BrandIdentity from './BrandIdentity.vue';

describe('BrandIdentity', () => {
  it('renders the mark and label in full mode', () => {
    const wrapper = mount(BrandIdentity, {
      props: {
        label: 'Graft',
      },
    });

    expect(wrapper.text()).toContain('Graft');
    expect(wrapper.find('.brand-identity__mark').exists()).toBe(true);
    expect(wrapper.find('.brand-identity__label').exists()).toBe(true);
  });

  it('renders only the mark in compact mode', () => {
    const wrapper = mount(BrandIdentity, {
      props: {
        compact: true,
        label: 'Graft',
        labelHidden: true,
      },
    });

    expect(wrapper.find('.brand-identity--compact').exists()).toBe(true);
    expect(wrapper.find('.brand-identity__label--hidden').exists()).toBe(true);
  });

  it('can hide the label without switching to compact centering', () => {
    const wrapper = mount(BrandIdentity, {
      props: {
        label: 'Graft',
        labelHidden: true,
      },
    });

    expect(wrapper.find('.brand-identity--compact').exists()).toBe(false);
    expect(wrapper.find('.brand-identity--label-hidden').exists()).toBe(true);
    expect(wrapper.find('.brand-identity__label--hidden').exists()).toBe(true);
  });
});
