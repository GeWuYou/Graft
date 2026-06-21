import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';

import PageHeader from './PageHeader.vue';

describe('PageHeader', () => {
  it('does not render breadcrumb markup', () => {
    const wrapper = mount(PageHeader, {
      props: {
        titleFallback: 'Server status',
        descriptionFallback: 'Health and runtime overview',
      },
    });

    expect(wrapper.find('.page-header__title').text()).toBe('Server status');
    expect(wrapper.find('.page-header__description').text()).toBe('Health and runtime overview');
    expect(wrapper.find('.page-header__breadcrumb').exists()).toBe(false);
    expect(wrapper.find('.t-breadcrumb').exists()).toBe(false);
  });
});
