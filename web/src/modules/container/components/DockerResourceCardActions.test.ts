import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import { defineComponent, h } from 'vue';

import DockerResourceCardActions from './DockerResourceCardActions.vue';

const TButtonStub = defineComponent({
  name: 'TButtonStub',
  props: {
    theme: { default: undefined, type: String },
    variant: { default: undefined, type: String },
  },
  emits: ['click'],
  setup(_, { emit, slots }) {
    return () => h('button', { onClick: () => emit('click') }, slots.default?.());
  },
});

const TDropdownStub = defineComponent({
  name: 'TDropdownStub',
  props: { options: { default: () => [], type: Array } },
  emits: ['click'],
  setup(props, { emit, slots }) {
    return () =>
      h('div', [
        slots.default?.(),
        h('button', { onClick: () => emit('click', (props.options as Array<unknown>)[0]) }, 'menu item'),
      ]);
  },
});

function mountActions(moreActions: Array<{ danger?: boolean; label: string; value: string }> = []) {
  return mount(DockerResourceCardActions, {
    global: { stubs: { 't-button': TButtonStub, 't-dropdown': TDropdownStub } },
    props: { detailLabel: '详情', moreActions, moreLabel: '更多' },
  });
}

describe('DockerResourceCardActions', () => {
  it('uses the themed outlined detail action and hides an empty more menu', () => {
    const wrapper = mountActions();

    expect(wrapper.findAllComponents(TButtonStub)).toHaveLength(1);
    expect(wrapper.findComponent(TButtonStub).props()).toMatchObject({ theme: 'primary', variant: 'outline' });
    expect(wrapper.findComponent(TDropdownStub).exists()).toBe(false);
  });

  it('emits card actions and maps dangerous menu entries to error semantics', async () => {
    const wrapper = mountActions([{ danger: true, label: '删除', value: 'remove' }]);

    expect(wrapper.findComponent(TDropdownStub).props('options')).toEqual([
      expect.objectContaining({ theme: 'error', value: 'remove' }),
    ]);

    await wrapper.findAll('button')[0].trigger('click');
    await wrapper.findAll('button')[2].trigger('click');

    expect(wrapper.emitted('detail')).toHaveLength(1);
    expect(wrapper.emitted('action')).toEqual([['remove']]);
  });
});
