// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h, nextTick } from 'vue';

import NotificationBellPanel from './NotificationBellPanel.vue';

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

vi.mock('../api/notification', () => ({
  getNotificationUnreadCount: vi.fn(async () => ({ count: 0 })),
  getNotifications: vi.fn(async () => ({ items: [], total: 0, page: 1, page_size: 5 })),
  markNotificationRead: vi.fn(),
  markNotificationsReadAll: vi.fn(),
}));

vi.mock('../shared/presentation', () => ({
  notificationSeverityTheme: () => 'default',
  presentNotification: () => ({
    categoryLabel: '任务',
    compactMeta: '任务 / 定时任务 · 2026/06/11 14:11',
    levelLabel: '信息',
    message: '审计日志保留清理已成功完成。',
    occurredAtLabel: '2026/06/11 14:11',
    sourceLabel: '定时任务',
    status: 'unread',
    title: '定时任务执行成功',
  }),
}));

vi.mock('@/shared/components/management', () => ({
  formatCompactDateTime: () => '',
}));

const t = (key: string) => {
  const messages: Record<string, string> = {
    'notification.action.markAllRead': '全部标为已读',
    'notification.action.markRead': '标记已读',
    'notification.action.viewAll': '查看全部通知',
    'notification.bell.open': '打开通知',
    'notification.bell.title': '通知',
    'notification.bell.unreadSummary': '1 条未读',
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

vi.mock('tdesign-vue-next/es/message', () => ({
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
    props: {
      count: { type: Number, default: 0 },
    },
    setup(props, { slots }) {
      return () => h('div', { 'data-count': String(props.count) }, slots.default?.());
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
    expect(footer.text()).toContain('查看全部通知');

    await footer.trigger('click');
    await nextTick();

    expect(popup.attributes('data-visible')).toBe('false');
    expect(pushMock).toHaveBeenCalledWith('/notifications');
  });

  it('renders compact notification content without an inline mark-read action', async () => {
    const api = await import('../api/notification');
    vi.mocked(api.getNotifications).mockResolvedValueOnce({
      items: [
        {
          category: 'TASK',
          delivery_created_at: '2026-06-11T14:11:00Z',
          delivery_id: 1,
          event_id: 1,
          event_type: 'task_succeeded',
          message: 'Scheduled task succeeded.',
          navigation: { kind: 'SCHEDULER_RUN', payload: {} },
          occurred_at: '2026-06-11T14:11:00Z',
          severity: 'info',
          source_module: 'scheduler',
          status: 'unread',
          target_ref: '1',
          target_type: 'USER',
          title: 'Scheduled task succeeded',
        },
      ],
      total: 1,
      page: 1,
      page_size: 5,
    });
    vi.mocked(api.getNotificationUnreadCount).mockResolvedValueOnce({ count: 1 });

    const wrapper = mount(NotificationBellPanel, {
      global: {
        stubs: componentStubs,
      },
    });
    const popup = wrapper.getComponent({ name: 'TPopupStub' });

    popup.vm.$emit('update:visible', true);
    popup.vm.$emit('visible-change', true);
    await flushPromises();
    await nextTick();

    expect(wrapper.text()).toContain('定时任务执行成功');
    expect(wrapper.text()).toContain('审计日志保留清理已成功完成。');
    expect(wrapper.text()).toContain('任务 / 定时任务 · 2026/06/11 14:11');
    expect(wrapper.text()).not.toContain('标记已读');
    expect(wrapper.find('.notification-bell-panel__item-main').exists()).toBe(true);
    expect(wrapper.find('.notification-bell-panel__unread-dot').exists()).toBe(true);
  });

  it('keeps the bell entry inside the standard header entry layout without badge-owned outer sizing', () => {
    const wrapper = mount(NotificationBellPanel, {
      global: {
        stubs: componentStubs,
      },
    });

    expect(wrapper.classes()).toContain('notification-header-entry');
    expect(wrapper.find('.notification-header-entry').exists()).toBe(true);
    expect(wrapper.find('[data-count]').exists()).toBe(true);
  });
});
