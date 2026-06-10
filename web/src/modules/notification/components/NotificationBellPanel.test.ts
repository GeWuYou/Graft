// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h, nextTick } from 'vue';

import NotificationBellPanel from './NotificationBellPanel.vue';

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

vi.mock('../api/notification', () => ({
  getNotificationUnreadCount: vi.fn(async () => ({ count: 0 })),
  getNotifications: vi.fn(async () => ({ items: [], total: 0, page: 1, page_size: 5 })),
  markNotificationRead: vi.fn(),
  markNotificationsReadAll: vi.fn(),
}));

vi.mock('../shared/presentation', () => ({
  notificationMessage: () => '',
  notificationSeverityTheme: () => 'default',
  notificationSourceLabel: () => '',
  notificationTitle: () => '',
}));

vi.mock('@/shared/components/management', () => ({
  formatCompactDateTime: () => '',
}));

const t = (key: string) => {
  const messages: Record<string, string> = {
    'notification.actions.viewAll': '打开通知中心 →',
    'notification.bell.open': '打开通知',
    'notification.empty.description': '当前筛选条件下没有通知。',
    'notification.empty.title': '暂无通知',
  };
  return messages[key] ?? key;
};

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    locale: 'zh-CN',
    t,
  }),
}));

vi.mock('tdesign-vue-next', () => ({
  MessagePlugin: {
    error: vi.fn(),
    success: vi.fn(),
  },
}));

const buttonStub = defineComponent({
  setup(_, { slots }) {
    return () => h('button', { type: 'button' }, slots.default?.());
  },
});

const componentStubs = {
  't-badge': defineComponent({
    setup(_, { slots }) {
      return () => h('div', slots.default?.());
    },
  }),
  't-button': buttonStub,
  't-empty': defineComponent({
    props: {
      description: { type: String, default: '' },
      title: { type: String, default: '' },
    },
    setup(props) {
      return () => h('div', [props.title, props.description]);
    },
  }),
  't-icon': defineComponent({
    setup() {
      return () => h('i');
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
  't-popup': defineComponent({
    name: 'TPopupStub',
    props: {
      visible: { type: Boolean, default: false },
    },
    emits: ['update:visible', 'visible-change'],
    setup(props, { slots }) {
      return () => h('div', { 'data-visible': String(props.visible) }, [slots.content?.(), slots.default?.()]);
    },
  }),
  't-tag': defineComponent({
    setup(_, { slots }) {
      return () => h('span', slots.default?.());
    },
  }),
};

describe('NotificationBellPanel', () => {
  it('renders a footer navigation link and opens the notification center', async () => {
    pushMock.mockReset();
    const wrapper = mount(NotificationBellPanel, {
      global: {
        stubs: componentStubs,
      },
    });
    const popup = wrapper.getComponent({ name: 'TPopupStub' });

    popup.vm.$emit('update:visible', true);
    popup.vm.$emit('visible-change', true);
    await nextTick();
    expect(popup.attributes('data-visible')).toBe('true');

    const footer = wrapper.get('.notification-bell-panel__foot');
    expect(footer.text()).toContain('打开通知中心 →');

    await footer.trigger('click');
    await nextTick();

    expect(popup.attributes('data-visible')).toBe('false');
    expect(pushMock).toHaveBeenCalledWith('/notifications');
  });
});
