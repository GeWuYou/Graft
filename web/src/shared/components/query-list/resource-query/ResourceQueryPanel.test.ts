import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import { defineComponent, h, ref } from 'vue';
import { createI18n } from 'vue-i18n';

import ResourceQueryPanel from './ResourceQueryPanel.vue';
import type { ResourceQueryFilterDefinition, ResourceQueryState } from './types';

const valueStub = defineComponent({
  props: ['modelValue', 'value'],
  emits: ['update:modelValue', 'update:value', 'enter'],
  setup(_, { attrs, emit, slots }) {
    return () => h('button', { ...attrs, onClick: () => emit('update:modelValue', 'updated') }, slots.default?.());
  },
});
const passthroughStub = defineComponent({
  setup(_, { slots }) {
    return () => h('div', [slots.default?.(), slots.content?.()]);
  },
});
const managementToolbarStub = defineComponent({
  setup(_, { slots }) {
    return () => h('div', [slots.filters?.(), slots.actions?.()]);
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

function mountPanel(
  modelValue: ResourceQueryState = { keyword: '', filters: { status: 'running' }, page: 3, pageSize: 20 },
  filterDefinitions: ResourceQueryFilterDefinition[] = [
    { key: 'status', label: 'Status', type: 'select' as const, options: [{ label: 'Running', value: 'running' }] },
  ],
) {
  const model = ref<ResourceQueryState>(modelValue);
  const wrapper = mount(ResourceQueryPanel, {
    props: {
      modelValue: model.value,
      config: {
        resource: 'test-resource',
        filters: filterDefinitions,
        quickFilters: [{ key: 'failed', label: 'Failed', patch: { status: 'failed' } }],
      },
      'onUpdate:modelValue': (value: ResourceQueryState) => {
        model.value = value;
        void wrapper.setProps({ modelValue: value });
      },
    },
    global: {
      plugins: [i18n],
      stubs: {
        ManagementToolbar: managementToolbarStub,
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

describe('ResourceQueryPanel', () => {
  it('applies configured filters from the first page and renders active tags', async () => {
    const { wrapper } = mountPanel();
    await wrapper.get('[data-testid="resource-query-search"]').trigger('click');
    expect(wrapper.emitted('search')?.[0]?.[0]).toMatchObject({ page: 1, filters: { status: 'running' } });
    expect(wrapper.get('[data-testid="resource-query-tags"]').text()).toContain('Status=Running');
  });

  it('does not reserve a second row until page-provided simple filters are explicitly expanded', async () => {
    const { wrapper } = mountPanel({ keyword: '', filters: {}, page: 1, pageSize: 20 });
    await wrapper.setProps({ simpleFiltersVisible: true });
    expect(wrapper.find('.resource-query-panel__simple-filters').exists()).toBe(false);
  });

  it('binds boolean filters through the switch modelValue contract', () => {
    const { wrapper } = mountPanel({ keyword: '', filters: { enabled: true }, page: 1, pageSize: 20 }, [
      { key: 'enabled', label: 'Enabled', type: 'boolean' },
    ]);

    const switchStub = wrapper.get('t-switch');
    expect(switchStub.attributes('modelvalue')).toBe('true');
    expect(switchStub.attributes('value')).toBeUndefined();
  });
});
