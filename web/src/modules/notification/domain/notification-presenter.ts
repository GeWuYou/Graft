import type { ComposerTranslation } from 'vue-i18n';

import { formatCompactDateTime } from '@/shared/components/management';

import {
  NOTIFICATION_CATEGORY_LABEL_KEY_BY_VALUE,
  NOTIFICATION_RESOURCE_TYPE,
  NOTIFICATION_RESOURCE_TYPE_LABEL_KEY_BY_VALUE,
  NOTIFICATION_SOURCE_LABEL_KEY_BY_MODULE,
} from '../contract/presentation';
import type { NotificationItem } from '../types/notification';

export const NOTIFICATION_MVP_SOURCE_MODULES = [
  'notification',
  'scheduler',
  'audit',
  'system-config',
  'access-log',
] as const;

type NotificationContext = Record<string, unknown>;

export interface NotificationViewModel {
  actionLabel: string;
  categoryLabel: string;
  compactMeta: string;
  deliveryId: NotificationItem['delivery_id'];
  eventId: NotificationItem['event_id'];
  levelLabel: string;
  message: string;
  occurredAtLabel: string;
  readAtLabel: string;
  resourceId: string;
  resourceName: string;
  resourceTypeLabel: string;
  sourceLabel: string;
  status: NotificationItem['status'];
  statusLabel: string;
  title: string;
}

const UNKNOWN_LABEL = 'notification.unknownLabel';
const EMPTY_VALUE = 'notification.emptyValue';

export function presentNotification(
  item: NotificationItem,
  t: ComposerTranslation,
  locale: string,
): NotificationViewModel {
  const context = notificationContext(item);
  const categoryLabel = resolveNotificationCategory(item, t);
  const sourceLabel = resolveNotificationSource(item, t);
  const occurredAtLabel = formatCompactDateTime(item.occurred_at, locale);

  return {
    actionLabel: resolveNotificationActionLabel(item, t, context),
    categoryLabel,
    compactMeta: `${categoryLabel} / ${sourceLabel} · ${occurredAtLabel}`,
    deliveryId: item.delivery_id,
    eventId: item.event_id,
    levelLabel: resolveNotificationLevel(item, t),
    message: resolveNotificationMessage(item, t, context),
    occurredAtLabel,
    readAtLabel: item.read_at
      ? formatCompactDateTime(item.read_at, locale)
      : resolveLabel(t, 'notification.status.unread'),
    resourceId: valueOrEmpty(item.resource_id, t),
    resourceName: resolveNotificationResourceName(item, t, context),
    resourceTypeLabel: resolveNotificationResourceType(item, t),
    sourceLabel,
    status: item.status,
    statusLabel: resolveNotificationStatus(item, t),
    title: resolveNotificationTitle(item, t, context),
  };
}

export function notificationSeverityTheme(severity: NotificationItem['severity']) {
  switch (severity) {
    case 'critical':
    case 'error':
      return 'danger';
    case 'warning':
      return 'warning';
    case 'info':
      return 'primary';
    default:
      return 'default';
  }
}

export function notificationStatusTheme(status: NotificationItem['status']) {
  return status === 'unread' ? 'primary' : 'default';
}

function resolveNotificationTitle(item: NotificationItem, t: ComposerTranslation, context: NotificationContext) {
  if (isSchedulerTaskRun(item)) {
    return resolveSchedulerTaskRunTitle(item, t, context) || unknown(t);
  }
  return resolveKeyFirst(t, item.title_key, context, item.title);
}

function resolveNotificationResourceName(item: NotificationItem, t: ComposerTranslation, context: NotificationContext) {
  if (isSchedulerTaskRun(item)) {
    const taskDisplayTitle = resolveSchedulerTaskDisplayTitle(t, context, item.resource_name);
    if (taskDisplayTitle) return taskDisplayTitle;
  }
  return valueOrEmpty(item.resource_name, t);
}

function resolveSchedulerTaskRunTitle(item: NotificationItem, t: ComposerTranslation, context: NotificationContext) {
  const taskDisplayTitle = resolveSchedulerTaskDisplayTitle(t, context, item.title);
  if (taskDisplayTitle) {
    return taskDisplayTitle;
  }
  return translateKey(t, item.title_key, context);
}

function resolveSchedulerTaskDisplayTitle(
  t: ComposerTranslation,
  context: NotificationContext,
  storedFallback?: unknown,
) {
  const literalTaskTitle = stringValue(context.taskTitle) || stringValue(context.taskName);
  const storedTitle = fallbackLabel(storedFallback);

  if (context.taskBuiltin === true || context.builtin === true) {
    const localizedTaskTitle =
      translateKey(t, stringValue(context.taskTitleKey)) || translateKey(t, stringValue(context.taskNameKey));
    return localizedTaskTitle || literalTaskTitle || storedTitle;
  }

  return literalTaskTitle || storedTitle;
}

function resolveNotificationMessage(item: NotificationItem, t: ComposerTranslation, context: NotificationContext) {
  return resolveKeyFirst(t, item.message_key, context, item.message);
}

function resolveNotificationCategory(item: NotificationItem, t: ComposerTranslation) {
  return resolveNotificationCategoryValue(item.category, t, item.category_key);
}

function resolveNotificationSource(item: NotificationItem, t: ComposerTranslation) {
  return resolveNotificationSourceValue(item.source_module, t, item.source_key);
}

function resolveNotificationLevel(item: NotificationItem, t: ComposerTranslation) {
  return resolveNotificationLevelValue(item.severity, t, item.level_key);
}

export function resolveNotificationCategoryValue(value: string, t: ComposerTranslation, key?: string | null) {
  return resolveKeyFirst(t, key || NOTIFICATION_CATEGORY_LABEL_KEY_BY_VALUE[value]);
}

export function resolveNotificationSourceValue(value: string, t: ComposerTranslation, key?: string | null) {
  return resolveKeyFirst(t, key || NOTIFICATION_SOURCE_LABEL_KEY_BY_MODULE[value]);
}

export function resolveNotificationLevelValue(value: string, t: ComposerTranslation, key?: string | null) {
  return resolveKeyFirst(t, key || `notification.level.${value}`);
}

function resolveNotificationStatus(item: NotificationItem, t: ComposerTranslation) {
  return resolveKeyFirst(t, `notification.status.${item.status}`);
}

function resolveNotificationActionLabel(item: NotificationItem, t: ComposerTranslation, context: NotificationContext) {
  return resolveKeyFirst(t, item.action_label_key, context, item.action_label);
}

function resolveNotificationResourceType(item: NotificationItem, t: ComposerTranslation) {
  return resolveKeyFirst(
    t,
    item.resource_type_key || NOTIFICATION_RESOURCE_TYPE_LABEL_KEY_BY_VALUE[item.resource_type ?? ''],
    undefined,
  );
}

export function resolveNotificationResultSummary(item: NotificationItem, t: ComposerTranslation) {
  return valueOrEmpty(rawNotificationContext(item).resultSummary, t);
}

function isSchedulerTaskRun(item: NotificationItem) {
  return item.source_module === 'scheduler' && item.resource_type === NOTIFICATION_RESOURCE_TYPE.SCHEDULED_TASK_RUN;
}

function resolveKeyFirst(
  t: ComposerTranslation,
  key?: string | null,
  context?: NotificationContext,
  fallback?: unknown,
) {
  return translateKey(t, key, context) || fallbackLabel(fallback) || unknown(t);
}

function translateKey(t: ComposerTranslation, key: string | null | undefined, context?: NotificationContext) {
  const normalized = key?.trim();
  if (!normalized) return '';
  const translated = context ? t(normalized, context) : t(normalized);
  return translated === normalized ? '' : translated;
}

function notificationContext(item: NotificationItem): NotificationContext {
  return { ...rawNotificationContext(item) };
}

function rawNotificationContext(item: NotificationItem): NotificationContext {
  return item.context && typeof item.context === 'object' ? item.context : {};
}

function stringValue(value: unknown) {
  return typeof value === 'string' && value.trim() ? value.trim() : '';
}

function valueOrEmpty(value: unknown, t: ComposerTranslation) {
  return typeof value === 'string' && value.trim() ? value.trim() : resolveLabel(t, EMPTY_VALUE);
}

function fallbackLabel(value: unknown) {
  return typeof value === 'string' && value.trim() ? value.trim() : '';
}

function unknown(t: ComposerTranslation) {
  return resolveLabel(t, UNKNOWN_LABEL);
}

function resolveLabel(t: ComposerTranslation, key: string) {
  const translated = t(key);
  return translated === key ? '-' : translated;
}
