import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

vi.mock('@/shared/components/management', () => ({
  ManagementToolbar: defineComponent({
    name: 'ManagementToolbarStub',
    setup(_, { slots }) {
      return () => h('div', slots.filters?.());
    },
  }),
}));

const { default: AdvancedQueryFilterBuilder } = await import('./AdvancedQueryFilterBuilder.vue');

const passthroughStub = defineComponent({
  emits: ['click'],
  setup(_, { attrs, emit, slots }) {
    return () => h('button', { ...attrs, onClick: () => emit('click') }, slots.default?.());
  },
});
const popupStub = defineComponent({
  props: { visible: Boolean },
  emits: ['update:visible'],
  setup(props, { emit, slots }) {
    return () =>
      h('div', { onClick: () => emit('update:visible', !props.visible) }, [slots.default?.(), slots.content?.()]);
  },
});

const inputStub = defineComponent({
  emits: ['enter'],
  setup(_props, { attrs, emit }) {
    return () => h('input', { ...attrs, onKeyup: (event: KeyboardEvent) => event.key === 'Enter' && emit('enter') });
  },
});

const tagStub = defineComponent({
  name: 'TTagStub',
  emits: ['close'],
  setup(_, { emit, slots }) {
    return () =>
      h('button', { 'data-testid': 'close-filter-tag', onClick: () => emit('close', {}) }, slots.default?.());
  },
});

const collapseStub = defineComponent({
  setup(_, { slots }) {
    return () => h('section', { 'data-testid': 'compact-collapse' }, slots.default?.());
  },
});

const collapsePanelStub = defineComponent({
  props: { value: { type: String, required: true } },
  setup(_props, { slots }) {
    return () => h('section', { 'data-testid': 'compact-collapse-panel' }, [slots.header?.(), slots.default?.()]);
  },
});

const defaultProps = {
  activePreset: 'all',
  addFilterLabel: 'Add Filter',
  addSorterLabel: 'Add Sorter',
  builderHint: 'Hint',
  builderTitle: 'Filters',
  fieldValues: {},
  fields: [],
  filtersGroupLabel: 'Filters',
  keyword: '',
  keywordPlaceholder: 'Search',
  moveDownLabel: 'Move Down',
  moveUpLabel: 'Move Up',
  presetLabel: 'Presets',
  presets: [],
  removeSorterLabel: 'Remove Sorter',
  resetLabel: 'Reset',
  searchLabel: 'Search',
  selectedFieldKey: '',
  sortDirectionOptions: [],
  sortDirectionPlaceholder: 'Direction',
  sortFieldOptionsByIndex: [],
  sortFieldKey: 'sorterBuilder',
  sortFieldPlaceholder: 'Field',
  sorters: [],
  tags: [{ key: 'keyword', label: 'Keyword: active' }],
  timeFieldKey: 'timeRange',
  timeFields: [],
};

describe('AdvancedQueryFilterBuilder', () => {
  it('emits reset when any active filter tag is closed', async () => {
    const wrapper = mount(AdvancedQueryFilterBuilder, {
      props: defaultProps,
      global: {
        stubs: {
          't-button': passthroughStub,
          't-popup': popupStub,
          't-collapse': collapseStub,
          't-collapse-panel': collapsePanelStub,
          't-input': inputStub,
          't-tag': tagStub,
        },
      },
    });

    await wrapper.get('[data-testid="close-filter-tag"]').trigger('click');

    expect(wrapper.emitted('reset')).toHaveLength(1);
  });

  it('always renders the active-filter row, including when no tags are active', async () => {
    const wrapper = mount(AdvancedQueryFilterBuilder, {
      props: { ...defaultProps, compactMode: true, tags: [] },
      global: {
        stubs: {
          't-button': passthroughStub,
          't-popup': popupStub,
          't-collapse': collapseStub,
          't-collapse-panel': collapsePanelStub,
          't-input': inputStub,
          't-tag': tagStub,
        },
      },
    });

    expect(wrapper.get('[data-testid="query-filter-builder-tags"]').classes()).toContain('graft-scrollbar');
    expect(wrapper.get('[data-testid="query-filter-builder-tags"]').text()).toBe('');

    await wrapper.setProps({ tags: defaultProps.tags });

    expect(wrapper.get('[data-testid="query-filter-builder-tags"]').text()).toContain('Keyword: active');
  });

  it('keeps compact filter controls inside the collapsed panel while presets remain outside it', async () => {
    const wrapper = mount(AdvancedQueryFilterBuilder, {
      props: {
        ...defaultProps,
        compactMode: true,
        compactToggleLabel: 'Filter',
        fields: [{ key: 'status', kind: 'select', label: 'Status' }],
      },
      slots: {
        'saved-query-views': '<span data-testid="saved-view">saved</span>',
      },
      global: {
        stubs: {
          't-button': passthroughStub,
          't-popup': popupStub,
          't-collapse': collapseStub,
          't-collapse-panel': collapsePanelStub,
          't-input': inputStub,
          't-tag': tagStub,
        },
      },
    });

    expect(wrapper.find('[data-testid="compact-collapse"]').exists()).toBe(true);
    expect(wrapper.get('[data-testid="query-filter-builder-compact-toggle"]').text()).toBe('Filter (1)');
    expect(wrapper.find('[data-testid="compact-collapse-panel"] .query-filter-builder__group').exists()).toBe(true);
    expect(wrapper.findAll('.query-filter-builder__group')).toHaveLength(1);

    await wrapper
      .findAll('button')
      .find((button) => button.text().trim() === 'Search')
      ?.trigger('click');
    expect(wrapper.emitted('search')).toHaveLength(1);
  });

  it('marks the add-filter trigger as pressed while its popup is open', async () => {
    const wrapper = mount(AdvancedQueryFilterBuilder, {
      props: {
        ...defaultProps,
        fields: [{ key: 'status', kind: 'select', label: 'Status' }],
      },
      global: {
        stubs: {
          't-button': passthroughStub,
          't-popup': popupStub,
          't-collapse': collapseStub,
          't-collapse-panel': collapsePanelStub,
          't-input': inputStub,
        },
      },
    });
    const trigger = wrapper.findAll('button').find((button) => button.text().trim() === 'Add Filter');

    expect(trigger).toBeDefined();

    expect(trigger?.attributes('theme')).toBe('default');
    expect(trigger?.attributes('variant')).toBe('dashed');
    expect(trigger?.attributes('aria-expanded')).toBe('false');

    await trigger?.trigger('click');

    expect(trigger?.attributes('theme')).toBe('primary');
    expect(trigger?.attributes('variant')).toBe('base');
    expect(trigger?.attributes('aria-expanded')).toBe('true');
  });
});
