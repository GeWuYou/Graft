// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { collectLocaleCatalogs, localeViolation } from '../locale-utils';
import type { I18nGovernanceRule, RuleViolation } from '../types';

const REQUIRED_NOTIFICATION_KEYS = [
  'notification.action.markAllRead',
  'notification.action.openNotificationCenter',
  'notification.action.openRelatedPage',
  'notification.action.openRunRecord',
  'notification.action.viewAll',
  'notification.category.task',
  'notification.emptyValue',
  'notification.level.info',
  'notification.message.scheduler.runSucceeded',
  'notification.resourceType.scheduledTaskRun',
  'notification.source.scheduler',
  'notification.status.read',
  'notification.status.unread',
  'notification.title.scheduler.runSucceeded',
  'notification.unknownLabel',
] as const;

const SUGGESTION =
  'Define notification presenter keys in both web/src/modules/notification/locales/zh-CN.json and en-US.json.';

function hasNotificationPresenterSurface(context: Parameters<I18nGovernanceRule['check']>[0]) {
  return context.sourceFiles.some(
    (file) => file.relativePath === 'src/modules/notification/domain/notification-presenter.ts',
  );
}

export const notificationRequiredKeysetRule: I18nGovernanceRule = {
  id: 'notification-required-keyset',
  description: 'Blocks Notification Center key-first presenter drift by requiring the core zh-CN/en-US key set.',
  defaultSeverity: 'error',
  appliesTo: ['locale'],
  check(context) {
    if (!hasNotificationPresenterSurface(context)) return [];

    const catalogs = collectLocaleCatalogs(context).filter((catalog) =>
      catalog.file.startsWith('src/modules/notification/locales/'),
    );
    const byLocale = new Map(catalogs.map((catalog) => [catalog.locale, catalog]));
    const violations: RuleViolation[] = [];

    for (const locale of ['zh-CN', 'en-US'] as const) {
      const catalog = byLocale.get(locale);
      if (!catalog) {
        violations.push(
          localeViolation(
            notificationRequiredKeysetRule.id,
            'error',
            `src/modules/notification/locales/${locale}.json`,
            `missing notification ${locale} locale catalog`,
            SUGGESTION,
          ),
        );
        continue;
      }

      for (const key of REQUIRED_NOTIFICATION_KEYS) {
        if (catalog.messages.has(key)) continue;
        violations.push(
          localeViolation(
            notificationRequiredKeysetRule.id,
            'error',
            catalog.file,
            `missing required notification key ${key}`,
            SUGGESTION,
          ),
        );
      }
    }

    return violations;
  },
};
