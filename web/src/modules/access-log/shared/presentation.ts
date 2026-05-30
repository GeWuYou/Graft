import type { ComposerTranslation } from 'vue-i18n';

import type { AccessLogItem } from '../types/access-log';

export function accessLogPathSecondary(record: Pick<AccessLogItem, 'path' | 'route'>) {
  const route = record.route?.trim();
  if (!route || route === record.path) {
    return '';
  }
  return route;
}

export function accessLogUserPrimary(record: Pick<AccessLogItem, 'username'>, t: ComposerTranslation) {
  const username = record.username?.trim();
  return username || t('accessLog.user.anonymous');
}

export function accessLogUserSecondary(record: Pick<AccessLogItem, 'user_id' | 'username'>, t: ComposerTranslation) {
  if (record.user_id !== null && record.user_id !== undefined) {
    return t('accessLog.user.userIdValue', { id: record.user_id });
  }
  if (record.username?.trim()) {
    return t('accessLog.user.noUserId');
  }
  return t('accessLog.user.unauthenticated');
}
