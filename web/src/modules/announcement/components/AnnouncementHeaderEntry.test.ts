// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h, nextTick } from 'vue';

import AnnouncementHeaderEntry from './AnnouncementHeaderEntry.vue';

const pushMock = vi.fn();

vi.mock('vue-router', async () => {
  const actual = await vi.importActual<typeof import('vue-router')>('vue-router');
  return {
    ...actual,
    useRouter: () => ({
      push: pushMock,
    }),
  };
});

vi.mock('../api/announcement', () => ({
  getAnnouncementUnreadCount: vi.fn(async () => ({ count: 3 })),
}));

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}));

const componentStubs = {
  't-tooltip': defineComponent({
    setup(_, { slots }) {
      return () => h('div', slots.default?.());
    },
  }),
  TTooltip: defineComponent({
    setup(_, { slots }) {
      return () => h('div', slots.default?.());
    },
  }),
  't-badge': defineComponent({
    props: {
      count: { type: Number, default: 0 },
    },
    setup(props, { slots }) {
      return () => h('div', { 'data-count': String(props.count) }, slots.default?.());
    },
  }),
  't-button': defineComponent({
    props: {
      ariaLabel: { type: String, default: '' },
      loading: { type: Boolean, default: false },
      title: { type: String, default: '' },
    },
    emits: ['click'],
    setup(props, { emit, slots }) {
      return () =>
        h(
          'button',
          {
            'aria-label': props.ariaLabel,
            title: props.title,
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
};

describe('AnnouncementHeaderEntry', () => {
  it('loads unread count and opens the announcement route', async () => {
    pushMock.mockReset();
    const wrapper = mount(AnnouncementHeaderEntry, {
      global: {
        stubs: componentStubs,
      },
    });

    await nextTick();
    await nextTick();

    expect(wrapper.get('[data-count]').attributes('data-count')).toBe('3');
    expect(wrapper.get('button').attributes('aria-label')).toBe('announcement.header.title');
    await wrapper.get('button').trigger('click');
    expect(pushMock).toHaveBeenCalledWith('/announcements');
  });
});
