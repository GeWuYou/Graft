// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, nextTick } from 'vue';

import type { AnnouncementItem } from '../../types/announcement';
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
    warning: vi.fn(),
  },
}));

vi.mock('@/shared/components/markdown', () => ({
  MarkdownViewer: defineComponent({
    props: {
      source: { type: String, default: '' },
    },
    setup(props) {
      return () => h('article', { 'data-testid': 'markdown-viewer' }, props.source);
    },
  }),
  markdownToPlainTextSummary: (source: string) => source,
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
    'announcement.management.columnSettings': 'Column Settings',
    'announcement.management.columns.archivedAt': 'Archived At',
    'announcement.management.columns.createdAt': 'Created At',
    'announcement.management.columns.expireAt': 'Expire At',
    'announcement.management.columns.level': 'Level',
    'announcement.management.columns.deliveryMode': 'Delivery',
    'announcement.management.columns.operation': 'Actions',
    'announcement.management.columns.pinned': 'Pinned',
    'announcement.management.columns.publishedAt': 'Published At',
    'announcement.management.columns.publishedBy': 'Published By',
    'announcement.management.columns.publishAt': 'Effective At',
    'announcement.management.columns.status': 'Status',
    'announcement.management.columns.title': 'Announcement',
    'announcement.management.columns.updatedAt': 'Updated At',
    'announcement.management.columns.visibility': 'Visibility',
    'announcement.management.create': 'Create Announcement',
    'announcement.management.createSuccess': 'Announcement Created',
    'announcement.management.delete': 'Delete',
    'announcement.management.deleteDialog.cancel': 'Cancel',
    'announcement.management.deleteDialog.confirm': 'Delete',
    'announcement.management.deleteDialog.description': 'Delete confirmation description',
    'announcement.management.deleteDialog.title': 'Delete Announcement',
    'announcement.management.detail': 'Details',
    'announcement.management.description': 'Manage announcements.',
    'announcement.management.deletePublishedHint': 'Archive before deleting',
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
    'announcement.management.form.expireAt': 'Expires At',
    'announcement.management.form.expireAtHelp': 'Expiration help',
    'announcement.management.form.expireAtPlaceholder': 'Select expire time',
    'announcement.management.form.deliveryMode': 'Delivery Mode',
    'announcement.management.form.deliveryModeHelp.popup': 'Popup help',
    'announcement.management.form.deliveryModeHelp.silent': 'Silent help',
    'announcement.management.form.deliveryModePlaceholder': 'Select delivery mode',
    'announcement.management.form.immediateEffective': 'Immediate Effective',
    'announcement.management.form.invalidTimeWindow':
      'Expiration time must be later than effective time; when effective time is empty, expiration time must be later than now',
    'announcement.management.form.level': 'Level',
    'announcement.management.form.levelPlaceholder': 'Select announcement level',
    'announcement.management.form.pinned': 'Pinned Announcement',
    'announcement.management.form.publishAt': 'Effective At',
    'announcement.management.form.publishAtHelp': 'Effective help',
    'announcement.management.form.publishAtPlaceholder': 'Select publish time',
    'announcement.management.form.required.content': 'Content is required',
    'announcement.management.form.required.deliveryMode': 'Delivery mode is required',
    'announcement.management.form.required.level': 'Level is required',
    'announcement.management.form.required.title': 'Title is required',
    'announcement.management.form.setCurrentTime': 'Use Current Time',
    'announcement.management.form.title': 'Title',
    'announcement.management.form.titlePlaceholder': 'Enter announcement title',
    'announcement.management.form.visibility': 'Visibility Window',
    'announcement.management.form.markdownPreview': 'Markdown Preview',
    'announcement.management.form.previewCurrent': 'Preview Current Content',
    'announcement.management.form.collapsePreview': 'Collapse Preview',
    'announcement.management.form.openFullPreview': 'Open Full Preview',
    'announcement.management.form.emptyPreview': 'No Preview Content',
    'announcement.management.form.closePreview': 'Close',
    'announcement.management.form.untitledPreview': 'Untitled Announcement',
    'announcement.management.more': 'More',
    'announcement.management.publishNow': 'Publish Now',
    'announcement.management.publishSuccess': 'Announcement Published',
    'announcement.management.republish': 'Republish',
    'announcement.management.refresh': 'Refresh',
    'announcement.management.reset': 'Clear Filters',
    'announcement.management.resetColumns': 'Restore Default Columns',
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
    'announcement.value.immediateEffective': 'After Publish',
    'announcement.value.longTerm': 'Long-Term',
    'announcement.visibility.archived': 'Archived',
    'announcement.visibility.draft': 'Draft',
    'announcement.visibility.expired': 'Expired',
    'announcement.visibility.scheduled': 'Not Started',
    'announcement.visibility.visible': 'Visible',
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
  props: ['columns', 'data'],
  setup(props, { slots }) {
    return () =>
      h('div', { 'data-testid': 'announcement-table' }, [
        h(
          'div',
          { 'data-testid': 'announcement-columns' },
          (props.columns ?? []).map((column: { colKey: string; title: string }) =>
            h('span', { 'data-col-key': column.colKey }, column.title),
          ),
        ),
        props.data.map((row: Record<string, unknown>) =>
          h(
            'section',
            { 'data-testid': 'announcement-row' },
            (props.columns ?? []).map((column: { colKey: string }) =>
              h(
                'div',
                { 'data-cell-key': column.colKey },
                slots[column.colKey]?.({ row }) ?? String(row[column.colKey] ?? ''),
              ),
            ),
          ),
        ),
      ]);
  },
});

const TDrawerStub = defineComponent({
  name: 'TDrawerStub',
  props: ['visible'],
  setup(props, { slots }) {
    return () =>
      props.visible ? h('section', { 'data-testid': 'drawer' }, [slots.default?.(), slots.footer?.()]) : null;
  },
});

const TFormStub = defineComponent({
  name: 'TFormStub',
  emits: ['submit'],
  setup(_props, { emit, expose, slots }) {
    const submit = () => emit('submit', { validateResult: true });
    expose({ submit });
    return () =>
      h(
        'form',
        {
          onSubmit: (event: Event) => {
            event.preventDefault();
            submit();
          },
        },
        slots.default?.(),
      );
  },
});

type TestAnnouncement = AnnouncementItem;

function announcement(overrides: Partial<TestAnnouncement> = {}) {
  return {
    ...baseAnnouncement(),
    ...overrides,
  };
}

function baseAnnouncement(): TestAnnouncement {
  return {
    content: 'Body',
    created_at: '2026-06-12T00:00:00Z',
    delivery_mode: 'silent' as const,
    expire_at: null,
    id: 1,
    level: 'info' as const,
    pinned: false,
    published_at: null,
    published_by: null,
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
          setup(props, { emit }) {
            return () =>
              h(
                'div',
                (props.actions as Array<{ disabled?: boolean; label: string; value: string }>).map((action) =>
                  h(
                    'button',
                    {
                      disabled: Boolean(action.disabled),
                      'data-testid': `${action.value}-action`,
                      onClick: () => emit('action', action.value),
                    },
                    translations[action.label] ?? action.label,
                  ),
                ),
              );
          },
        }),
        't-button': TButtonStub,
        't-checkbox': defineComponent({
          name: 'TCheckboxStub',
          setup(_props, { slots }) {
            return () => h('label', slots.default?.());
          },
        }),
        't-checkbox-group': PassthroughStub,
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
        't-dialog': defineComponent({
          name: 'TDialogStub',
          props: ['cancelBtn', 'confirmBtn', 'header', 'visible'],
          emits: ['close', 'confirm', 'update:visible'],
          setup(props, { emit, slots }) {
            return () =>
              props.visible
                ? h(
                    'aside',
                    {
                      'data-testid':
                        props.header === 'Delete Announcement' ? 'delete-confirm-dialog' : 'full-preview-dialog',
                    },
                    [
                      props.header ? h('h2', String(props.header)) : null,
                      slots.default?.(),
                      props.cancelBtn
                        ? h(
                            'button',
                            {
                              'data-testid': 'dialog-cancel',
                              onClick: () => {
                                emit('close');
                                emit('update:visible', false);
                              },
                            },
                            String(props.cancelBtn),
                          )
                        : null,
                      props.confirmBtn
                        ? h(
                            'button',
                            {
                              'data-testid': 'dialog-confirm',
                              disabled: Boolean(props.confirmBtn.disabled),
                              onClick: () => emit('confirm'),
                            },
                            String(props.confirmBtn.content),
                          )
                        : null,
                    ],
                  )
                : null;
          },
        }),
        't-empty': defineComponent({
          name: 'TEmptyStub',
          props: ['description'],
          setup(props, { slots }) {
            return () =>
              h('div', { 'data-testid': 'empty-state' }, slots.default?.() ?? String(props.description ?? ''));
          },
        }),
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
        't-space': PassthroughStub,
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
    vi.useRealTimers();
    vi.clearAllMocks();
    dispatchSpy.mockClear();
    window.localStorage.clear();
    apiMocks.getAnnouncements.mockResolvedValue({
      items: [announcement()],
      page: 1,
      page_size: 20,
      total: 1,
    });
    apiMocks.getAnnouncement.mockResolvedValue(announcement({ id: 1 }));
    apiMocks.publishAnnouncement.mockResolvedValue(announcement({ status: 'published' }));
    apiMocks.createAnnouncement.mockResolvedValue(announcement({ id: 2 }));
    apiMocks.deleteAnnouncement.mockResolvedValue({});
  });

  it('renders status-specific row actions', async () => {
    apiMocks.getAnnouncements.mockResolvedValue({
      items: [
        announcement({ id: 1, status: 'draft' }),
        announcement({ id: 2, status: 'published' }),
        announcement({ id: 3, status: 'archived' }),
      ],
      page: 1,
      page_size: 20,
      total: 3,
    });

    const wrapper = mountPage();
    await flushPromises();

    const rows = wrapper.findAll('[data-testid="announcement-row"]');
    expect(rows[0].text()).toContain('Draft');
    expect(rows[0].text()).toContain('Details');
    expect(rows[0].text()).toContain('Edit');
    expect(rows[0].text()).toContain('Publish Now');
    expect(rows[0].text()).toContain('Delete');
    expect(rows[0].text()).not.toContain('Archive');
    expect(rows[0].text()).not.toContain('Republish');

    expect(rows[1].text()).toContain('Details');
    expect(rows[1].text()).toContain('Visible');
    expect(rows[1].text()).toContain('Edit');
    expect(rows[1].text()).toContain('Archive');
    expect(rows[1].text()).not.toContain('Delete');
    expect(rows[1].text()).not.toContain('Publish Now');
    expect(rows[1].text()).not.toContain('Republish');

    expect(rows[2].text()).toContain('Details');
    expect(rows[2].text()).toContain('Archived');
    expect(rows[2].text()).toContain('Edit');
    expect(rows[2].text()).toContain('Republish');
    expect(rows[2].text()).toContain('Delete');
    expect(rows[2].find('[data-testid="archive-action"]').exists()).toBe(false);
    expect(rows[2].find('[data-testid="edit-action"]').attributes('disabled')).toBeDefined();
  });

  it('defaults the management table to published, effective, and expiration time columns', async () => {
    apiMocks.getAnnouncements.mockResolvedValue({
      items: [
        announcement({
          expire_at: null,
          published_at: '2026-06-12T00:15:00Z',
          publish_at: null,
          status: 'published',
        }),
      ],
      page: 1,
      page_size: 20,
      total: 1,
    });

    const wrapper = mountPage();
    await flushPromises();

    const columnKeys = wrapper
      .findAll('[data-testid="announcement-columns"] [data-col-key]')
      .map((column) => column.attributes('data-col-key'));

    expect(columnKeys).toEqual([
      'title',
      'status',
      'visibility',
      'level',
      'published_at',
      'publish_at',
      'expire_at',
      'operation',
    ]);
    expect(columnKeys).not.toContain('updated_at');
    expect(columnKeys).not.toContain('delivery_mode');
    expect(wrapper.get('[data-cell-key="publish_at"]').text()).toBe('After Publish');
    expect(wrapper.get('[data-cell-key="expire_at"]').text()).toBe('Long-Term');
    expect(wrapper.get('[data-cell-key="published_at"]').text()).not.toBe('Not Set');
  });

  it('persists optional column settings while keeping title and operation visible', async () => {
    window.localStorage.setItem(
      'graft.announcement.management.visibleColumns',
      JSON.stringify(['published_at', 'publish_at', 'expire_at', 'updated_at']),
    );

    const wrapper = mountPage();
    await flushPromises();

    const columnKeys = wrapper
      .findAll('[data-testid="announcement-columns"] [data-col-key]')
      .map((column) => column.attributes('data-col-key'));

    expect(columnKeys).toEqual(['title', 'published_at', 'publish_at', 'expire_at', 'updated_at', 'operation']);

    const storedKeys = JSON.parse(window.localStorage.getItem('graft.announcement.management.visibleColumns') ?? '[]');
    expect(storedKeys).toEqual(['title', 'published_at', 'publish_at', 'expire_at', 'updated_at', 'operation']);
  });

  it('refreshes the list after a publish action succeeds', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('[data-testid="publish-action"]').trigger('click');
    await flushPromises();

    expect(apiMocks.publishAnnouncement).toHaveBeenCalledWith(1);
    expect(apiMocks.getAnnouncements).toHaveBeenCalledTimes(2);
  });

  it('republishes an archived announcement without forcing an effective time', async () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-06-13T08:09:10.000Z'));
    apiMocks.getAnnouncements.mockResolvedValue({
      items: [announcement({ status: 'archived' })],
      page: 1,
      page_size: 20,
      total: 1,
    });

    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('[data-testid="publish-action"]').trigger('click');
    await flushPromises();

    expect(wrapper.text()).toContain('Republish');
    expect(apiMocks.publishAnnouncement).toHaveBeenCalledWith(1);

    vi.useRealTimers();
  });

  it('opens a TDesign confirmation dialog before deleting a draft announcement', async () => {
    const nativeConfirmSpy = vi.spyOn(window, 'confirm');
    const wrapper = mountPage();
    await flushPromises();
    apiMocks.getAnnouncements.mockClear();

    await wrapper.get('[data-testid="delete-action"]').trigger('click');
    await nextTick();

    expect(nativeConfirmSpy).not.toHaveBeenCalled();
    expect(apiMocks.deleteAnnouncement).not.toHaveBeenCalled();
    expect(wrapper.get('[data-testid="delete-confirm-dialog"]').text()).toContain('Maintenance');

    await wrapper.get('[data-testid="dialog-confirm"]').trigger('click');
    await flushPromises();

    expect(apiMocks.deleteAnnouncement).toHaveBeenCalledWith(1);
    expect(apiMocks.getAnnouncements).toHaveBeenCalledTimes(1);
    expect(window.dispatchEvent).toHaveBeenCalledWith(expect.objectContaining({ type: 'graft:announcement-changed' }));
  });

  it('archives a published announcement instead of exposing delete', async () => {
    apiMocks.getAnnouncements.mockResolvedValue({
      items: [announcement({ status: 'published' })],
      page: 1,
      page_size: 20,
      total: 1,
    });
    apiMocks.archiveAnnouncement.mockResolvedValue(announcement({ status: 'archived' }));
    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.find('[data-testid="delete-action"]').exists()).toBe(false);

    await wrapper.get('[data-testid="archive-action"]').trigger('click');
    await flushPromises();

    expect(apiMocks.archiveAnnouncement).toHaveBeenCalledWith(1);
    expect(apiMocks.deleteAnnouncement).not.toHaveBeenCalled();
    expect(apiMocks.getAnnouncements).toHaveBeenCalledTimes(2);
  });

  it('searches through a single fetch chain when the current page resets', async () => {
    const wrapper = mountPage();
    await flushPromises();
    apiMocks.getAnnouncements.mockClear();

    await wrapper
      .findAll('button')
      .find((button) => button.text() === 'Search')!
      .trigger('click');
    await flushPromises();

    expect(apiMocks.getAnnouncements).toHaveBeenCalledTimes(1);
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
    expect(MessagePlugin.error).toHaveBeenCalledWith(
      'Expiration time must be later than effective time; when effective time is empty, expiration time must be later than now',
    );
  });

  it('blocks create submission when only expire time is in the past', async () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date(2026, 5, 13, 16, 17, 18));
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
    await wrapper.findAll('input').at(-1)?.setValue('2026-06-13 15:17:18');
    await wrapper.get('form').trigger('submit');
    await flushPromises();

    expect(apiMocks.createAnnouncement).not.toHaveBeenCalled();
    const { MessagePlugin } = await import('tdesign-vue-next/es/message');
    expect(MessagePlugin.error).toHaveBeenCalledWith(
      'Expiration time must be later than effective time; when effective time is empty, expiration time must be later than now',
    );

    vi.useRealTimers();
  });

  it('keeps DatePicker state as local display strings and converts to UTC only on submit', async () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date(2026, 5, 13, 16, 17, 18));
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
    const publishInput = inputs.at(-2)!;
    const expireInput = inputs.at(-1)!;
    expect(publishInput.attributes('data-default-time')).toBeUndefined();

    await publishInput.setValue('2026-06-13 16:17:18');
    await expireInput.setValue('2026-06-13 17:17:18');
    expect(publishInput.element.value).toBe('2026-06-13 16:17:18');
    await wrapper.get('form').trigger('submit');
    await flushPromises();

    expect(apiMocks.createAnnouncement).toHaveBeenCalledWith(
      expect.objectContaining({
        expire_at: new Date(2026, 5, 13, 17, 17, 18).toISOString(),
        publish_at: new Date(2026, 5, 13, 16, 17, 18).toISOString(),
      }),
    );

    vi.useRealTimers();
  });

  it('sets the effective time explicitly or clears it for immediate visibility', async () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date(2026, 5, 13, 16, 17, 18));
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('[data-testid="announcement-create"]').trigger('click');
    await nextTick();
    const publishInput = wrapper.findAll('input').at(-2)!;

    await wrapper
      .findAll('button')
      .find((button) => button.text() === 'Use Current Time')!
      .trigger('click');
    await nextTick();
    expect(publishInput.element.value).toBe('2026-06-13 16:17:18');

    await wrapper
      .findAll('button')
      .find((button) => button.text() === 'Immediate Effective')!
      .trigger('click');
    await nextTick();
    expect(publishInput.element.value).toBe('');

    vi.useRealTimers();
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

  it('keeps markdown preview collapsed by default and toggles inline preview on demand', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('[data-testid="announcement-create"]').trigger('click');
    await nextTick();

    expect(wrapper.find('.announcement-form__inline-preview').exists()).toBe(false);
    expect(wrapper.find('[data-testid="markdown-viewer"]').exists()).toBe(false);

    await wrapper.get('textarea').setValue('## Preview Body');
    await wrapper
      .findAll('button')
      .find((button) => button.text() === 'Preview Current Content')!
      .trigger('click');
    await nextTick();

    expect(wrapper.find('.announcement-form__inline-preview').exists()).toBe(true);
    expect(wrapper.get('[data-testid="markdown-viewer"]').text()).toBe('## Preview Body');

    await wrapper
      .findAll('button')
      .find((button) => button.text() === 'Collapse Preview')!
      .trigger('click');
    await nextTick();

    expect(wrapper.find('.announcement-form__inline-preview').exists()).toBe(false);
  });

  it('shows an empty inline preview state without rendering markdown content', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('[data-testid="announcement-create"]').trigger('click');
    await nextTick();
    await wrapper
      .findAll('button')
      .find((button) => button.text() === 'Preview Current Content')!
      .trigger('click');
    await nextTick();

    expect(wrapper.text()).toContain('No Preview Content');
    expect(wrapper.find('[data-testid="markdown-viewer"]').exists()).toBe(false);
  });

  it('opens full markdown preview without submitting the form', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('[data-testid="announcement-create"]').trigger('click');
    await nextTick();
    const titleInput = wrapper
      .findAll('input')
      .find(
        (input) => input.attributes('value') === '' && input.attributes('placeholder') !== 'Search title or content',
      );
    await titleInput!.setValue('Preview Title');
    await wrapper.get('textarea').setValue('Full **body**');
    await wrapper
      .findAll('button')
      .find((button) => button.text() === 'Open Full Preview')!
      .trigger('click');
    await nextTick();

    expect(wrapper.get('[data-testid="full-preview-dialog"]').text()).toContain('Preview Title');
    expect(wrapper.get('[data-testid="full-preview-dialog"]').text()).toContain('Info');
    expect(wrapper.get('[data-testid="full-preview-dialog"]').text()).toContain('Silent');
    expect(wrapper.get('[data-testid="markdown-viewer"]').text()).toBe('Full **body**');
    expect(apiMocks.createAnnouncement).not.toHaveBeenCalled();
  });

  it('keeps delivery mode help in a tooltip instead of inline form copy', async () => {
    const wrapper = mountPage();
    await flushPromises();

    await wrapper.get('[data-testid="announcement-create"]').trigger('click');
    await nextTick();

    expect(wrapper.find('.announcement-form__field-help').exists()).toBe(true);
    expect(wrapper.find('.announcement-form__help-icon').exists()).toBe(true);
    expect(wrapper.find('[title="Silent help"]').exists()).toBe(true);
  });
});
