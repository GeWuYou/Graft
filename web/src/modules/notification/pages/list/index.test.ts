// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, nextTick, reactive } from 'vue';

import type { NotificationItem } from '../../types/notification';
import NotificationListIndex from './index.vue';

const mocks = vi.hoisted(() => ({
  deleteNotification: vi.fn(),
  getNotifications: vi.fn(),
  markNotificationRead: vi.fn(),
  markNotificationsReadAll: vi.fn(),
  requestNotificationHeaderRefresh: vi.fn(),
  routerPush: vi.fn(),
  routerReplace: vi.fn(),
}));

const routeState = reactive<{ query: Record<string, unknown> }>({
  query: {},
});

vi.mock('vue-router', async () => {
  const actual = await vi.importActual<typeof import('vue-router')>('vue-router');
  return {
    ...actual,
    useRoute: () => routeState,
    useRouter: () => ({
      push: mocks.routerPush,
      replace: mocks.routerReplace,
    }),
  };
});

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}));

vi.mock('tdesign-vue-next', async () => {
  const actual = await vi.importActual<typeof import('tdesign-vue-next')>('tdesign-vue-next');
  return {
    ...actual,
    MessagePlugin: {
      error: vi.fn(),
      success: vi.fn(),
      warning: vi.fn(),
    },
  };
});

vi.mock('@/modules/shared/localized-api-error', () => ({
  resolveLocalizedErrorMessage: (_t: unknown, _error: unknown, fallback: string) => fallback,
}));

vi.mock('@/shared/components/query-list', () => ({
  AdvancedQueryListPage: defineComponent({
    name: 'AdvancedQueryListPage',
    setup(_, { slots }) {
      return () => h('section', [slots.actions?.(), slots.filters?.(), slots.table?.(), slots.detail?.()]);
    },
  }),
}));

vi.mock('@/shared/observability', () => ({
  localDateTimeToUtcIso: (value: string) => value,
  normalizePageStateRangeForRoute: (value: string[]) => value,
  normalizeRouteRangeForPageState: (value: unknown[]) => value.filter(Boolean),
}));

vi.mock('@/utils/logger', () => ({
  createLogger: () => ({
    error: vi.fn(),
  }),
}));

vi.mock('../../api/notification', () => ({
  deleteNotification: mocks.deleteNotification,
  getNotifications: mocks.getNotifications,
  markNotificationRead: mocks.markNotificationRead,
  markNotificationsReadAll: mocks.markNotificationsReadAll,
}));

vi.mock('../../components/NotificationDetailDrawer.vue', () => ({
  default: defineComponent({
    name: 'NotificationDetailDrawer',
    props: {
      item: { type: Object, default: null },
      visible: { type: Boolean, default: false },
    },
    emits: ['mark-read', 'navigate', 'update:visible'],
    setup(props, { emit }) {
      return () =>
        h('aside', { 'data-testid': 'detail-drawer', 'data-visible': String(props.visible) }, [
          props.item
            ? h(
                'button',
                {
                  'data-testid': 'detail-mark-read',
                  onClick: () => emit('mark-read', props.item),
                },
                'mark read',
              )
            : null,
          h(
            'button',
            {
              'data-testid': 'detail-close',
              onClick: () => emit('update:visible', false),
            },
            'close',
          ),
        ]);
    },
  }),
}));

vi.mock('../../components/NotificationFilters.vue', () => ({
  default: defineComponent({
    name: 'NotificationFilters',
    setup() {
      return () => h('div');
    },
  }),
}));

vi.mock('../../components/NotificationTable.vue', () => ({
  default: defineComponent({
    name: 'NotificationTable',
    props: {
      items: { type: Array, default: () => [] },
    },
    emits: ['delete', 'detail', 'page-change'],
    setup(props, { emit }) {
      return () =>
        h(
          'div',
          (props.items as NotificationItem[]).map((item) =>
            h(
              'button',
              {
                'data-testid': `detail-${item.delivery_id}`,
                onClick: () => emit('detail', item),
              },
              item.title,
            ),
          ),
        );
    },
  }),
}));

vi.mock('../../contract/refresh', () => ({
  requestNotificationHeaderRefresh: mocks.requestNotificationHeaderRefresh,
}));

vi.mock('../../shared/presentation', async () => {
  const actual = await vi.importActual<typeof import('../../shared/presentation')>('../../shared/presentation');
  return {
    ...actual,
    NOTIFICATION_MVP_SOURCE_MODULES: ['scheduler'],
  };
});

function notification(overrides: Partial<NotificationItem> = {}): NotificationItem {
  return {
    category: 'TASK',
    delivery_created_at: '2026-06-11T10:47:21Z',
    delivery_id: 8,
    event_id: 1,
    event_type: 'task_succeeded',
    message: 'Scheduled task succeeded.',
    navigation: { kind: 'SCHEDULER_RUN', payload: {} },
    occurred_at: '2026-06-11T10:47:21Z',
    severity: 'info',
    source_module: 'scheduler',
    status: 'unread',
    target_ref: '1',
    target_type: 'USER',
    title: 'Scheduled task succeeded',
    ...overrides,
  };
}

function resetRoute(query: Record<string, unknown>) {
  routeState.query = query;
  mocks.routerReplace.mockImplementation(async (location: { query?: Record<string, unknown> }) => {
    const nextQuery = { ...routeState.query, ...(location.query ?? {}) };
    for (const [key, value] of Object.entries(nextQuery)) {
      if (value === undefined) {
        delete nextQuery[key];
      }
    }
    routeState.query = nextQuery;
  });
}

const tdesignStubs = {
  't-button': defineComponent({
    setup(_, { slots }) {
      return () => h('button', slots.default?.());
    },
  }),
  't-tab-panel': defineComponent({
    setup() {
      return () => null;
    },
  }),
  't-tabs': defineComponent({
    setup(_, { slots }) {
      return () => h('div', slots.default?.());
    },
  }),
};

function mountPage() {
  return mount(NotificationListIndex, {
    global: {
      stubs: tdesignStubs,
    },
  });
}

describe('NotificationListIndex', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resetRoute({});
  });

  it('consumes the delivery_id deep link after marking the open unread detail as read', async () => {
    const unread = notification();
    const read = notification({ read_at: '2026-06-11T10:48:00Z', status: 'read' });
    mocks.getNotifications
      .mockResolvedValueOnce({ items: [unread], page: 1, page_size: 20, total: 1 })
      .mockResolvedValueOnce({ items: [], page: 1, page_size: 20, total: 0 });
    mocks.markNotificationRead.mockResolvedValueOnce(read);
    resetRoute({ delivery_id: '8', status: 'unread' });

    const wrapper = mountPage();
    await flushPromises();
    await nextTick();

    expect(wrapper.get('[data-testid="detail-drawer"]').attributes('data-visible')).toBe('true');

    await wrapper.get('[data-testid="detail-mark-read"]').trigger('click');
    await flushPromises();
    await nextTick();

    expect(mocks.markNotificationRead).toHaveBeenCalledWith(8);
    expect(mocks.requestNotificationHeaderRefresh).toHaveBeenCalledTimes(1);
    expect(mocks.routerReplace).toHaveBeenCalledWith({ query: { delivery_id: undefined, status: 'unread' } });
    expect(routeState.query).toEqual({ status: 'unread' });
    expect(wrapper.get('[data-testid="detail-drawer"]').attributes('data-visible')).toBe('false');
  });

  it('clears delivery_id when the detail drawer is closed manually', async () => {
    mocks.getNotifications.mockResolvedValueOnce({ items: [notification()], page: 1, page_size: 20, total: 1 });
    resetRoute({ delivery_id: '8' });

    const wrapper = mountPage();
    await flushPromises();
    await nextTick();

    await wrapper.get('[data-testid="detail-close"]').trigger('click');
    await flushPromises();
    await nextTick();

    expect(mocks.routerReplace).toHaveBeenCalledWith({ query: { delivery_id: undefined } });
    expect(routeState.query).toEqual({});
    expect(wrapper.get('[data-testid="detail-drawer"]').attributes('data-visible')).toBe('false');
  });
});
