import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h, nextTick } from 'vue';

import type { AnnouncementViewModel } from '../domain/announcement-presenter';
import AnnouncementReadPanel from './AnnouncementReadPanel.vue';

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}));

vi.mock('@/shared/components/markdown', () => ({
  MarkdownViewer: defineComponent({
    props: {
      source: { type: String, default: '' },
    },
    setup(props) {
      return () => h('article', { class: 'markdown-viewer-stub' }, props.source);
    },
  }),
}));

const buttonStub = defineComponent({
  emits: ['click'],
  setup(_, { emit, slots }) {
    return () => h('button', { type: 'button', onClick: () => emit('click') }, slots.default?.());
  },
});

const componentStubs = {
  teleport: true,
  't-button': buttonStub,
  't-icon': defineComponent({ setup: () => () => h('i') }),
  't-tag': defineComponent({
    setup:
      (_, { slots }) =>
      () =>
        h('span', slots.default?.()),
  }),
};

const unreadAnnouncement: AnnouncementViewModel = {
  content: '# Long content',
  createdAtLabel: '2026-06-12 10:00:00',
  deliveryMode: 'popup',
  deliveryModeLabel: 'Popup',
  archivedAtLabel: 'Not Set',
  expireAtLabel: 'Not Set',
  id: 7,
  level: 'warning',
  levelLabel: 'Warning',
  levelTheme: 'warning',
  pinned: true,
  pinnedLabel: 'Pinned',
  publishedAtLabel: '2026-06-12 10:30:00',
  publishedByLabel: '7',
  publishAtLabel: '2026-06-12 11:00:00',
  readAtLabel: 'Not Set',
  status: 'published',
  statusLabel: 'Published',
  statusTheme: 'success',
  summary: 'Long content',
  title: 'Panel title',
  unread: true,
  unreadLabel: 'Unread',
  updatedAtLabel: '2026-06-12 12:00:00',
  visibility: 'visible',
  visibilityLabel: 'Visible',
  visibilityTheme: 'success',
};

describe('AnnouncementReadPanel', () => {
  it('renders announcement metadata and emits panel actions', async () => {
    const wrapper = mount(AnnouncementReadPanel, {
      props: {
        announcement: unreadAnnouncement,
        source: 'popup',
        visible: true,
      },
      global: { stubs: componentStubs },
    });

    expect(wrapper.text()).toContain('Panel title');
    expect(wrapper.text()).toContain('Warning');
    expect(wrapper.text()).toContain('Unread');
    expect(wrapper.text()).toContain('Pinned');
    expect(wrapper.text()).toContain('# Long content');

    const markReadButton = wrapper
      .findAll('button')
      .find((button) => button.text() === 'announcement.readPanel.markRead');
    expect(markReadButton).toBeDefined();
    await markReadButton!.trigger('click');
    expect(wrapper.emitted('mark-read')).toHaveLength(1);

    const openCenterButton = wrapper
      .findAll('button')
      .find((button) => button.text() === 'announcement.readPanel.openCenter');
    expect(openCenterButton).toBeDefined();
    await openCenterButton!.trigger('click');
    expect(wrapper.emitted('open-center')).toHaveLength(1);

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
    await nextTick();
    expect(wrapper.emitted('close')).toHaveLength(1);
  });

  it('hides mark-read and open-center actions when reading a read center announcement', () => {
    const wrapper = mount(AnnouncementReadPanel, {
      props: {
        announcement: {
          ...unreadAnnouncement,
          readAtLabel: '2026-06-12 12:30:00',
          unread: false,
          unreadLabel: 'Read',
        },
        source: 'center',
        visible: true,
      },
      global: { stubs: componentStubs },
    });

    expect(wrapper.text()).toContain('Read');
    expect(wrapper.text()).not.toContain('announcement.readPanel.markRead');
    expect(wrapper.text()).not.toContain('announcement.readPanel.openCenter');
  });

  it('does not emit close for Escape while hidden', async () => {
    const wrapper = mount(AnnouncementReadPanel, {
      props: {
        announcement: unreadAnnouncement,
        source: 'popup',
        visible: false,
      },
      global: { stubs: componentStubs },
    });

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
    await nextTick();

    expect(wrapper.emitted('close')).toBeUndefined();
  });
});
