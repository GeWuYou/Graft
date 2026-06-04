import type { ComposerTranslation } from 'vue-i18n';

import type { AppLogItem, AppLogSeverity } from '../types/app-log';

export function appLogSeverityTheme(severity: AppLogSeverity) {
  switch (severity) {
    case 'error':
      return 'danger';
    case 'warn':
      return 'warning';
    case 'debug':
      return 'default';
    default:
      return 'primary';
  }
}

export function appLogOperationText(record: Pick<AppLogItem, 'operation'>, t: ComposerTranslation) {
  return record.operation?.trim() || t('appLog.values.noOperation');
}

export function appLogCorrelationText(record: Pick<AppLogItem, 'request_id' | 'trace_id'>, t: ComposerTranslation) {
  if (record.request_id) {
    return record.request_id;
  }
  if (record.trace_id) {
    return record.trace_id;
  }
  return t('appLog.values.noCorrelation');
}

export function appLogFieldsCount(record: Pick<AppLogItem, 'fields'>) {
  return Object.keys(record.fields || {}).length;
}
