// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { defineComponent, h } from 'vue';

import type { NotificationItem } from '../types/notification';
import NotificationDetailDrawer from './NotificationDetailDrawer.vue';
import NotificationTable from './NotificationTable.vue';

vi.mock('@/shared/components/management', () => ({
  calculateTableContentWidth: () => 1000,
  createActionColumn: (title: string, width: number) => ({ colKey: 'operation', title, width }),
  createConfiguredColumns: (columns: Array<{ key: string; title: string; config?: Record<string, unknown> }>) =>
    columns.map((column) => ({ colKey: column.key, title: column.title, ...(column.config ?? {}) })),
  formatCompactDateTime: () => '2026/06/11 10:47:21',
}));

vi.mock('../contract/navigation', () => ({
  NOTIFICATION_NAVIGATION_KIND: {
    AUDIT_INCIDENT: 'AUDIT_INCIDENT',
    AUDIT_LOG: 'AUDIT_LOG',
    MODULE_RUNTIME_ITEM: 'MODULE_RUNTIME_ITEM',
    SCHEDULER_RUN: 'SCHEDULER_RUN',
    SYSTEM_CONFIG_ITEM: 'SYSTEM_CONFIG_ITEM',
  },
  resolveNotificationNavigationLocation: () => ({ path: '/scheduled-tasks/runs' }),
}));

const messages: Record<string, string> = {
  'notification.action.delete': '删除',
  'notification.action.detail': '详情',
  'notification.action.markRead': '标记已读',
  'notification.action.openRunRecord': '打开运行记录',
  'notification.category.task': '任务',
  'notification.columns.actions': '操作',
  'notification.columns.category': '分类',
  'notification.columns.notification': '通知',
  'notification.columns.occurredAt': '发生时间',
  'notification.columns.severity': '级别',
  'notification.columns.sourceModule': '来源',
  'notification.columns.status': '状态',
  'notification.detail.basic': '基础信息',
  'notification.detail.navigation': '业务上下文',
  'notification.detail.readAt': '已读时间',
  'notification.detail.resource': '关联资源',
  'notification.detail.resourceId': '资源 ID',
  'notification.detail.resourceName': '资源名称',
  'notification.detail.resourceType': '资源类型',
  'notification.detail.resultSummary': '结果摘要',
  'notification.detail.title': '通知详情',
  'notification.emptyValue': '无',
  'notification.level.info': '信息',
  'notification.message.scheduler.runSucceeded': '{taskName}已成功完成。',
  'notification.navigation.schedulerRun': '定时任务运行',
  'notification.resourceType.scheduledTaskRun': '定时任务运行记录',
  'scheduler.job.accessLogRetentionCleanup.title': '访问日志保留清理',
  'notification.source.scheduler': '定时任务',
  'notification.status.read': '已读',
  'notification.status.unread': '未读',
  'notification.table.summary': '共 1 条通知',
  'notification.table.title': '通知列表',
  'notification.title.scheduler.runSucceeded': '定时任务执行成功',
  'notification.unknownLabel': '未知',
};

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    locale: { value: 'zh-CN' },
    t: (key: string, context?: Record<string, unknown>) => {
      const template = messages[key];
      if (!template) return key;
      return template.replaceAll(/\{(\w+)\}/g, (_, name: string) => String(context?.[name] ?? ''));
    },
  }),
}));

const passthroughStub = defineComponent({
  setup(_, { slots }) {
    return () => h('div', slots.default?.());
  },
});

const drawerStub = defineComponent({
  setup(_, { slots }) {
    return () => h('div', [slots.header?.(), slots.default?.()]);
  },
});

const buttonStub = defineComponent({
  emits: ['click'],
  props: {
    loading: { type: Boolean, default: false },
  },
  setup(props, { emit, slots }) {
    return () =>
      h(
        'button',
        {
          'data-loading': props.loading ? 'true' : 'false',
          onClick: () => emit('click'),
        },
        slots.default?.(),
      );
  },
});

const tableStub = defineComponent({
  props: {
    data: { type: Array, default: () => [] },
  },
  setup(props, { slots }) {
    return () =>
      h(
        'div',
        (props.data as unknown[]).map((row) =>
          h('div', [
            slots.notification?.({ row }),
            slots.severity?.({ row }),
            slots.category?.({ row }),
            slots.source_module?.({ row }),
            slots.status?.({ row }),
            slots.occurred_at?.({ row }),
            slots.operation?.({ row }),
          ]),
        ),
      );
  },
});

const stubs = {
  't-button': buttonStub,
  't-card': passthroughStub,
  't-drawer': drawerStub,
  't-empty': passthroughStub,
  't-space': passthroughStub,
  't-table': tableStub,
  't-tag': passthroughStub,
};

function notification(): NotificationItem {
  return {
    action_label_key: 'notification.action.openRunRecord',
    category: 'TASK',
    category_key: 'notification.category.task',
    context: {
      taskName: 'Access log retention cleanup',
      taskNameKey: 'scheduler.job.accessLogRetentionCleanup.title',
    },
    delivery_created_at: '2026-06-11T10:47:21Z',
    delivery_id: 1,
    event_id: 1,
    event_type: 'task_succeeded',
    level_key: 'notification.level.info',
    message: 'Scheduled task Access log retention cleanup succeeded.',
    message_key: 'notification.message.scheduler.runSucceeded',
    navigation: { kind: 'SCHEDULER_RUN', payload: {} },
    occurred_at: '2026-06-11T10:47:21Z',
    resource_id: '25',
    resource_name: 'Access log retention cleanup',
    resource_type: 'scheduled_task_run',
    resource_type_key: 'notification.resourceType.scheduledTaskRun',
    severity: 'info',
    source_key: 'notification.source.scheduler',
    source_module: 'scheduler',
    status: 'unread',
    target_ref: '1',
    target_type: 'USER',
    title: 'Scheduled task succeeded',
    title_key: 'notification.title.scheduler.runSucceeded',
  };
}

describe('notification display consistency', () => {
  it('renders list and detail from the same notification view model fields', () => {
    const item = notification();
    const table = mount(NotificationTable, {
      props: {
        current: 1,
        emptyDescription: '',
        emptyTitle: '',
        items: [item],
        pageSize: 20,
        total: 1,
      },
      global: { stubs },
    });
    const detail = mount(NotificationDetailDrawer, {
      props: {
        item,
        visible: true,
      },
      global: { stubs },
    });

    for (const expected of ['定时任务执行成功', '访问日志保留清理已成功完成。', '信息', '任务', '定时任务']) {
      expect(table.text()).toContain(expected);
      expect(detail.text()).toContain(expected);
    }
    expect(detail.text()).toContain('访问日志保留清理');
    expect(detail.text()).toContain('定时任务运行记录');
    expect(detail.text()).toContain('标记已读');
    expect(detail.text()).toContain('打开运行记录');
    expect(table.text()).not.toContain('标记已读');
  });

  it('emits mark-read from the unread detail drawer action', async () => {
    const item = notification();
    const detail = mount(NotificationDetailDrawer, {
      props: {
        item,
        markingRead: true,
        visible: true,
      },
      global: { stubs },
    });

    const markReadButton = detail.findAll('button').find((button) => button.text() === '标记已读');

    expect(markReadButton?.attributes('data-loading')).toBe('true');
    await markReadButton?.trigger('click');
    expect(detail.emitted('mark-read')?.[0]).toEqual([item]);
  });

  it('shows read status in the detail header after the notification is read', () => {
    const item = {
      ...notification(),
      read_at: '2026-06-11T10:48:00Z',
      status: 'read',
    } satisfies NotificationItem;
    const detail = mount(NotificationDetailDrawer, {
      props: {
        item,
        visible: true,
      },
      global: { stubs },
    });

    expect(detail.text()).toContain('已读');
    expect(detail.findAll('button').some((button) => button.text() === '标记已读')).toBe(false);
  });
});
