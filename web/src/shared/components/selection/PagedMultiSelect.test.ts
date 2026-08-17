import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h, nextTick } from 'vue';

import PagedMultiSelect from './PagedMultiSelect.vue';
import { createExplicitSelection } from './selection-model';

const DialogStub = defineComponent({
  name: 'TDialog',
  props: ['header', 'visible'],
  emits: ['close'],
  setup(props, { emit, slots }) {
    return () =>
      h('section', { 'data-testid': 'dialog', 'data-visible': String(props.visible) }, [
        h('h2', props.header),
        h('button', { 'data-testid': 'close-dialog', onClick: () => emit('close') }, 'close'),
        slots.default?.(),
      ]);
  },
});

const InputStub = defineComponent({
  name: 'TInput',
  props: ['modelValue'],
  emits: ['update:modelValue', 'clear', 'enter'],
  setup(_props, { emit }) {
    return () =>
      h('div', [
        h('button', { 'data-testid': 'change-search', onClick: () => emit('update:modelValue', 'alpha') }, 'change'),
        h(
          'button',
          { 'data-testid': 'clear-search', onClick: () => (emit('update:modelValue', ''), emit('clear')) },
          'clear',
        ),
        h('button', { 'data-testid': 'enter-search', onClick: () => emit('enter') }, 'enter'),
      ]);
  },
});

const ButtonStub = defineComponent({
  name: 'TButton',
  props: ['disabled'],
  emits: ['click'],
  setup(props, { emit, slots }) {
    return () => h('button', { disabled: props.disabled, onClick: () => emit('click') }, slots.default?.());
  },
});

const PagedTableStub = defineComponent({
  name: 'ManagementPagedTable',
  props: ['rows', 'selectedRowKeys'],
  emits: ['select-change', 'page-change'],
  setup(props, { emit, slots }) {
    return () =>
      h('div', [
        h('button', { 'data-testid': 'select-page', onClick: () => emit('select-change', [2]) }, 'select'),
        h(
          'button',
          { 'data-testid': 'change-page', onClick: () => emit('page-change', { current: 2, pageSize: 20 }) },
          'page',
        ),
        props.rows.length === 0 ? slots.empty?.() : null,
        slots.footer?.(),
      ]);
  },
});

function mountSubject(search?: { clearLabel?: string; placeholder: string }) {
  return mount(PagedMultiSelect, {
    props: {
      cancelLabel: 'Cancel',
      columns: [],
      confirmLabel: 'Confirm',
      current: 2,
      emptyDescription: 'No rows',
      emptyTitle: 'Empty',
      keyword: '',
      pageSize: 20,
      rowKey: 'id',
      rows: [{ id: 2 }],
      search: search ?? { clearLabel: 'Clear search', placeholder: 'Search people' },
      selectedCountLabel: (count: number) => `${count} selected`,
      selection: createExplicitSelection([1]),
      title: 'Select people',
      total: 2,
      visible: true,
    },
    global: {
      stubs: {
        't-button': ButtonStub,
        't-dialog': DialogStub,
        't-input': InputStub,
        't-empty': {
          props: ['description', 'title'],
          template: '<div><span>{{ title }}</span><span>{{ description }}</span><slot name="action" /></div>',
        },
        't-pagination': { template: '<div data-testid="pagination" />' },
        't-space': { template: '<div><slot /></div>' },
        ManagementPagedTable: PagedTableStub,
      },
    },
  });
}

describe('PagedMultiSelect', () => {
  it('does not render a search surface when no search capability is supplied', async () => {
    const wrapper = mountSubject({ clearLabel: 'Clear search', placeholder: 'Search people' });
    await wrapper.setProps({ search: undefined });

    expect(wrapper.findComponent(InputStub).exists()).toBe(false);
    expect(wrapper.get('[data-testid="dialog"]').attributes('data-visible')).toBe('true');
  });

  it('debounces keyword searches and resets the current page', async () => {
    vi.useFakeTimers();
    const wrapper = mountSubject();

    await wrapper.get('[data-testid="change-search"]').trigger('click');
    await nextTick();
    vi.advanceTimersByTime(299);
    expect(wrapper.emitted('search')).toBeUndefined();

    vi.advanceTimersByTime(1);
    expect(wrapper.emitted('search')).toEqual([['alpha']]);
    expect(wrapper.emitted('update:current')).toEqual([[1]]);
    vi.useRealTimers();
  });

  it('runs an immediate search on Enter and cancels a pending debounce', async () => {
    vi.useFakeTimers();
    const wrapper = mountSubject();

    await wrapper.get('[data-testid="change-search"]').trigger('click');
    await wrapper.get('[data-testid="enter-search"]').trigger('click');
    vi.advanceTimersByTime(300);

    expect(wrapper.emitted('search')).toEqual([['alpha']]);
    vi.useRealTimers();
  });

  it('clears and immediately searches from the first page', async () => {
    vi.useFakeTimers();
    const wrapper = mountSubject();

    await wrapper.get('[data-testid="change-search"]').trigger('click');
    await wrapper.get('[data-testid="clear-search"]').trigger('click');
    vi.advanceTimersByTime(300);

    expect(wrapper.emitted('search')).toEqual([['']]);
    expect(wrapper.emitted('update:current')).toEqual([[1]]);
    vi.useRealTimers();
  });

  it('does not let a clear without a keyword suppress the next keyword search', async () => {
    vi.useFakeTimers();
    const wrapper = mountSubject();

    await wrapper.get('[data-testid="clear-search"]').trigger('click');
    await wrapper.get('[data-testid="change-search"]').trigger('click');
    vi.advanceTimersByTime(300);

    expect(wrapper.emitted('search')).toEqual([[''], ['alpha']]);
    vi.useRealTimers();
  });

  it('renders search-specific empty content and a clear action for a keyword query', async () => {
    const wrapper = mountSubject();

    await wrapper.setProps({
      keyword: 'alpha',
      rows: [],
      searchEmptyDescription: 'No matching people',
      searchEmptyTitle: 'No matches',
      total: 0,
    });

    expect(wrapper.text()).toContain('No matches');
    expect(wrapper.text()).toContain('No matching people');
    expect(wrapper.findAll('button').some((button) => button.text() === 'Clear search')).toBe(true);
  });

  it('retains selections from previous pages while replacing the active page selection', async () => {
    const wrapper = mountSubject();

    await wrapper.get('[data-testid="select-page"]').trigger('click');

    expect(wrapper.emitted('update:selection')).toHaveLength(1);
    expect(wrapper.emitted('update:selection')?.[0]?.[0]).toMatchObject({ mode: 'explicit' });
    expect(
      Array.from((wrapper.emitted('update:selection')?.[0]?.[0] as { selectedIds: Set<number> }).selectedIds),
    ).toEqual([1, 2]);
  });

  it('keeps the selection summary with pagination and reports selections beyond an optional maximum', async () => {
    const wrapper = mountSubject();
    await wrapper.setProps({ maxSelection: 1 });

    await wrapper.get('[data-testid="select-page"]').trigger('click');

    expect(wrapper.emitted('update:selection')).toBeUndefined();
    expect(wrapper.emitted('selection-limit')).toEqual([[{ attemptedCount: 2, maxSelection: 1 }]]);
    expect(wrapper.get('.paged-multi-select__data-footer').text()).toContain('1 selected');
    expect(wrapper.findAll('[data-testid="pagination"]')).toHaveLength(1);
  });

  it('emits page changes and cancellation without taking ownership of the data request', async () => {
    const wrapper = mountSubject();

    await wrapper.get('[data-testid="change-page"]').trigger('click');
    await wrapper.get('[data-testid="close-dialog"]').trigger('click');

    expect(wrapper.emitted('page-change')).toEqual([[{ current: 2, pageSize: 20 }]]);
    expect(wrapper.emitted('cancel')).toEqual([[]]);
  });
});
