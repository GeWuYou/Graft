import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import { defineComponent, h } from 'vue';

import AdvancedQueryColumnDrawer from './AdvancedQueryColumnDrawer.vue';

const CheckboxGroupStub = defineComponent({
  name: 'CheckboxGroupStub',
  props: ['modelValue'],
  emits: ['update:modelValue'],
  setup(_props, { emit, slots }) {
    return () =>
      h('button', { 'data-testid': 'clear-columns', onClick: () => emit('update:modelValue', []) }, slots.default?.());
  },
});

describe('AdvancedQueryColumnDrawer', () => {
  it('keeps at least one supported column selected', async () => {
    const wrapper = mount(AdvancedQueryColumnDrawer, {
      global: {
        stubs: {
          't-checkbox': true,
          't-checkbox-group': CheckboxGroupStub,
          't-drawer': { template: '<section><slot /></section>' },
        },
      },
      props: {
        columns: [
          { label: 'Name', value: 'name' },
          { label: 'Status', value: 'status' },
        ],
        selectedKeys: ['name'],
        title: 'Columns',
        visible: true,
        'onUpdate:selectedKeys': (keys: string[]) => wrapper.setProps({ selectedKeys: keys }),
      },
    });

    await wrapper.get('[data-testid="clear-columns"]').trigger('click');

    expect(wrapper.props('selectedKeys')).toEqual(['name']);
  });
});
