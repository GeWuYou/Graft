import { describe, expect, it } from 'vitest';
import type { ComposerTranslation } from 'vue-i18n';

import { presentNotification } from '../shared/presentation';
import type { NotificationItem } from '../types/notification';

const t = ((key: string, context?: unknown) => {
  const messages: Record<string, string> = {
    'notification.action.openAuditLog': '查看审计日志',
    'notification.action.openRunRecord': '打开运行记录',
    'notification.action.openTarget': '打开相关页面',
    'notification.category.security': '安全',
    'notification.category.task': '任务',
    'notification.level.high': '高',
    'notification.level.info': '信息',
    'notification.message.audit.highRisk': '高风险审计活动需要复核。',
    'notification.message.scheduler.runSucceeded': '已成功完成。',
    'scheduler.job.accessLogRetentionCleanup.title': '访问日志保留清理',
    'notification.title.scheduler.runSucceeded': '定时任务执行成功',
    'notification.resourceType.scheduledTaskRun': '定时任务运行记录',
    'notification.source.audit': '安全审计',
    'notification.source.scheduler': '定时任务',
    'notification.status.read': '已读',
    'notification.status.unread': '未读',
    'notification.title.audit.highRisk': '高风险审计事件',
    'notification.unknownLabel': '未知',
    'notification.emptyValue': '无',
  };
  const template = messages[key];
  if (!template) return key;
  const values = context && typeof context === 'object' ? (context as Record<string, unknown>) : {};
  return template.replaceAll(/\{(\w+)\}/g, (_, name: string) => String(values[name] ?? ''));
}) as ComposerTranslation;

describe('notification presentation', () => {
  it('keeps scheduler notifications localized through the shared presenter', () => {
    const item = {
      action_label_key: 'notification.action.openRunRecord',
      category: 'TASK',
      category_key: 'notification.category.task',
      context: {
        taskBuiltin: true,
        taskTitle: 'Access log retention cleanup',
        taskTitleKey: 'scheduler.job.accessLogRetentionCleanup.title',
      },
      delivery_created_at: '2026-06-11T10:47:21Z',
      delivery_id: 1,
      event_id: 1,
      event_type: 'task_succeeded',
      level_key: 'notification.level.info',
      message: 'Completed successfully.',
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
      title: 'Nightly audit cleanup',
      title_key: 'notification.title.scheduler.runSucceeded',
    } satisfies NotificationItem;

    const view = presentNotification(item, t, 'zh-CN');

    expect(view.title).toBe('访问日志保留清理');
    expect(view.message).toBe('已成功完成。');
    expect(view.actionLabel).toBe('打开运行记录');
    expect(view.resourceName).toBe('访问日志保留清理');
  });

  it('keeps audit notifications localized and exposes audit-specific copy', () => {
    const item = {
      action_label: '查看审计日志',
      action_label_key: 'notification.action.openAuditLog',
      category: 'SECURITY',
      category_key: 'notification.category.security',
      context: {
        action: 'ops.container.action.start',
        reason: 'Container start completed.',
        requestId: 'req-1',
        resourceName: 'web-1',
        resourceType: 'container',
        result: 'SUCCESS',
        riskLevel: 'HIGH',
        traceId: 'trace-1',
      },
      delivery_created_at: '2026-06-11T10:47:21Z',
      delivery_id: 1,
      event_id: 21,
      event_type: 'high_risk',
      level_key: 'notification.level.warning',
      message: 'High-risk audit activity needs review.',
      message_key: 'notification.message.audit.highRisk',
      navigation: { kind: 'AUDIT_LOG', payload: { audit_log_id: 21, request_id: 'req-1', trace_id: 'trace-1' } },
      occurred_at: '2026-06-11T10:47:21Z',
      resource_id: '21',
      resource_name: 'web-1',
      resource_type: 'audit_log',
      resource_type_key: 'notification.resourceType.scheduledTaskRun',
      severity: 'warning',
      source_key: 'notification.source.audit',
      source_module: 'audit',
      status: 'unread',
      target_ref: '1',
      target_type: 'USER',
      title: 'High-risk audit event',
      title_key: 'notification.title.audit.highRisk',
    } satisfies NotificationItem;

    const view = presentNotification(item, t, 'zh-CN');

    expect(view.title).toBe('高风险审计事件');
    expect(view.message).toBe('高风险审计活动需要复核。');
    expect(view.actionLabel).toBe('查看审计日志');
    expect(view.sourceLabel).toBe('安全审计');
  });
});
