// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { mount } from '@vue/test-utils';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, nextTick } from 'vue';

import AnnouncementPopupHost from './AnnouncementPopupHost.vue';

const pushMock = vi.fn();
const dispatchSpy = vi.spyOn(window, 'dispatchEvent');

async function flushPromises() {
  await new Promise((resolve) => setTimeout(resolve, 0));
}

vi.mock('vue-router', async () => {
  const actual = await vi.importActual<typeof import('vue-router')>('vue-router');
  return {
    ...actual,
    useRouter: () => ({ push: pushMock }),
  };
});

vi.mock('tdesign-vue-next/es/message', () => ({
  MessagePlugin: {
    error: vi.fn(),
  },
}));

vi.mock('../api/announcement', () => ({
  getMyAnnouncements: vi.fn(async () => ({
    items: [
      {
        content: 'Silent content',
        created_at: '2026-06-12T00:00:00Z',
        delivery_mode: 'silent',
        id: 6,
        level: 'info',
        pinned: false,
        publish_at: '2026-06-12T01:00:00Z',
        read_at: null,
        status: 'published',
        title: 'silent',
        unread: true,
        updated_at: '2026-06-12T01:00:00Z',
      },
      {
        content: 'Popup content',
        created_at: '2026-06-12T00:00:00Z',
        delivery_mode: 'popup',
        id: 7,
        level: 'warning',
        pinned: true,
        publish_at: '2026-06-12T01:00:00Z',
        read_at: null,
        status: 'published',
        title: 'popup',
        unread: true,
        updated_at: '2026-06-12T01:00:00Z',
      },
    ],
    page: 1,
    page_size: 10,
    total: 2,
  })),
  markAnnouncementRead: vi.fn(async () => ({
    content: 'Popup content',
    created_at: '2026-06-12T00:00:00Z',
    delivery_mode: 'popup',
    id: 7,
    level: 'warning',
    pinned: true,
    publish_at: '2026-06-12T01:00:00Z',
    read_at: '2026-06-12T02:00:00Z',
    status: 'published',
    title: 'popup',
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

const panelStub = defineComponent({
  props: {
    announcement: { type: Object, default: null },
    source: { type: String, default: '' },
    visible: { type: Boolean, default: false },
  },
  emits: ['close', 'mark-read', 'open-center'],
  setup(props, { emit }) {
    return () =>
      props.visible
        ? h('section', { 'data-source': props.source, 'data-title': props.announcement?.title }, [
            h('button', { class: 'close', type: 'button', onClick: () => emit('close') }, 'close'),
            h('button', { class: 'mark-read', type: 'button', onClick: () => emit('mark-read') }, 'mark'),
            h('button', { class: 'open-center', type: 'button', onClick: () => emit('open-center') }, 'center'),
          ])
        : null;
  },
});

describe('AnnouncementPopupHost', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it('opens only unread popup announcements and dismisses without marking read', async () => {
    const api = await import('../api/announcement');
    vi.mocked(api.markAnnouncementRead).mockClear();

    const wrapper = mount(AnnouncementPopupHost, {
      global: { stubs: { AnnouncementReadPanel: panelStub } },
    });

    await flushPromises();
    await nextTick();

    expect(api.getMyAnnouncements).toHaveBeenCalledWith({ page: 1, page_size: 10, unread_only: true });
    expect(wrapper.get('section').attributes('data-title')).toBe('popup');

    await wrapper.get('.close').trigger('click');
    expect(api.markAnnouncementRead).not.toHaveBeenCalled();
  });

  it('persists popup dismissal across host reloads', async () => {
    const firstWrapper = mount(AnnouncementPopupHost, {
      global: { stubs: { AnnouncementReadPanel: panelStub } },
    });

    await flushPromises();
    await nextTick();
    await firstWrapper.get('.close').trigger('click');
    firstWrapper.unmount();

    const secondWrapper = mount(AnnouncementPopupHost, {
      global: { stubs: { AnnouncementReadPanel: panelStub } },
    });

    await flushPromises();
    await nextTick();

    expect(secondWrapper.find('section').exists()).toBe(false);
  });

  it('marks popup announcement read, closes, and emits refresh', async () => {
    const api = await import('../api/announcement');
    vi.mocked(api.markAnnouncementRead).mockClear();
    vi.mocked(MessagePlugin.error).mockClear();
    dispatchSpy.mockClear();

    const wrapper = mount(AnnouncementPopupHost, {
      global: { stubs: { AnnouncementReadPanel: panelStub } },
    });

    await flushPromises();
    await nextTick();
    await wrapper.get('.mark-read').trigger('click');
    await flushPromises();

    expect(api.markAnnouncementRead).toHaveBeenCalledWith(7);
    expect(dispatchSpy).toHaveBeenCalledWith(expect.objectContaining({ type: 'graft:announcement-changed' }));
    expect(MessagePlugin.error).not.toHaveBeenCalled();
  });
});
