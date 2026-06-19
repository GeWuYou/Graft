// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { mount } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, nextTick, ref } from 'vue';

import GlobalMenuSearch from './GlobalMenuSearch.vue';

const pushMock = vi.fn();

const searchItems = [
  {
    key: 'announcement-center',
    module: 'announcement',
    navigationPath: '/announcements',
    parentTitles: ['Operations'],
    path: '/announcements',
    routeName: 'AnnouncementCenterIndex',
    title: 'Announcement Center',
    titleKey: 'announcement.center.title',
  },
];

vi.mock('vue-router', async () => {
  const actual = await vi.importActual<typeof import('vue-router')>('vue-router');
  return {
    ...actual,
    useRoute: () => ({
      path: '/dashboard',
    }),
    useRouter: () => ({
      push: pushMock,
    }),
  };
});

vi.mock('@/locales', () => ({
  t: (key: string) =>
    (
      ({
        'global.search.empty': 'No searchable menu items available',
        'global.search.idle': 'Type to search page entries',
        'global.search.noResults': 'No results',
        'global.search.placeholder': 'Search menus or pages',
        'global.search.trigger': 'Search',
      }) as Record<string, string>
    )[key] ?? key,
}));

vi.mock('@/shared/composables/useGlobalMenuSearch', () => ({
  normalizeGlobalMenuSearchKeyword: (keyword: string) => keyword.trim().toLowerCase(),
  useGlobalMenuSearch: () => ({
    routesInitialized: ref(true),
    searchIndex: ref(searchItems),
    searchItems: (keyword: string) =>
      keyword.trim()
        ? searchItems.filter((item) => item.title.toLowerCase().includes(keyword.trim().toLowerCase()))
        : [],
  }),
}));

const componentStubs = {
  teleport: false,
  't-button': defineComponent({
    emits: ['click'],
    setup(_, { emit, slots }) {
      return () =>
        h(
          'button',
          {
            type: 'button',
            onClick: () => emit('click'),
          },
          slots.default?.(),
        );
    },
  }),
  't-icon': defineComponent({
    setup() {
      return () => h('i');
    },
  }),
  't-input': defineComponent({
    props: {
      modelValue: {
        default: '',
        type: String,
      },
      placeholder: {
        default: '',
        type: String,
      },
    },
    emits: ['enter', 'keydown', 'update:modelValue'],
    setup(props, { emit, expose, slots }) {
      const inputRef = ref<HTMLInputElement | null>(null);

      expose({
        focus() {
          inputRef.value?.focus();
        },
      });

      return () =>
        h('div', { class: 't-input-stub' }, [
          slots['prefix-icon']?.(),
          h('input', {
            ref: inputRef,
            class: 't-input__inner',
            placeholder: props.placeholder,
            value: props.modelValue,
            onInput: (event: Event) => {
              emit('update:modelValue', (event.target as HTMLInputElement).value);
            },
            onKeydown: (event: KeyboardEvent) => {
              emit('keydown', props.modelValue, { e: event });
              if (event.key === 'Enter') {
                emit('enter', props.modelValue, { e: event });
              }
            },
          }),
        ]);
    },
  }),
  't-loading': defineComponent({
    setup() {
      return () => h('div', 'loading');
    },
  }),
  't-tooltip': defineComponent({
    setup(_, { slots }) {
      return () => h('div', slots.default?.());
    },
  }),
};

describe('GlobalMenuSearch', () => {
  beforeEach(() => {
    pushMock.mockReset();
  });

  afterEach(() => {
    document.body.innerHTML = '';
  });

  it('uses a compact trigger by default and expands the input after clicking search', async () => {
    const wrapper = mount(GlobalMenuSearch, {
      attachTo: document.body,
      global: {
        stubs: componentStubs,
      },
    });

    expect(wrapper.find('.header-menu-search-left__button').exists()).toBe(true);
    expect(wrapper.find('.header-search.width-zero').exists()).toBe(true);
    expect(document.body.querySelector('.global-menu-search__panel-layer')).toBeNull();

    await wrapper.get('button').trigger('click');
    await nextTick();

    expect(wrapper.find('.header-menu-search-left').classes()).toContain('is-open');
    expect(wrapper.find('.header-search').exists()).toBe(true);
    expect(wrapper.find('.header-search.width-zero').exists()).toBe(false);
    expect(wrapper.find('.header-menu-search-left__button.search-icon-hide').exists()).toBe(true);
    expect(document.body.querySelector('.global-menu-search__panel-layer')).not.toBeNull();
    expect(wrapper.find('.global-menu-search__panel-layer').exists()).toBe(false);

    document.body.dispatchEvent(new Event('pointerdown', { bubbles: true }));
    await nextTick();

    expect(document.body.querySelector('.global-menu-search__panel-layer')).toBeNull();
    expect(wrapper.find('.header-search.width-zero').exists()).toBe(true);
    expect(wrapper.find('.header-menu-search-left__button').exists()).toBe(true);

    wrapper.unmount();
  });

  it('navigates to a matching result and closes the result panel', async () => {
    const wrapper = mount(GlobalMenuSearch, {
      attachTo: document.body,
      global: {
        stubs: componentStubs,
      },
    });

    await wrapper.get('button').trigger('click');
    await nextTick();

    const input = document.body.querySelector<HTMLInputElement>('.t-input__inner');
    expect(input).not.toBeNull();
    input!.value = 'announcement';
    input!.dispatchEvent(new Event('input', { bubbles: true }));
    await nextTick();

    const result = document.body.querySelector<HTMLButtonElement>('.global-menu-search-result');
    expect(result).not.toBeNull();
    result!.click();
    await nextTick();

    expect(pushMock).toHaveBeenCalledWith('/announcements');
    expect(document.body.querySelector('.global-menu-search__panel-layer')).toBeNull();

    wrapper.unmount();
  });
});
