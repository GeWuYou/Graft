// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h, nextTick } from 'vue';

import AnnouncementHeaderEntry from './AnnouncementHeaderEntry.vue';

const pushMock = vi.fn();

async function flushPromises() {
  await new Promise((resolve) => setTimeout(resolve, 0));
}

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
  getMyAnnouncements: vi.fn(async () => ({
    items: [
      {
        content: 'Header content',
        created_at: '2026-06-12T00:00:00Z',
        delivery_mode: 'popup',
        id: 7,
        level: 'warning',
        pinned: true,
        publish_at: '2026-06-12T01:00:00Z',
        read_at: null,
        status: 'published',
        title: 'header announcement',
        unread: true,
        updated_at: '2026-06-12T01:00:00Z',
      },
    ],
    page: 1,
    page_size: 1,
    total: 1,
  })),
  markAnnouncementRead: vi.fn(async () => ({
    content: 'Header content',
    created_at: '2026-06-12T00:00:00Z',
    delivery_mode: 'popup',
    id: 7,
    level: 'warning',
    pinned: true,
    publish_at: '2026-06-12T01:00:00Z',
    read_at: '2026-06-12T02:00:00Z',
    status: 'published',
    title: 'header announcement',
    unread: false,
    updated_at: '2026-06-12T01:00:00Z',
  })),
}));

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    locale: { value: 'en-US' },
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
  AnnouncementReadPanel: defineComponent({
    props: {
      announcement: { type: Object, default: null },
      visible: { type: Boolean, default: false },
    },
    emits: ['open-center', 'mark-read'],
    setup(props, { emit }) {
      return () =>
        props.visible
          ? h('section', { 'data-testid': 'read-panel', 'data-title': props.announcement?.title }, [
              h('button', { class: 'open-center', type: 'button', onClick: () => emit('open-center') }, 'center'),
              h('button', { class: 'mark-read', type: 'button', onClick: () => emit('mark-read') }, 'mark'),
            ])
          : null;
    },
  }),
};

describe('AnnouncementHeaderEntry', () => {
  it('loads unread count and opens latest unread announcement first', async () => {
    const api = await import('../api/announcement');
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
    await flushPromises();
    await nextTick();

    expect(api.getMyAnnouncements).toHaveBeenCalledWith({ page: 1, page_size: 1, unread_only: true });
    const panel = wrapper.findComponent(componentStubs.AnnouncementReadPanel);
    expect(panel.props('visible')).toBe(true);
    expect((panel.props('announcement') as { title?: string }).title).toBe('header announcement');

    await panel.vm.$emit('open-center');
    expect(pushMock).toHaveBeenCalledWith('/announcements');
  });

  it('keeps the announcement trigger inside the standard header entry layout', async () => {
    const wrapper = mount(AnnouncementHeaderEntry, {
      global: {
        stubs: componentStubs,
      },
    });

    await nextTick();

    expect(wrapper.classes()).toContain('announcement-header-entry');
    expect(wrapper.find('[data-count]').exists()).toBe(true);
    expect(wrapper.find('button').exists()).toBe(true);
  });
});
