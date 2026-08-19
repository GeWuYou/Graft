import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import { defineComponent, h } from 'vue';

import ManagementBatchBar from './ManagementBatchBar.vue';

const SpaceStub = defineComponent({
  name: 'TSpaceStub',
  setup(_, { slots }) {
    return () => h('div', { class: 't-space-stub' }, slots.default?.());
  },
});

const ButtonStub = defineComponent({
  name: 'TButtonStub',
  inheritAttrs: false,
  setup(_, { attrs, emit, slots }) {
    return () => h('button', { ...attrs, onClick: () => emit('click') }, slots.default?.());
  },
});

const DropdownStub = defineComponent({
  name: 'TDropdownStub',
  props: {
    options: {
      type: Array,
      default: () => [],
    },
  },
  emits: ['click'],
  setup(props, { emit, slots }) {
    return () =>
      h('div', [
        slots.default?.(),
        ...(props.options as Array<{ content: string; value: string }>).map((option) =>
          h(
            'button',
            { 'data-testid': `compact-action-${option.value}`, onClick: () => emit('click', option) },
            option.content,
          ),
        ),
      ]);
  },
});

describe('ManagementBatchBar', () => {
  it('renders the selection summary, action slot, and clear button in one bar', () => {
    const wrapper = mount(ManagementBatchBar, {
      global: { components: { 't-button': ButtonStub, 't-space': SpaceStub } },
      props: { clearLabel: 'Cancel selection', selectedLabel: 'Selected 3 items' },
      slots: { default: '<button data-testid="batch-action">Delete</button>' },
    });

    expect(wrapper.get('.management-batch-bar__summary').text()).toBe('Selected 3 items');
    expect(wrapper.get('[data-testid="batch-action"]').text()).toBe('Delete');
    expect(wrapper.get('.management-batch-bar__actions').text()).toContain('Cancel selection');
  });

  it('emits clear from the configured clear button and forwards its test id', async () => {
    const wrapper = mount(ManagementBatchBar, {
      global: { components: { 't-button': ButtonStub, 't-space': SpaceStub } },
      props: { clearLabel: 'Cancel', clearTestId: 'batch-clear', selectedLabel: 'Selected 1 item' },
    });

    expect(wrapper.get('[data-testid="batch-clear"]').attributes('size')).toBe('small');
    await wrapper.get('[data-testid="batch-clear"]').trigger('click');

    expect(wrapper.emitted('clear')).toHaveLength(1);
  });

  it('allows a page to opt into medium-sized batch controls without changing the default', () => {
    const wrapper = mount(ManagementBatchBar, {
      global: { components: { 't-button': ButtonStub, 't-dropdown': DropdownStub, 't-space': SpaceStub } },
      props: {
        buttonSize: 'medium',
        clearLabel: 'Cancel',
        clearTestId: 'batch-clear',
        compactActionLabel: 'Batch actions',
        compactActionTestId: 'batch-actions',
        compactActions: [{ content: 'Assign roles', value: 'assign-roles' }],
        selectedLabel: 'Selected 1 item',
      },
    });

    expect(wrapper.get('[data-testid="batch-clear"]').attributes('size')).toBe('medium');
    expect(wrapper.get('[data-testid="batch-actions"]').attributes('size')).toBe('medium');
  });

  it('exposes compact batch actions through one shared dropdown trigger', async () => {
    const wrapper = mount(ManagementBatchBar, {
      global: { components: { 't-button': ButtonStub, 't-dropdown': DropdownStub, 't-space': SpaceStub } },
      props: {
        clearLabel: 'Cancel',
        compactActionLabel: 'Batch actions',
        compactActionTestId: 'batch-actions',
        compactActions: [{ content: 'Assign roles', value: 'assign-roles' }],
        selectedLabel: 'Selected 1 item',
      },
    });

    expect(wrapper.get('[data-testid="batch-actions"]').text()).toBe('Batch actions');
    expect(wrapper.get('[data-testid="batch-actions"]').attributes('size')).toBe('small');
    await wrapper.get('[data-testid="compact-action-assign-roles"]').trigger('click');

    expect(wrapper.emitted('action')).toEqual([['assign-roles']]);
  });

  it('renders and emits current-page selection shortcuts', async () => {
    const wrapper = mount(ManagementBatchBar, {
      global: { components: { 't-button': ButtonStub, 't-dropdown': DropdownStub, 't-space': SpaceStub } },
      props: {
        clearLabel: 'Clear',
        invertCurrentPageLabel: 'Invert current page',
        selectCurrentPageLabel: 'Select current page',
        selectedLabel: 'Selected 2 items',
      },
    });

    await wrapper.get('[data-testid="management-batch-select-current-page"]').trigger('click');
    await wrapper.get('[data-testid="management-batch-invert-current-page"]').trigger('click');

    expect(wrapper.emitted('select-current-page')).toHaveLength(1);
    expect(wrapper.emitted('invert-current-page')).toHaveLength(1);
  });

  it('places selection shortcuts before custom actions in compact mode', async () => {
    const wrapper = mount(ManagementBatchBar, {
      global: { components: { 't-button': ButtonStub, 't-dropdown': DropdownStub, 't-space': SpaceStub } },
      props: {
        clearLabel: 'Clear',
        compactActionLabel: 'Batch actions',
        compactActions: [{ content: 'Assign roles', value: 'assign-roles' }],
        invertCurrentPageLabel: 'Invert current page',
        selectCurrentPageLabel: 'Select current page',
        selectedLabel: 'Selected 2 items',
      },
    });

    const menuButtons = wrapper.findAll('[data-testid^="compact-action-"]').map((button) => button.text());
    expect(menuButtons).toEqual(['Select current page', 'Invert current page', 'Assign roles']);

    await wrapper.get('[data-testid="compact-action-select-current-page"]').trigger('click');
    await wrapper.get('[data-testid="compact-action-invert-current-page"]').trigger('click');
    expect(wrapper.emitted('select-current-page')).toHaveLength(1);
    expect(wrapper.emitted('invert-current-page')).toHaveLength(1);
  });
});
