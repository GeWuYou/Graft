// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import { defineComponent, h } from 'vue';

import TableViewToolbar from './TableViewToolbar.vue';

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
        slots.icon?.() ?? slots.default?.(),
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
        stubs: {
          'refresh-icon': true,
          't-button': TButtonStub,
          't-tooltip': TTooltipStub,
          'view-column-icon': true,
          'view-module-icon': true,
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

    expect(wrapper.emitted('refresh')).toHaveLength(1);
    expect(wrapper.emitted('column-settings')).toHaveLength(1);
    expect(wrapper.emitted('density')).toHaveLength(1);
  });
});
