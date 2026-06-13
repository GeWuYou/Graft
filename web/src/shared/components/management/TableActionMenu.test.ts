// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import TableActionMenu from './TableActionMenu.vue';

type DropdownStubOption = {
  content?: string;
  value?: string;
};

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => (key === 'components.commonTable.more' ? '更多' : key),
  }),
}));

const TButtonStub = defineComponent({
  name: 'TButtonStub',
  props: {
    disabled: Boolean,
  },
  emits: ['click'],
  setup(props, { emit, slots }) {
    function handleClick(event: MouseEvent) {
      emit('click', event);
    }

    return () =>
      h(
        'button',
        {
          disabled: props.disabled,
          onClick: handleClick,
        },
        slots.default?.(),
      );
  },
});

const TDropdownStub = defineComponent({
  name: 'TDropdownStub',
  props: {
    options: {
      default: () => [],
      type: Array,
    },
  },
  emits: ['click'],
  setup(props, { emit, slots }) {
    const options = props.options as DropdownStubOption[];

    return () =>
      h('div', { 'data-testid': 'action-dropdown' }, [
        slots.default?.(),
        h(
          'button',
          {
            'data-testid': 'dropdown-option',
            onClick: (event: MouseEvent) => emit('click', options[0], { e: event }),
          },
          options[0]?.content,
        ),
      ]);
  },
});

function mountInsideClickableRow() {
  const rowClick = vi.fn();

  const wrapper = mount(
    {
      components: { TableActionMenu },
      setup() {
        return { rowClick };
      },
      template: `
      <div data-testid="row" @click="rowClick">
        <TableActionMenu
          :actions="[
            { label: 'components.commonTable.detail', value: 'detail' },
            { label: 'components.commonTable.rawJson', value: 'raw-json' },
          ]"
        />
      </div>
    `,
    },
    {
      global: {
        stubs: {
          't-button': TButtonStub,
          't-dropdown': TDropdownStub,
        },
      },
    },
  );

  return { rowClick, wrapper };
}

describe('TableActionMenu', () => {
  it('uses the shared localized fallback for the more action label', () => {
    const wrapper = mount(TableActionMenu, {
      global: {
        stubs: {
          't-button': TButtonStub,
          't-dropdown': TDropdownStub,
        },
      },
      props: {
        actions: [
          {
            label: 'components.commonTable.detail',
            value: 'detail',
          },
          {
            label: 'components.commonTable.detail',
            value: 'edit',
          },
        ],
      },
    });

    expect(wrapper.get('[data-testid="action-dropdown"]').text()).toContain('更多');
  });

  it('keeps primary action clicks from bubbling to the table row', async () => {
    const { rowClick, wrapper } = mountInsideClickableRow();

    await wrapper.findAll('button')[0].trigger('click');

    expect(wrapper.findComponent(TableActionMenu).emitted('action')?.[0]).toEqual(['detail']);
    expect(rowClick).not.toHaveBeenCalled();
  });

  it('keeps the more trigger from acting like the primary row detail action', async () => {
    const { rowClick, wrapper } = mountInsideClickableRow();

    await wrapper.findAll('button')[1].trigger('click');

    expect(wrapper.findComponent(TableActionMenu).emitted('action')).toBeUndefined();
    expect(rowClick).not.toHaveBeenCalled();
  });

  it('emits dropdown item actions without bubbling to the table row', async () => {
    const { rowClick, wrapper } = mountInsideClickableRow();

    await wrapper.get('[data-testid="dropdown-option"]').trigger('click');

    expect(wrapper.findComponent(TableActionMenu).emitted('action')?.[0]).toEqual(['raw-json']);
    expect(rowClick).not.toHaveBeenCalled();
  });
});
