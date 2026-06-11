// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import type { ComposerTranslation } from 'vue-i18n';

import type { NotificationItem } from '../types/notification';
import { presentNotification } from './notification-presenter';

const messages: Record<string, string> = {
  'notification.action.openRunRecord': '打开运行记录',
  'notification.category.task': '任务',
  'notification.emptyValue': '无',
  'notification.level.info': '信息',
  'notification.message.scheduler.runSucceeded': '已成功完成。',
  'notification.resourceType.scheduledTaskRun': '定时任务运行记录',
  'scheduler.job.accessLogRetentionCleanup.title': '访问日志保留清理',
  'notification.source.scheduler': '定时任务',
  'notification.status.unread': '未读',
  'notification.title.scheduler.runSucceeded': '定时任务执行成功',
  'notification.unknownLabel': '未知',
};

const t = ((key: string, context?: Record<string, unknown>) => {
  const template = messages[key];
  if (!template) return key;
  return template.replaceAll(/\{(\w+)\}/g, (_, name: string) => String(context?.[name] ?? ''));
}) as ComposerTranslation;

function notification(overrides: Partial<NotificationItem> = {}): NotificationItem {
  return {
    category: 'TASK',
    delivery_created_at: '2026-06-11T10:47:21Z',
    delivery_id: 1,
    event_id: 1,
    event_type: 'task_succeeded',
    message: 'Completed successfully.',
    navigation: { kind: 'SCHEDULER_RUN', payload: {} },
    occurred_at: '2026-06-11T10:47:21Z',
    severity: 'info',
    source_module: 'scheduler',
    status: 'unread',
    target_ref: '1',
    target_type: 'USER',
    title: 'Nightly audit cleanup',
    ...overrides,
  };
}

describe('notification presenter', () => {
  it('maps key-first notification payloads into one unified view model', () => {
    const view = presentNotification(
      notification({
        action_label_key: 'notification.action.openRunRecord',
        category_key: 'notification.category.task',
        context: {
          taskBuiltin: true,
          taskNameKey: 'scheduler.job.accessLogRetentionCleanup.title',
        },
        level_key: 'notification.level.info',
        message_key: 'notification.message.scheduler.runSucceeded',
        resource_id: '25',
        resource_name: 'Access log retention cleanup',
        resource_type: 'scheduled_task_run',
        resource_type_key: 'notification.resourceType.scheduledTaskRun',
        source_key: 'notification.source.scheduler',
        title_key: 'notification.title.scheduler.runSucceeded',
      }),
      t,
      'zh-CN',
    );

    expect(view.title).toBe('访问日志保留清理');
    expect(view.message).toBe('已成功完成。');
    expect(view.compactMeta).toBe(`任务 / 定时任务 · ${view.occurredAtLabel}`);
    expect(view.levelLabel).toBe('信息');
    expect(view.categoryLabel).toBe('任务');
    expect(view.sourceLabel).toBe('定时任务');
    expect(view.resourceTypeLabel).toBe('定时任务运行记录');
    expect(view.statusLabel).toBe('未读');
    expect(view.actionLabel).toBe('打开运行记录');
    expect(view.resourceName).toBe('访问日志保留清理');
    expect(view.resourceId).toBe('25');
  });

  it('keeps a user-created scheduled task title when it uses a built-in job definition', () => {
    const view = presentNotification(
      notification({
        context: {
          taskBuiltin: false,
          taskTitle: '审计日志保留清理1',
          taskNameKey: 'scheduler.job.accessLogRetentionCleanup.title',
          jobTitleKey: 'scheduler.job.accessLogRetentionCleanup.title',
        },
        message_key: 'notification.message.scheduler.runSucceeded',
        resource_name: 'Audit log retention cleanup',
        resource_type: 'scheduled_task_run',
        title_key: 'notification.title.scheduler.runSucceeded',
      }),
      t,
      'zh-CN',
    );

    expect(view.title).toBe('审计日志保留清理1');
    expect(view.resourceName).toBe('审计日志保留清理1');
    expect(view.message).toBe('已成功完成。');
  });

  it('uses fallback copy when a display key is missing', () => {
    const view = presentNotification(notification(), t, 'zh-CN');

    expect(view.title).toBe('Nightly audit cleanup');
    expect(view.message).toBe('Completed successfully.');
  });

  it('keeps the literal scheduler task title when taskNameKey is present without builtin status', () => {
    const view = presentNotification(
      notification({
        context: {
          taskName: '审计日志保留清理1',
          taskNameKey: 'scheduler.job.accessLogRetentionCleanup.title',
        },
        message_key: 'notification.message.scheduler.runSucceeded',
        resource_type: 'scheduled_task_run',
        title_key: 'notification.title.scheduler.runSucceeded',
      }),
      t,
      'zh-CN',
    );

    expect(view.title).toBe('审计日志保留清理1');
    expect(view.message).toBe('已成功完成。');
  });

  it('falls back to the stored title when builtin localization key is missing', () => {
    const view = presentNotification(
      notification({
        context: {
          taskBuiltin: true,
          taskNameKey: 'scheduler.job.missing.title',
        },
        message_key: 'notification.message.scheduler.runSucceeded',
        resource_type: 'scheduled_task_run',
        title_key: 'notification.title.scheduler.runSucceeded',
      }),
      t,
      'zh-CN',
    );

    expect(view.title).toBe('Nightly audit cleanup');
    expect(view.message).toBe('已成功完成。');
  });

  it('falls back to the stored scheduler title when task context is missing', () => {
    const view = presentNotification(
      notification({
        message_key: 'notification.message.scheduler.runSucceeded',
        resource_type: 'scheduled_task_run',
        title_key: 'notification.title.scheduler.runSucceeded',
      }),
      t,
      'zh-CN',
    );

    expect(view.title).toBe('Nightly audit cleanup');
    expect(view.message).toBe('已成功完成。');
  });

  it('uses unknown labels when both key and fallback are missing', () => {
    const view = presentNotification(
      notification({
        action_label: '',
        action_label_key: '',
        category: 'CONFIG',
        category_key: '',
        message: '',
        message_key: '',
        resource_type: '',
        source_module: 'unknown-module',
        source_key: '',
        title: '',
        title_key: '',
      }),
      t,
      'zh-CN',
    );

    expect(view.title).toBe('未知');
    expect(view.message).toBe('未知');
    expect(view.sourceLabel).toBe('未知');
    expect(view.resourceTypeLabel).toBe('未知');
  });
});
