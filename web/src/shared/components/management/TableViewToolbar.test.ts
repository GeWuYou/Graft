import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import TableViewToolbar from './TableViewToolbar.vue';

vi.mock('tdesign-icons-vue-next', async () => {
  const { defineComponent: defineVueComponent, h: createElement } = await import('vue');
  const IconStub = defineVueComponent({
    name: 'IconStub',
    setup() {
      return () => createElement('i');
    },
  });

  return {
    RefreshIcon: IconStub,
    ViewColumnIcon: IconStub,
    ViewModuleIcon: IconStub,
  };
});

const TButtonStub = defineComponent({
  name: 'TButtonStub',
  emits: ['click'],
  setup(_props, { attrs, emit, slots }) {
    return () =>
      h(
        'button',
        {
          'aria-label': attrs['aria-label'],
          onClick: () => emit('click'),
        },
        [slots.icon?.(), slots.default?.()],
      );
  },
});

const TTooltipStub = defineComponent({
  name: 'TTooltipStub',
  setup(_props, { slots }) {
    return () => h('span', slots.default?.());
  },
});

describe('TableViewToolbar', () => {
  it('emits table view toolbar actions from shared icon buttons', async () => {
    const wrapper = mount(TableViewToolbar, {
      global: {
        components: {
          't-button': TButtonStub,
          't-tooltip': TTooltipStub,
        },
      },
      props: {
        columnSettingsLabel: 'Columns',
        densityLabel: 'Density',
        refreshLabel: 'Refresh',
      },
    });

    await wrapper.get('[aria-label="Refresh"]').trigger('click');
    await wrapper.get('[aria-label="Columns"]').trigger('click');
    await wrapper.get('[aria-label="Density"]').trigger('click');

    expect(wrapper.text()).toContain('Refresh');
    expect(wrapper.text()).toContain('Columns');
    expect(wrapper.emitted('refresh')).toHaveLength(1);
    expect(wrapper.emitted('column-settings')).toHaveLength(1);
    expect(wrapper.emitted('density')).toHaveLength(1);
  });
});
