import { mount } from '@vue/test-utils';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h, nextTick } from 'vue';

import UserAnnouncementPage from './index.vue';

const dispatchSpy = vi.spyOn(window, 'dispatchEvent');

async function flushPromises() {
  await new Promise((resolve) => setTimeout(resolve, 0));
}

vi.mock('tdesign-vue-next/es/message', () => ({
  MessagePlugin: {
    error: vi.fn(),
    success: vi.fn(),
  },
}));

vi.mock('../../api/announcement', () => ({
  getAnnouncementUnreadCount: vi.fn(async () => ({ count: 1 })),
  getMyAnnouncements: vi.fn(async () => ({
    items: [
      {
        content: 'Maintenance window',
        created_at: '2026-06-12T00:00:00Z',
        delivery_mode: 'popup',
        id: 7,
        level: 'warning',
        pinned: true,
        publish_at: '2026-06-12T01:00:00Z',
        read_at: null,
        status: 'published',
        title: 'announcement.test.title',
        unread: true,
        updated_at: '2026-06-12T01:00:00Z',
      },
    ],
    page: 1,
    page_size: 20,
    total: 1,
  })),
  markAllAnnouncementsRead: vi.fn(async () => ({ updated_count: 1 })),
  markAnnouncementRead: vi.fn(async () => ({
    content: 'announcement.test.content',
    created_at: '2026-06-12T00:00:00Z',
    delivery_mode: 'popup',
    id: 7,
    level: 'warning',
    pinned: true,
    read_at: '2026-06-12T02:00:00Z',
    status: 'published',
    title: 'announcement.test.title',
    unread: false,
    updated_at: '2026-06-12T01:00:00Z',
  })),
}));

vi.mock('vue-i18n', () => ({
  createI18n: () => ({ global: { getLocaleMessage: () => ({}), t: (key: string) => key } }),
  useI18n: () => ({
    locale: { value: 'en-US' },
    t: (key: string, params?: Record<string, unknown>) => {
      const labels: Record<string, string> = {
        'announcement.user.footerTotal': `announcement.user.footerTotal:${params?.count ?? 0}`,
        'announcement.user.summary': `announcement.user.summary:${params?.count ?? 0}`,
        'announcement.user.unreadSummary': `announcement.user.unreadSummary:${params?.count ?? 0}`,
      };
      return labels[key] ?? key;
    },
  }),
}));

vi.mock('@/shared/components/management', () => ({
  ManagementEmptyState: defineComponent({
    props: {
      description: { type: String, default: '' },
      title: { type: String, default: '' },
    },
    setup(props, { slots }) {
      return () => h('div', { class: 'management-empty' }, [props.title, props.description, slots.actions?.()]);
    },
  }),
  ManagementPageContent: defineComponent({
    setup(_, { slots }) {
      return () => h('main', slots.default?.());
    },
  }),
  ManagementPageHeader: defineComponent({
    setup(_, { slots }) {
      return () => h('header', [slots.default?.(), slots.actions?.()]);
    },
  }),
  ManagementStatisticsBar: defineComponent({
    props: { items: { type: Array, default: () => [] } },
    setup(props) {
      return () => h('div', { class: 'management-statistics-bar' }, JSON.stringify(props.items));
    },
  }),
  ManagementTablePagination: defineComponent({
    props: {
      summary: { type: String, default: '' },
    },
    setup(props, { slots }) {
      return () => h('footer', [props.summary, slots.default?.()]);
    },
  }),
  ManagementToolbar: defineComponent({
    setup(_, { slots }) {
      return () => h('section', [slots.filters?.(), slots.actions?.()]);
    },
  }),
  formatCompactDateTime: (value: string) => value,
}));

const buttonStub = defineComponent({
  props: {
    disabled: { type: Boolean, default: false },
  },
  emits: ['click'],
  setup(props, { emit, slots }) {
    return () =>
      h(
        'button',
        {
          disabled: props.disabled,
          type: 'button',
          onClick: (event: MouseEvent) => emit('click', event),
        },
        slots.default?.(),
      );
  },
});

const componentStubs = {
  ResponsiveTable: defineComponent({
    setup(_, { slots }) {
      return () => h('section', { 'data-responsive-presentation': 'entity' }, [slots.default?.(), slots.cards?.()]);
    },
  }),
  't-button': buttonStub,
  't-card': defineComponent({
    setup(_, { slots }) {
      return () => h('section', [slots.header?.(), slots.default?.(), slots.footer?.()]);
    },
  }),
  't-checkbox': defineComponent({
    props: {
      modelValue: { type: Boolean, default: false },
    },
    emits: ['update:modelValue'],
    setup(props, { emit, slots }) {
      return () =>
        h('label', [
          h('input', {
            checked: props.modelValue,
            type: 'checkbox',
            onChange: (event: Event) => emit('update:modelValue', (event.target as HTMLInputElement).checked),
          }),
          slots.default?.(),
        ]);
    },
  }),
  't-empty': defineComponent({
    props: {
      description: { type: String, default: '' },
      title: { type: String, default: '' },
    },
    setup(props, { slots }) {
      return () => h('div', [props.title, props.description, slots.action?.()]);
    },
  }),
  't-list': defineComponent({
    setup(_, { slots }) {
      return () => h('div', slots.default?.());
    },
  }),
  't-list-item': defineComponent({
    setup(_, { slots }) {
      return () => h('div', slots.default?.());
    },
  }),
  't-loading': defineComponent({
    setup(_, { slots }) {
      return () => h('div', slots.default?.());
    },
  }),
  't-pagination': defineComponent({
    emits: ['change'],
    setup(_, { emit }) {
      return () => h('button', { class: 'pagination-change', type: 'button', onClick: () => emit('change') });
    },
  }),
  't-tag': defineComponent({
    setup(_, { slots }) {
      return () => h('span', slots.default?.());
    },
  }),
  't-table': defineComponent({
    props: {
      columns: { type: Array, default: () => [] },
      data: { type: Array, default: () => [] },
    },
    emits: ['row-click'],
    setup(props, { emit, slots }) {
      return () =>
        h(
          'table',
          {
            'data-row-count': props.data.length,
            'data-column-count': props.columns.length,
            onClick: (event: MouseEvent) => {
              if (props.data.length) {
                emit('row-click', { row: props.data[0], index: 0, e: event });
              }
            },
          },
          [props.data.length ? slots.title?.({ row: props.data[0] }) : null, slots.default?.()],
        );
    },
  }),
  't-tooltip': defineComponent({
    props: {
      content: { type: String, default: '' },
    },
    setup(props, { slots }) {
      return () => h('span', { title: props.content }, slots.default?.());
    },
  }),
  AnnouncementReadPanel: defineComponent({
    props: {
      announcement: { type: Object, default: null },
      visible: { type: Boolean, default: false },
    },
    emits: ['mark-read'],
    setup(props, { emit }) {
      return () =>
        props.visible
          ? h('section', { 'data-testid': 'read-panel', 'data-title': props.announcement?.title }, [
              h('button', { class: 'panel-mark-read', type: 'button', onClick: () => emit('mark-read') }, 'panel mark'),
            ])
          : null;
    },
  }),
};

describe('UserAnnouncementPage', () => {
  it('loads current-user announcements and marks one announcement read', async () => {
    const api = await import('../../api/announcement');
    vi.mocked(MessagePlugin.success).mockReset();
    dispatchSpy.mockClear();

    const wrapper = mount(UserAnnouncementPage, {
      global: {
        stubs: componentStubs,
      },
    });

    await flushPromises();
    await nextTick();

    expect(wrapper.text()).toContain('announcement.test.title');
    expect(wrapper.text()).toContain('announcement.readState.unread');
    expect(wrapper.find('[data-responsive-presentation="entity"]').exists()).toBe(true);
    expect(wrapper.get('table').attributes('data-row-count')).toBe('1');

    const markReadButton = wrapper.findAll('button').find((button) => button.text() === 'announcement.user.markRead');
    expect(markReadButton).toBeTruthy();
    await markReadButton!.trigger('click');
    await flushPromises();

    expect(api.markAnnouncementRead).toHaveBeenCalledWith(7);
    expect(MessagePlugin.success).toHaveBeenCalledWith('announcement.user.markReadSuccess');
    expect(dispatchSpy).toHaveBeenCalledWith(expect.objectContaining({ type: 'graft:announcement-changed' }));
  });

  it('opens the read panel from a list item and marks the selected announcement read', async () => {
    const api = await import('../../api/announcement');
    vi.mocked(api.markAnnouncementRead).mockClear();
    dispatchSpy.mockClear();

    const wrapper = mount(UserAnnouncementPage, {
      global: {
        stubs: componentStubs,
      },
    });

    await flushPromises();
    await nextTick();

    await wrapper.get('.announcement-user-page__item').trigger('click');
    expect(wrapper.get('[data-testid="read-panel"]').attributes('data-title')).toBe('announcement.test.title');

    await wrapper.get('.panel-mark-read').trigger('click');
    await flushPromises();

    expect(api.markAnnouncementRead).toHaveBeenCalledWith(7);
    expect(dispatchSpy).toHaveBeenCalledWith(expect.objectContaining({ type: 'graft:announcement-changed' }));
  });

  it('opens the read panel from a table row', async () => {
    const wrapper = mount(UserAnnouncementPage, {
      global: {
        stubs: componentStubs,
      },
    });

    await flushPromises();
    await nextTick();

    await wrapper.get('table').trigger('click');

    expect(wrapper.get('[data-testid="read-panel"]').attributes('data-title')).toBe('announcement.test.title');
  });

  it('toggles unread-only list query through page filters', async () => {
    const api = await import('../../api/announcement');
    vi.mocked(api.getMyAnnouncements).mockClear();

    const wrapper = mount(UserAnnouncementPage, {
      global: {
        stubs: componentStubs,
      },
    });

    await flushPromises();
    vi.mocked(api.getMyAnnouncements).mockClear();
    await wrapper.get('input[type="checkbox"]').setValue(true);
    await flushPromises();

    expect(api.getMyAnnouncements).toHaveBeenLastCalledWith({
      page: 1,
      page_size: 20,
      unread_only: true,
    });
    expect(api.getMyAnnouncements).toHaveBeenCalledTimes(1);
  });
});
