// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, nextTick } from 'vue';

import AnnouncementManagementPage from './index.vue';

const dispatchSpy = vi.spyOn(window, 'dispatchEvent');

const apiMocks = vi.hoisted(() => ({
  archiveAnnouncement: vi.fn(),
  createAnnouncement: vi.fn(),
  deleteAnnouncement: vi.fn(),
  getAnnouncement: vi.fn(),
  getAnnouncements: vi.fn(),
  publishAnnouncement: vi.fn(),
  updateAnnouncement: vi.fn(),
}));

vi.mock('../../api/announcement', () => apiMocks);

vi.mock('tdesign-vue-next/es/message', () => ({
  MessagePlugin: {
    error: vi.fn(),
    success: vi.fn(),
  },
}));

vi.mock('@/shared/components/markdown', () => ({
  markdownToPlainTextSummary: (source: string) => source,
  SafeMarkdown: defineComponent({
    props: {
      source: { type: String, default: '' },
    },
    setup(props) {
      return () => h('div', props.source);
    },
  }),
}));

const translations = vi.hoisted(
  (): Record<string, string> => ({
    'announcement.level.error': 'Error',
    'announcement.level.info': 'Info',
    'announcement.level.success': 'Success',
    'announcement.level.warning': 'Warning',
    'announcement.deliveryMode.popup': 'Popup',
    'announcement.deliveryMode.silent': 'Silent',
    'announcement.management.archive': 'Archive',
    'announcement.management.archiveSuccess': 'Announcement Archived',
    'announcement.management.columns.expireAt': 'Expire At',
    'announcement.management.columns.level': 'Level',
    'announcement.management.columns.deliveryMode': 'Delivery',
    'announcement.management.columns.operation': 'Actions',
    'announcement.management.columns.pinned': 'Pinned',
    'announcement.management.columns.publishAt': 'Publish At',
    'announcement.management.columns.status': 'Status',
    'announcement.management.columns.title': 'Announcement',
    'announcement.management.columns.updatedAt': 'Updated At',
    'announcement.management.create': 'Create Announcement',
    'announcement.management.createSuccess': 'Announcement Created',
    'announcement.management.delete': 'Delete',
    'announcement.management.detail': 'Details',
    'announcement.management.description': 'Manage announcements.',
    'announcement.management.edit': 'Edit',
    'announcement.management.emptyCreate': 'Create Announcement',
    'announcement.management.emptyDescription': 'No announcements match filters.',
    'announcement.management.emptyTitle': 'No Announcements',
    'announcement.management.filters.keyword': 'Search title or content',
    'announcement.management.filters.level': 'Filter Level',
    'announcement.management.filters.pinned': 'Filter Pinned',
    'announcement.management.filters.sort': 'Sort',
    'announcement.management.filters.status': 'Filter Status',
    'announcement.management.footerTotal': '{count} Announcements Total',
    'announcement.management.form.basicInfo': 'Announcement Content',
    'announcement.management.form.cancel': 'Cancel',
    'announcement.management.form.confirm': 'Save Announcement',
    'announcement.management.form.content': 'Content',
    'announcement.management.form.contentPlaceholder': 'Enter announcement content',
    'announcement.management.form.createTitle': 'Create Announcement',
    'announcement.management.form.expireAt': 'Expire At',
    'announcement.management.form.expireAtPlaceholder': 'Select expire time',
    'announcement.management.form.deliveryMode': 'Delivery Mode',
    'announcement.management.form.deliveryModeHelp.popup': 'Popup help',
    'announcement.management.form.deliveryModeHelp.silent': 'Silent help',
    'announcement.management.form.deliveryModePlaceholder': 'Select delivery mode',
    'announcement.management.form.invalidTimeWindow': 'Expire time must be later than publish time',
    'announcement.management.form.level': 'Level',
    'announcement.management.form.levelPlaceholder': 'Select announcement level',
    'announcement.management.form.pinned': 'Pinned Announcement',
    'announcement.management.form.publishAt': 'Publish At',
    'announcement.management.form.publishAtPlaceholder': 'Select publish time',
    'announcement.management.form.required.content': 'Content is required',
    'announcement.management.form.required.deliveryMode': 'Delivery mode is required',
    'announcement.management.form.required.level': 'Level is required',
    'announcement.management.form.required.title': 'Title is required',
    'announcement.management.form.title': 'Title',
    'announcement.management.form.titlePlaceholder': 'Enter announcement title',
    'announcement.management.form.visibility': 'Visibility Window',
    'announcement.management.form.markdownPreview': 'Markdown Preview',
    'announcement.management.more': 'More',
    'announcement.management.publishNow': 'Publish Now',
    'announcement.management.publishSuccess': 'Announcement Published',
    'announcement.management.refresh': 'Refresh',
    'announcement.management.reset': 'Clear Filters',
    'announcement.management.search': 'Search',
    'announcement.management.sort.pinnedPublishDesc': 'Pinned First',
    'announcement.management.sort.publishDesc': 'Recently Published',
    'announcement.management.sort.updatedDesc': 'Recently Updated',
    'announcement.management.summary': '{count} Announcements',
    'announcement.management.tableHint': 'Filter announcements.',
    'announcement.management.title': 'Announcement Management',
    'announcement.pinned.no': 'Normal',
    'announcement.pinned.yes': 'Pinned',
    'announcement.status.archived': 'Archived',
    'announcement.status.draft': 'Draft',
    'announcement.status.published': 'Published',
    'announcement.value.notSet': 'Not Set',
    'components.commonTable.more': 'More',
    'menu.server.title': 'Server',
  }),
);

vi.mock('vue-i18n', () => ({
  createI18n: () => ({
    global: {
      getLocaleMessage: () => ({}),
      locale: { value: 'en-US' },
      t: (key: string) => translations[key] ?? key,
    },
  }),
  useI18n: () => ({
    locale: { value: 'en-US' },
    t: (key: string, params?: Record<string, unknown>) =>
      (translations[key] ?? key).replace(/\{(\w+)\}/g, (_, name) => String(params?.[name] ?? `{${name}}`)),
  }),
}));

const PassthroughStub = defineComponent({
  name: 'PassthroughStub',
  inheritAttrs: false,
  setup(_props, { slots }) {
    return () =>
      h(
        'div',
        Object.entries(slots).flatMap(([name, slot]) => (name === 'default' ? (slot?.() ?? []) : (slot?.({}) ?? []))),
      );
  },
});

const TButtonStub = defineComponent({
  name: 'TButtonStub',
  props: ['loading', 'theme', 'variant'],
  emits: ['click'],
  setup(_props, { emit, slots }) {
    return () => h('button', { onClick: () => emit('click') }, slots.default?.());
  },
});

const TTableStub = defineComponent({
  name: 'TTableStub',
  props: ['data'],
  setup(props, { slots }) {
    return () =>
      h(
        'div',
        { 'data-testid': 'announcement-table' },
        props.data.flatMap((row: Record<string, unknown>) => [
          slots.title?.({ row }),
          slots.delivery_mode?.({ row }),
          slots.operation?.({ row }),
        ]),
      );
  },
});

const TDrawerStub = defineComponent({
  name: 'TDrawerStub',
  props: ['visible'],
  setup(props, { slots }) {
    return () => (props.visible ? h('section', { 'data-testid': 'drawer' }, slots.default?.()) : null);
  },
});

const TFormStub = defineComponent({
  name: 'TFormStub',
  emits: ['submit'],
  setup(_props, { emit, slots }) {
    return () =>
      h(
        'form',
        {
          onSubmit: (event: Event) => {
            event.preventDefault();
            emit('submit', { validateResult: true });
          },
        },
        slots.default?.(),
      );
  },
});

type TestAnnouncement = ReturnType<typeof baseAnnouncement>;

function announcement(overrides: Partial<TestAnnouncement> = {}) {
  return {
    ...baseAnnouncement(),
    ...overrides,
  };
}

function baseAnnouncement() {
  return {
    content: 'Body',
    created_at: '2026-06-12T00:00:00Z',
    delivery_mode: 'silent' as const,
    expire_at: null,
    id: 1,
    level: 'info' as const,
    pinned: false,
    publish_at: null,
    status: 'draft' as 'draft' | 'published' | 'archived',
    title: 'Maintenance',
    updated_at: '2026-06-12T00:30:00Z',
  };
}

function mountPage() {
  return mount(AnnouncementManagementPage, {
    global: {
      directives: {
        permission: () => undefined,
      },
      stubs: {
        ManagementEmptyState: PassthroughStub,
        ManagementPageContent: PassthroughStub,
        ManagementPageHeader: PassthroughStub,
        ManagementTableCard: PassthroughStub,
        ManagementTablePagination: PassthroughStub,
        ManagementToolbar: PassthroughStub,
        TableActionMenu: defineComponent({
          name: 'TableActionMenuStub',
          props: ['actions'],
          emits: ['action'],
          setup(_props, { emit }) {
            return () =>
              h('div', [
                h('button', { 'data-testid': 'publish-action', onClick: () => emit('action', 'publish') }, 'publish'),
                h('button', { 'data-testid': 'detail-action', onClick: () => emit('action', 'detail') }, 'detail'),
              ]);
          },
        }),
        't-button': TButtonStub,
        't-checkbox': defineComponent({
          name: 'TCheckboxStub',
          setup(_props, { slots }) {
            return () => h('label', slots.default?.());
          },
        }),
        't-date-picker': defineComponent({
          name: 'TDatePickerStub',
          props: ['modelValue'],
          emits: ['update:modelValue'],
          setup(props, { emit }) {
            return () =>
              h('input', {
                value: props.modelValue,
                onInput: (event: Event) => emit('update:modelValue', (event.target as HTMLInputElement).value),
              });
          },
        }),
        't-drawer': TDrawerStub,
        't-empty': PassthroughStub,
        't-form': TFormStub,
        't-form-item': PassthroughStub,
        't-icon': defineComponent({
          name: 'TIconStub',
          setup() {
            return () => h('i');
          },
        }),
        't-input': defineComponent({
          name: 'TInputStub',
          props: ['modelValue'],
          emits: ['update:modelValue', 'enter'],
          setup(props, { attrs, emit }) {
            return () =>
              h('input', {
                ...attrs,
                value: props.modelValue,
                onInput: (event: Event) => emit('update:modelValue', (event.target as HTMLInputElement).value),
                onKeydown: (event: KeyboardEvent) => {
                  if (event.key === 'Enter') emit('enter');
                },
              });
          },
        }),
        't-pagination': PassthroughStub,
        't-select': defineComponent({
          name: 'TSelectStub',
          inheritAttrs: false,
          props: ['modelValue'],
          emits: ['update:modelValue'],
          setup(props, { emit }) {
            return () =>
              h('select', {
                value: props.modelValue,
                onChange: (event: Event) => emit('update:modelValue', (event.target as HTMLSelectElement).value),
              });
          },
        }),
        't-table': TTableStub,
        't-tag': PassthroughStub,
        't-textarea': defineComponent({
          name: 'TTextareaStub',
          props: ['modelValue'],
          emits: ['update:modelValue'],
          setup(props, { emit }) {
            return () =>
              h('textarea', {
                value: props.modelValue,
                onInput: (event: Event) => emit('update:modelValue', (event.target as HTMLTextAreaElement).value),
              });
          },
        }),
        't-tooltip': defineComponent({
          name: 'TTooltipStub',
          props: ['content'],
          setup(props, { slots }) {
            return () => h('span', { title: String(props.content ?? '') }, slots.default?.());
          },
        }),
      },
    },
  });
}

describe('announcement management page', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    dispatchSpy.mockClear();
    apiMocks.getAnnouncements.mockResolvedValue({
      items: [announcement()],
      page: 1,
      page_size: 20,
      total: 1,
    });
    apiMocks.getAnnouncement.mockResolvedValue(announcement({ id: 1 }));
    apiMocks.publishAnnouncement.mockResolvedValue(announcement({ status: 'published' }));
    apiMocks.createAnnouncement.mockResolvedValue(announcement({ id: 2 }));
    vi.spyOn(window, 'confirm').mockReturnValue(true);
  });

  it('refreshes the list after a publish action succeeds', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('[data-testid="publish-action"]').trigger('click');
    await flushPromises();

    expect(apiMocks.publishAnnouncement).toHaveBeenCalledWith(1);
    expect(apiMocks.getAnnouncements).toHaveBeenCalledTimes(2);
  });

  it('blocks create submission when the expire time is before the publish time', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('[data-testid="announcement-create"]').trigger('click');
    await nextTick();
    const titleInput = wrapper
      .findAll('input')
      .find(
        (input) => input.attributes('value') === '' && input.attributes('placeholder') !== 'Search title or content',
      );
    expect(titleInput).toBeTruthy();
    await titleInput!.setValue('Title');
    await wrapper.get('textarea').setValue('Body');
    const inputs = wrapper.findAll('input');
    await inputs.at(-2)?.setValue('2026-06-12 10:00:00');
    await inputs.at(-1)?.setValue('2026-06-12 09:00:00');
    await wrapper.get('form').trigger('submit');
    await flushPromises();

    expect(apiMocks.createAnnouncement).not.toHaveBeenCalled();
    const { MessagePlugin } = await import('tdesign-vue-next/es/message');
    expect(MessagePlugin.error).toHaveBeenCalledWith('Expire time must be later than publish time');
  });

  it('submits create payload with the selected delivery mode', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('[data-testid="announcement-create"]').trigger('click');
    await nextTick();
    const titleInput = wrapper
      .findAll('input')
      .find(
        (input) => input.attributes('value') === '' && input.attributes('placeholder') !== 'Search title or content',
      );
    expect(titleInput).toBeTruthy();
    await titleInput!.setValue('Title');
    await wrapper.get('textarea').setValue('**Body**');
    await wrapper.get('form').trigger('submit');
    await flushPromises();

    expect(apiMocks.createAnnouncement).toHaveBeenCalledWith(
      expect.objectContaining({
        content: '**Body**',
        delivery_mode: 'silent',
        level: 'info',
        title: 'Title',
      }),
    );
    expect(window.dispatchEvent).toHaveBeenCalledWith(expect.objectContaining({ type: 'graft:announcement-changed' }));
  });

  it('keeps delivery mode help in a tooltip instead of inline form copy', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('[data-testid="announcement-create"]').trigger('click');
    await nextTick();

    expect(wrapper.find('.announcement-form__field-help').exists()).toBe(false);
    expect(wrapper.find('.announcement-form__help-icon').exists()).toBe(true);
    expect(wrapper.find('[title="Silent help"]').exists()).toBe(true);
  });
});
