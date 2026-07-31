import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import { defineComponent, h } from 'vue';

import DockerCleanupLoadingHost from './DockerCleanupLoadingHost.vue';
import sourceText from './DockerCleanupLoadingHost.vue?raw';

const LoadingStub = defineComponent({
  name: 'TLoading',
  props: { loading: { type: Boolean, default: false } },
  setup(props, { slots }) {
    return () =>
      h('div', { 'data-loading': String(props.loading), 'data-testid': 'cleanup-loading' }, slots.default?.());
  },
});

describe('DockerCleanupLoadingHost', () => {
  it('keeps an empty cleanup result area measurable while the loading overlay is active', () => {
    const wrapper = mount(DockerCleanupLoadingHost, {
      props: { loading: true },
      global: { components: { 't-loading': LoadingStub } },
    });

    expect(wrapper.get('[data-testid="cleanup-loading"]').attributes('data-loading')).toBe('true');
    expect(wrapper.find('.docker-cleanup-loading-host').exists()).toBe(true);
    expect(wrapper.text()).toBe('');
    expect(sourceText).toContain('min-block-size: 10rem;');
  });

  it('renders cleanup content after loading completes', () => {
    const wrapper = mount(DockerCleanupLoadingHost, {
      props: { loading: false },
      slots: { default: '<p>candidate content</p>' },
      global: { components: { 't-loading': LoadingStub } },
    });

    expect(wrapper.get('[data-testid="cleanup-loading"]').attributes('data-loading')).toBe('false');
    expect(wrapper.text()).toContain('candidate content');
  });
});
