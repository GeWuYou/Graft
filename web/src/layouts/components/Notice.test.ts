import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import Notice from './Notice.vue';

vi.mock('@/modules/announcement', () => ({
  announcementHeaderEntry: defineComponent({
    name: 'AnnouncementHeaderEntryStub',
    setup() {
      return () => h('div', { class: 'announcement-header-entry', 'data-testid': 'announcement-entry' });
    },
  }),
  announcementPopupHost: defineComponent({
    name: 'AnnouncementPopupHostStub',
    setup() {
      return () => h('div', { class: 'announcement-popup-host', 'data-testid': 'announcement-popup-host' });
    },
  }),
}));

vi.mock('@/modules/notification', () => ({
  notificationHeaderWidget: defineComponent({
    name: 'NotificationHeaderWidgetStub',
    setup() {
      return () => h('div', { class: 'notification-header-entry', 'data-testid': 'notification-entry' });
    },
  }),
}));

describe('Notice', () => {
  it('wraps notification and announcement entries with standard header operation items', () => {
    const wrapper = mount(Notice);

    const operationItems = wrapper.findAll('.header-operation-item');
    expect(operationItems).toHaveLength(2);
    expect(operationItems[0]?.find('[data-testid="notification-entry"]').exists()).toBe(true);
    expect(operationItems[1]?.find('[data-testid="announcement-entry"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="announcement-popup-host"]').exists()).toBe(true);
    expect(wrapper.find('.header-notice-actions').exists()).toBe(false);
  });
});
