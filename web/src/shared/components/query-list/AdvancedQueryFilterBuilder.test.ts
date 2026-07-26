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
          't-input': inputStub,
          't-tag': tagStub,
        },
      },
    });

    await wrapper.get('[data-testid="close-filter-tag"]').trigger('click');

    expect(wrapper.emitted('reset')).toHaveLength(1);
  });

  it('keeps compact filters collapsed until the filter entry is opened', async () => {
    const wrapper = mount(AdvancedQueryFilterBuilder, {
      props: {
        ...defaultProps,
        compactMode: true,
        compactToggleLabel: 'Filter',
      },
      slots: {
        'saved-query-views': '<span data-testid="saved-view">saved</span>',
      },
      global: {
        stubs: {
          't-button': passthroughStub,
          't-input': inputStub,
          't-tag': tagStub,
        },
      },
    });

    expect(wrapper.find('[data-testid="saved-view"]').exists()).toBe(false);
    expect(wrapper.text()).not.toContain('Keyword: active');

    await wrapper.get('[data-testid="query-filter-builder-compact-toggle"]').trigger('click');

    expect(wrapper.get('[data-testid="saved-view"]').text()).toBe('saved');
    expect(wrapper.text()).toContain('Keyword: active');

    await wrapper
      .findAll('button')
      .find((button) => button.text().trim() === 'Search')
      ?.trigger('click');
    expect(wrapper.emitted('search')).toHaveLength(1);
  });
});
