import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import { defineComponent, h } from 'vue';
import type { ComposerTranslation } from 'vue-i18n';
import { createI18n } from 'vue-i18n';

import { presentNotification } from '../shared/presentation';
import type { NotificationItem } from '../types/notification';
import NotificationDetailDrawer from './NotificationDetailDrawer.vue';

const t = ((key: string, context?: unknown) => {
  const messages: Record<string, string> = {
    'notification.action.openAuditIncident': '查看审计事件',
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

const i18n = createI18n({
  legacy: false,
  locale: 'zh-CN',
  messages: {
    'zh-CN': {
      notification: {
        action: {
          openAuditIncident: '查看审计事件',
          openAuditLog: '查看审计日志',
          openRunRecord: '打开运行记录',
          openTarget: '打开相关页面',
          markRead: '标记已读',
        },
        category: {
          security: '安全',
          task: '任务',
        },
        source: {
          audit: '安全审计',
          scheduler: '定时任务',
        },
        detail: {
          title: '通知详情',
          auditSummary: '审计摘要',
          auditAction: '审计动作',
          auditResult: '审计结果',
          auditResource: '审计资源',
          auditRequestId: '请求 ID',
          auditTraceId: 'Trace ID',
          auditRiskLevel: '风险等级',
          auditReason: '审计原因',
          basic: '基础信息',
          readAt: '已读时间',
          resource: '关联资源',
          resourceName: '资源名称',
          resourceType: '资源类型',
          resourceId: '资源 ID',
          resultSummary: '结果摘要',
          navigation: '业务上下文',
          unsupportedNavigation: '该通知的跳转目标将在后续阶段开放。',
        },
        columns: {
          status: '状态',
          severity: '级别',
          category: '分类',
          sourceModule: '来源',
          occurredAt: '发生时间',
        },
        navigation: {
          auditIncident: '审计事件',
          auditLog: '审计日志',
          schedulerRun: '定时任务运行',
          systemConfigItem: '系统配置',
          moduleRuntimeItem: '模块运行项',
          unknown: '未知上下文',
        },
        status: {
          read: '已读',
          unread: '未读',
        },
        level: {
          warning: '警告',
          info: '信息',
        },
        message: {
          audit: {
            highRisk: '高风险审计活动需要复核。',
          },
        },
        title: {
          audit: {
            highRisk: '高风险审计事件',
          },
        },
        resourceType: {
          scheduledTaskRun: '定时任务运行记录',
        },
        emptyValue: '无',
        unknownLabel: '未知',
      },
    },
  },
});

const tdesignStubs = {
  't-button': defineComponent({
    emits: ['click'],
    setup(_, { attrs, emit, slots }) {
      return () =>
        h(
          'button',
          {
            ...attrs,
            onClick: () => emit('click'),
          },
          slots.default?.(),
        );
    },
  }),
  't-drawer': defineComponent({
    props: { visible: { type: Boolean, default: false } },
    emits: ['update:visible'],
    setup(props, { slots }) {
      return () => h('section', { 'data-visible': String(props.visible) }, [slots.header?.(), slots.default?.()]);
    },
  }),
  't-tag': defineComponent({
    setup(_, { slots }) {
      return () => h('span', slots.default?.());
    },
  }),
};

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

  it('keeps drawer-level audit CTA and summary fallbacks aligned with audit navigation semantics', () => {
    const item = {
      action_label: '查看审计日志',
      action_label_key: 'notification.action.openAuditLog',
      category: 'SECURITY',
      category_key: 'notification.category.security',
      context: {
        reason: 'High-risk audit activity needs review.',
        resourceName: 'web-1',
      },
      delivery_created_at: '2026-06-11T10:47:21Z',
      delivery_id: 1,
      event_id: 21,
      event_type: 'high_risk',
      level_key: 'notification.level.warning',
      message: 'High-risk audit activity needs review.',
      message_key: 'notification.message.audit.highRisk',
      navigation: { kind: 'AUDIT_INCIDENT', payload: { incident_id: '21' } },
      occurred_at: '2026-06-11T10:47:21Z',
      resource_id: '21',
      resource_name: 'web-1',
      resource_type: 'audit_log',
      resource_type_key: 'notification.resourceType.scheduledTaskRun',
      severity: 'warning',
      source_key: 'notification.source.audit',
      source_module: 'audit',
      status: 'read',
      target_ref: '1',
      target_type: 'USER',
      title: 'High-risk audit event',
      title_key: 'notification.title.audit.highRisk',
    } satisfies NotificationItem;

    const wrapper = mount(NotificationDetailDrawer, {
      props: {
        item,
        visible: true,
      },
      global: {
        plugins: [i18n],
        stubs: tdesignStubs,
      },
    });

    const text = wrapper.text();
    expect(text).toContain('查看审计事件');
    expect(text).not.toContain('查看审计日志');
    expect(text).toContain('审计结果无');
    expect(text).toContain('审计动作无');
    expect(text).toContain('风险等级无');
    expect(text).toContain('High-risk audit activity needs review.');
  });
});
