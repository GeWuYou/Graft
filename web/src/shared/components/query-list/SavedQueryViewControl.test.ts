import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { computed, defineComponent, h, ref } from 'vue';
import { createI18n } from 'vue-i18n';

import type { SavedQueryViewController, SavedQueryViewId } from './saved-query-views';
import SavedQueryViewControl from './SavedQueryViewControl.vue';

const selectStub = defineComponent({
  name: 'TSelectStub',
  props: ['modelValue', 'inputValue', 'empty'],
  emits: ['update:modelValue', 'update:inputValue'],
  setup(props, { emit, slots }) {
    return () =>
      h('section', { 'data-testid': 'saved-view-select' }, [
        h('span', { 'data-testid': 'saved-view-search' }, String(props.inputValue ?? '')),
        h('span', { 'data-testid': 'saved-view-empty' }, String(props.empty ?? '')),
        h('button', {
          'data-testid': 'saved-view-search-2',
          onClick: () => emit('update:inputValue', '2'),
        }),
        h('button', {
          'data-testid': 'saved-view-search-missing',
          onClick: () => emit('update:inputValue', 'missing'),
        }),
        h('ul', { 'data-testid': 'saved-view-options' }, slots.default?.()),
      ]);
  },
});

const optionStub = defineComponent({
  name: 'TOptionStub',
  props: ['label', 'value'],
  setup(props) {
    return () => h('li', { 'data-value': String(props.value) }, String(props.label));
  },
});

const passthroughStub = defineComponent({
  setup(_, { slots }) {
    return () => h('div', slots.default?.());
  },
});

const i18n = createI18n({
  legacy: false,
  locale: 'en-US',
  messages: {
    'en-US': {
      app: {
        savedQueryViews: {
          label: 'Saved Filters',
          placeholder: 'Select a saved filter',
          noResults: 'No matching saved filters',
          namePlaceholder: 'Enter a filter name',
          nameRequired: 'Enter a filter name.',
          actions: {
            saveAs: 'Save Filter',
            update: 'Update Filter',
            delete: 'Delete Filter',
            cancel: 'Cancel',
            save: 'Save',
          },
          dialog: {
            createTitle: 'Save Filter',
            updateTitle: 'Update Filter',
            deleteTitle: 'Delete Filter',
            deleteDescription: 'Delete filter "{name}"?',
          },
        },
      },
    },
  },
});

function createController() {
  const selectedId = ref<SavedQueryViewId | undefined>(12);
  const views = ref(
    Array.from({ length: 12 }, (_, index) => ({
      id: index + 1,
      name: `View ${index + 1}`,
      state: {},
    })),
  );
  const selectedView = computed(() => views.value.find((view) => view.id === selectedId.value));
  return {
    applying: ref(false),
    deleting: ref(false),
    hasSelectedView: computed(() => selectedView.value !== undefined),
    isBusy: computed(() => false),
    load: vi.fn(),
    loading: ref(false),
    removeSelected: vi.fn(),
    save: vi.fn(),
    selectedId,
    selectedView,
    select: vi.fn(),
    submitting: ref(false),
    views,
  } as unknown as SavedQueryViewController<unknown, number>;
}

describe('SavedQueryViewControl', () => {
  it('limits the initial list to ten views while keeping a selected view visible', () => {
    const wrapper = mount(SavedQueryViewControl, {
      props: { controller: createController() },
      global: {
        plugins: [i18n],
        stubs: {
          't-select': selectStub,
          't-option': optionStub,
          't-button': passthroughStub,
          't-dialog': passthroughStub,
          't-input': passthroughStub,
        },
      },
    });

    const options = wrapper.findAll('[data-testid="saved-view-options"] li');
    expect(options).toHaveLength(10);
    expect(options.at(-1)?.text()).toBe('View 12');
  });

  it('filters loaded views by name and reports an empty result', async () => {
    const wrapper = mount(SavedQueryViewControl, {
      props: { controller: createController() },
      global: {
        plugins: [i18n],
        stubs: {
          't-select': selectStub,
          't-option': optionStub,
          't-button': passthroughStub,
          't-dialog': passthroughStub,
          't-input': passthroughStub,
        },
      },
    });

    await wrapper.get('[data-testid="saved-view-search-2"]').trigger('click');

    expect(wrapper.findAll('[data-testid="saved-view-options"] li').map((item) => item.text())).toEqual([
      'View 2',
      'View 12',
    ]);

    await wrapper.get('[data-testid="saved-view-search-missing"]').trigger('click');

    expect(wrapper.findAll('[data-testid="saved-view-options"] li').map((item) => item.text())).toEqual([]);
    expect(wrapper.get('[data-testid="saved-view-empty"]').text()).toBe('No matching saved filters');
  });
});
