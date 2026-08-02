import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h, ref } from 'vue';
import { createI18n } from 'vue-i18n';

vi.mock('@/shared/composables/useContainerSize', () => ({
  useContainerSize: () => ref({ height: 640, width: 1200 }),
}));

import ResourceQueryPanel from './ResourceQueryPanel.vue';
import type { ResourceQueryFilterDefinition, ResourceQueryState } from './types';

const valueStub = defineComponent({
  props: ['modelValue', 'value'],
  emits: ['update:modelValue', 'update:value', 'enter'],
  setup(props, { attrs, emit, slots }) {
    return () =>
      h(
        'button',
        {
          ...attrs,
          modelvalue: props.modelValue,
          value: props.value,
          onClick: () => emit('update:modelValue', 'updated'),
        },
        slots.default?.(),
      );
  },
});
const passthroughStub = defineComponent({
  setup(_, { slots }) {
    return () => h('div', [slots.default?.(), slots.content?.()]);
  },
});
const popupStub = defineComponent({
  props: { visible: Boolean },
  emits: ['update:visible'],
  setup(props, { emit, slots }) {
    return () =>
      h(
        'div',
        {
          onClick: () => emit('update:visible', !props.visible),
        },
        [slots.default?.(), slots.content?.()],
      );
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
        't-popup': popupStub,
        't-drawer': passthroughStub,
        't-button': buttonStub,
        't-tag': tagStub,
      },
    },
  });
  return { model, wrapper };
}

describe('ResourceQueryPanel', () => {
  it('uses the inline command-bar layout for wide query surfaces', () => {
    const { wrapper } = mountPanel();

    expect(wrapper.get('[data-testid="resource-query-builder-trigger"]').attributes('size')).toBe('medium');
    expect(wrapper.get('[data-testid="resource-query-search"]').attributes('size')).toBe('medium');
    expect(wrapper.get('[data-testid="resource-query-reset"]').attributes('size')).toBe('medium');
    expect(wrapper.get('.resource-query-panel__main').attributes('data-command-layout')).toBe('inline');
  });

  it('applies configured filters from the first page and renders active tags without redundant actions', async () => {
    const { wrapper } = mountPanel();
    await wrapper.get('[data-testid="resource-query-search"]').trigger('click');
    expect(wrapper.emitted('search')?.[0]?.[0]).toMatchObject({ page: 1, filters: { status: 'running' } });
    expect(wrapper.get('[data-testid="resource-query-tags"]').text()).toContain('Status=Running');
    expect(wrapper.get('[data-testid="resource-query-tags"]').text()).not.toContain('Applied:');
    expect(wrapper.get('[data-testid="resource-query-tags"]').text()).not.toContain('Clear All');
  });

  it('keeps the active-tag row visible when no filters are active', () => {
    const { wrapper } = mountPanel({ keyword: '', filters: {}, page: 1, pageSize: 20 });

    expect(wrapper.get('[data-testid="resource-query-tags"]').text()).toBe('');
  });

  it('does not reserve a second row until page-provided simple filters are explicitly expanded', async () => {
    const { wrapper } = mountPanel({ keyword: '', filters: {}, page: 1, pageSize: 20 });
    await wrapper.setProps({ simpleFiltersVisible: true });
    expect(wrapper.find('.resource-query-panel__simple-filters').exists()).toBe(false);
  });

  it('binds boolean filters through the switch modelValue contract', async () => {
    const { wrapper } = mountPanel({ keyword: '', filters: { enabled: true }, page: 1, pageSize: 20 }, [
      { key: 'enabled', label: 'Enabled', type: 'boolean' },
    ]);

    await wrapper.get('[data-testid="resource-query-builder-trigger"]').trigger('click');
    const switchStub = wrapper.get('.resource-query-panel__field button');
    expect(switchStub.attributes('modelvalue')).toBe('true');
    expect(switchStub.attributes('value')).toBeUndefined();
  });

  it('binds each number-range input through matching value and modelValue endpoints', async () => {
    const { wrapper } = mountPanel({ keyword: '', filters: { ports: [8080, 8443] }, page: 1, pageSize: 20 }, [
      { key: 'ports', label: 'Ports', type: 'number-range' },
    ]);

    await wrapper.get('[data-testid="resource-query-builder-trigger"]').trigger('click');
    const inputs = wrapper.findAll('.resource-query-panel__number-range button');

    expect(inputs).toHaveLength(2);
    expect(inputs.map((input) => input.attributes('modelvalue'))).toEqual(['8080', '8443']);
    expect(inputs.map((input) => input.attributes('value'))).toEqual(['8080', '8443']);
  });

  it('normalizes a cleared number range to an empty filter value', async () => {
    const { model, wrapper } = mountPanel({ keyword: '', filters: { ports: [8080, ''] }, page: 1, pageSize: 20 }, [
      { key: 'ports', label: 'Ports', type: 'number-range' },
    ]);

    await wrapper.get('[data-testid="resource-query-builder-trigger"]').trigger('click');
    wrapper
      .find('.resource-query-panel__number-range')
      .findComponent(valueStub)
      .vm.$emit('update:modelValue', undefined);
    await wrapper.vm.$nextTick();

    await wrapper.get('[data-testid="resource-query-search"]').trigger('click');

    expect(model.value).toMatchObject({ filters: { ports: [] } });
  });

  it('marks the more-filters trigger as pressed while its popup is open', async () => {
    const { wrapper } = mountPanel();
    const trigger = wrapper.get('[data-testid="resource-query-builder-trigger"]');

    expect(trigger.attributes('theme')).toBe('default');
    expect(trigger.attributes('variant')).toBe('outline');
    expect(trigger.attributes('aria-expanded')).toBe('false');

    await trigger.trigger('click');

    expect(trigger.attributes('theme')).toBe('primary');
    expect(trigger.attributes('variant')).toBe('base');
    expect(trigger.attributes('aria-expanded')).toBe('true');
  });

  it('keeps the expanded filters visible after searching', async () => {
    const { wrapper } = mountPanel();
    const trigger = wrapper.get('[data-testid="resource-query-builder-trigger"]');

    await trigger.trigger('click');
    await wrapper.get('[data-testid="resource-query-search"]').trigger('click');

    expect(trigger.attributes('aria-expanded')).toBe('true');
    expect(wrapper.find('.resource-query-panel__expanded-filters').exists()).toBe(true);
  });

  it('does not render a more-filters trigger without filter definitions', () => {
    const { wrapper } = mountPanel({ keyword: '', filters: {}, page: 1, pageSize: 20 }, []);

    expect(wrapper.find('[data-testid="resource-query-builder-trigger"]').exists()).toBe(false);
  });
});
