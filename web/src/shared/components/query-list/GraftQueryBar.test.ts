import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import { defineComponent, h, ref } from 'vue';
import { createI18n } from 'vue-i18n';

import type { GraftQueryState } from './graft-query-bar';
import GraftQueryBar from './GraftQueryBar.vue';

const valueStub = defineComponent({
  props: ['modelValue', 'value'],
  emits: ['update:modelValue', 'update:value', 'enter'],
  setup(_, { attrs, emit, slots }) {
    return () => h('button', { ...attrs, onClick: () => emit('update:modelValue', 'updated') }, slots.default?.());
  },
});
const passthroughStub = defineComponent({
  setup(_, { slots }) {
    return () => h('div', slots.default?.());
  },
});
const buttonStub = defineComponent({
  setup(_, { attrs, slots }) {
    return () => h('button', attrs, slots.default?.());
  },
});
const tagStub = defineComponent({
  emits: ['close'],
  setup(_, { emit, slots }) {
    return () => h('button', { onClick: () => emit('close') }, slots.default?.());
  },
});
const i18n = createI18n({
  legacy: false,
  locale: 'en-US',
  messages: {
    'en-US': {
      app: {
        queryBar: {
          searchPlaceholder: 'Search',
          moreFilters: 'More Filters',
          search: 'Search',
          reset: 'Reset',
          applied: 'Applied:',
          clearAll: 'Clear All',
          clear: 'Clear',
          apply: 'Apply',
          yes: 'Yes',
          no: 'No',
        },
      },
    },
  },
});

function mountBar(
  modelValue: GraftQueryState = { keyword: '', filters: { status: 'running' }, page: 3, pageSize: 20 },
) {
  const model = ref<GraftQueryState>(modelValue);
  const wrapper = mount(GraftQueryBar, {
    props: {
      modelValue: model.value,
      config: {
        filters: [
          { key: 'status', label: 'Status', type: 'select', options: [{ label: 'Running', value: 'running' }] },
          {
            key: 'health',
            label: 'Health',
            type: 'multi-select',
            options: [
              { label: 'Healthy', value: 'healthy' },
              { label: 'Unhealthy', value: 'unhealthy' },
            ],
          },
        ],
        quickFilters: [{ key: 'failed', label: 'Failed', patch: { status: 'failed' } }],
      },
      'onUpdate:modelValue': (value) => {
        model.value = value;
        wrapper.setProps({ modelValue: value });
      },
    },
    global: {
      plugins: [i18n],
      stubs: {
        't-input': valueStub,
        't-select': valueStub,
        't-date-range-picker': valueStub,
        't-input-number': valueStub,
        't-switch': valueStub,
        't-popup': passthroughStub,
        't-drawer': passthroughStub,
        't-button': buttonStub,
        't-tag': tagStub,
      },
    },
  });
  return { model, wrapper };
}

describe('GraftQueryBar', () => {
  it('emits applied state with the first page and renders configured active tags', async () => {
    const { wrapper } = mountBar();
    await wrapper
      .findAll('button')
      .find((button) => button.text() === 'Search')
      ?.trigger('click');
    expect(wrapper.emitted('search')?.[0]?.[0]).toMatchObject({ page: 1, filters: { status: 'running' } });
    expect(wrapper.get('[data-testid="graft-query-bar-tags"]').text()).toContain('Status=Running');
  });

  it('resets conditions and applies configured quick filters from page one', async () => {
    const { model, wrapper } = mountBar();
    await wrapper
      .findAll('button')
      .find((button) => button.text() === 'Failed')
      ?.trigger('click');
    expect(model.value).toMatchObject({ page: 1, filters: { status: 'failed' } });
    await wrapper
      .findAll('button')
      .find((button) => button.text() === 'Reset')
      ?.trigger('click');
    expect(model.value).toEqual({ keyword: '', filters: {}, page: 1, pageSize: 20 });
  });

  it('uses configured option labels for multi-select tags and removes one condition', async () => {
    const { model, wrapper } = mountBar({
      keyword: '',
      filters: { health: ['healthy', 'unhealthy'], status: 'running' },
      page: 2,
      pageSize: 20,
    });
    expect(wrapper.get('[data-testid="graft-query-bar-tags"]').text()).toContain('Health=Healthy ~ Unhealthy');
    await wrapper.findAll('[data-testid="graft-query-bar-tags"] button').at(0)?.trigger('click');
    expect(model.value).toMatchObject({ page: 1, filters: { health: ['healthy', 'unhealthy'] } });
  });

  it('does not render a filter trigger for a search-only configuration', async () => {
    const { wrapper } = mountBar({ keyword: '', filters: {}, page: 1, pageSize: 20 });
    await wrapper.setProps({ config: { placeholder: 'Search images' } });
    expect(wrapper.find('[data-testid="graft-query-bar-more"]').exists()).toBe(false);
  });
});
