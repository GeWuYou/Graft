import { formatLocaleDateTime } from '@/shared/observability';

import type { AuditLogListItem } from '../types/audit';
import type { AuditBusinessCategory } from '../types/audit';
import type { AuditResult as AuditResultEnum, AuditRiskLevel as AuditRiskLevelEnum } from '../types/audit';
import type { AuditSorter } from '../types/audit';

type Translate = (key: string, params?: Record<string, unknown>) => string;
type AuditPresentationRecord = {
  action?: string;
  result?: AuditResultEnum;
  source?: AuditSourceValue;
  resource_id?: string;
  resource_name?: string;
  resource_type?: string;
  target_label?: string | null;
  target_type?: string | null;
  request_path?: string;
  message?: string;
  metadata?: Record<string, unknown>;
};

export type AuditRiskValue = 'all' | AuditRiskLevelEnum;
export type AuditResultValue = 'all' | AuditResultEnum;

export type AuditClientFilterState = {
  keyword: string;
  actor: string;
  success: 'all' | 'true' | 'false';
  action: string;
  actionPrefix: string;
  actionPrefixes: string[];
  actionKeywords: string[];
  requestPathPrefixes: string[];
  source: string;
  businessCategory: '' | AuditBusinessCategory;
  createdRange: string[];
  resourceType: string;
  resourceTypes: string[];
  resourceName: string;
  resourceId: string;
  result: AuditResultValue;
  results: AuditResultEnum[];
  riskLevel: 'all' | AuditRiskValue;
  riskLevels: AuditRiskLevelEnum[];
  session: string;
  requestId: string;
  sorters: AuditSorter[];
};

type AuditSourceValue = 'REQUEST' | 'SECURITY_EVENT' | 'DOMAIN_EVENT' | 'UNKNOWN';

export function actorLabel(row: AuditLogListItem, t: Translate) {
  return row.actor_display_name || row.actor_username || t('audit.common.unknownActor');
}

export function actorSecondaryLabel(row: AuditLogListItem) {
  return row.actor_username || row.actor_user_id?.toString() || '-';
}

export function resourceLabel(row: AuditPresentationRecord, t: Translate) {
  return (
    row.target_label ||
    row.resource_name ||
    resourceSecondaryLabel(row, t) ||
    row.request_path ||
    metadataLookup(row, 'request_path') ||
    t('audit.common.unknownResource')
  );
}

function resourceSecondaryLabel(row: AuditPresentationRecord, t: Translate) {
  const secondary = [targetTypeLabel(row.target_type ?? row.resource_type, t), row.resource_id].filter(Boolean);
  return secondary.join(' / ');
}

export function resourceDetailLabel(row: AuditLogListItem, t: Translate) {
  const label = row.target_label || row.resource_name || resourceSecondaryLabel(row, t) || row.request_path;
  return (
    [label, targetTypeLabel(row.target_type ?? row.resource_type, t), row.resource_id].filter(Boolean).join(' / ') ||
    t('audit.common.unknownResource')
  );
}

export function requestIdForRecord(row: AuditLogListItem) {
  return row.request_id || metadataLookup(row, 'request_id') || '-';
}

export function sessionIdForRecord(row: AuditLogListItem) {
  return row.session_id || metadataLookup(row, 'session_id') || '-';
}

export function traceIdForRecord(row: AuditLogListItem) {
  return row.trace_id || metadataLookup(row, 'traceId') || metadataLookup(row, 'trace_id') || '-';
}

export function eventTypeForRecord(row: AuditLogListItem) {
  return metadataLookup(row, 'eventType') || metadataLookup(row, 'event_type') || row.action || '-';
}

export function permissionForRecord(row: AuditLogListItem) {
  return metadataLookup(row, 'permission') || (row.resource_type === 'permission' ? row.resource_id : '') || '-';
}

export function securityTargetForRecord(row: AuditLogListItem, t: Translate) {
  return (
    metadataLookup(row, 'targetName') ||
    metadataLookup(row, 'target_name') ||
    resourceDetailLabel(row, t) ||
    permissionForRecord(row)
  );
}

export function reasonForRecord(row: AuditLogListItem, t: Translate) {
  return (
    metadataLookup(row, 'reason') ||
    metadataLookup(row, 'deny_reason') ||
    metadataLookup(row, 'error_reason') ||
    row.message ||
    t('audit.logList.reasonFallback')
  );
}

function sourceForRecord(row: AuditPresentationRecord): AuditSourceValue {
  const source = (
    metadataLookup(row, 'auditSource') ||
    metadataLookup(row, 'audit_source') ||
    metadataLookup(row, 'source')
  )
    .trim()
    .toUpperCase();

  if (source === 'REQUEST' || source === 'SECURITY_EVENT' || source === 'DOMAIN_EVENT') {
    return source;
  }

  if (row.result === 'DENIED' || row.result === 'ERROR') {
    return 'SECURITY_EVENT';
  }

  return 'UNKNOWN';
}

export function sourceLabel(row: AuditPresentationRecord, t: Translate) {
  return t(`audit.common.source.${row.source || sourceForRecord(row)}`);
}

function translateIfPresent(t: Translate, key: string, fallback: string) {
  const translated = t(key);
  return translated === key ? fallback : translated;
}

export function actionCategoryLabel(row: AuditPresentationRecord, t: Translate) {
  return sourceLabel(row, t);
}

export function actionTitle(row: AuditPresentationRecord, t: Translate) {
  const actionKey = row.action?.trim();
  if (!actionKey) {
    return t('audit.common.unknownResource');
  }

  return translateIfPresent(t, `audit.actionLabel.${actionKey}`, actionCategoryLabel(row, t));
}

export function metadataLookup(row: AuditPresentationRecord, key: string) {
  const metadata = row.metadata;
  if (!metadata || typeof metadata !== 'object' || !(key in metadata)) {
    return '';
  }

  const value = metadata[key];
  return typeof value === 'string' || typeof value === 'number' ? String(value) : JSON.stringify(value);
}

export function isSensitiveAction(row: AuditLogListItem) {
  return ['HIGH', 'CRITICAL'].includes(row.risk_level ?? '');
}

function riskValue(row: AuditLogListItem): AuditRiskLevelEnum {
  return row.risk_level || 'LOW';
}

export function riskTone(row: AuditLogListItem) {
  const level = riskValue(row);

  if (level === 'CRITICAL') {
    return 'danger' as const;
  }
  if (level === 'HIGH') {
    return 'warning' as const;
  }
  if (level === 'MEDIUM') {
    return 'primary' as const;
  }
  return 'default' as const;
}

export function riskLabel(row: AuditLogListItem, t: Translate) {
  const level = riskValue(row);
  return t(`audit.common.risk.${level}`);
}

export function resultTone(row: AuditLogListItem) {
  switch (row.result) {
    case 'SUCCESS':
      return 'success' as const;
    case 'DENIED':
      return 'warning' as const;
    case 'ERROR':
      return 'danger' as const;
    default:
      return 'danger' as const;
  }
}

export function resultLabel(row: AuditLogListItem, t: Translate) {
  return t(`audit.common.result.${row.result || 'FAILED'}`);
}

function targetTypeLabel(value: string | null | undefined, t: Translate) {
  if (!value) {
    return '';
  }

  return translateIfPresent(t, `audit.common.targetType.${value}`, value);
}

export function formatAuditTimestamp(value?: string | null, locale?: string) {
  return formatLocaleDateTime(value, locale);
}
