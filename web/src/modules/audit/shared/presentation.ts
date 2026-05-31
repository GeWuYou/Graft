import { formatLocaleDateTime } from '@/shared/observability';

import type { AuditLogListItem } from '../types/audit';
import type { AuditResult as AuditResultEnum, AuditRiskLevel as AuditRiskLevelEnum } from '../types/audit';
import type { AuditSorter } from '../types/audit';

type Translate = (key: string, params?: Record<string, unknown>) => string;

export type AuditRiskValue = 'all' | AuditRiskLevelEnum;
export type AuditResultValue = 'all' | AuditResultEnum;

export type AuditClientFilterState = {
  keyword: string;
  actor: string;
  actorUserId: string;
  success: 'all' | 'true' | 'false';
  action: string;
  actionPrefix: string;
  actionPrefixes: string[];
  actionKeywords: string[];
  requestPathPrefixes: string[];
  source: string;
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

export function resourceLabel(row: AuditLogListItem, t: Translate) {
  return (
    row.target_label ||
    row.resource_name ||
    resourceSecondaryLabel(row) ||
    row.request_path ||
    metadataLookup(row, 'request_path') ||
    t('audit.common.unknownResource')
  );
}

function resourceSecondaryLabel(row: AuditLogListItem) {
  const secondary = [targetTypeLabel(row.target_type), row.resource_id].filter(Boolean);
  return secondary.join(' / ') || '-';
}

export function resourceDetailLabel(row: AuditLogListItem, t: Translate) {
  const label = row.target_label || row.resource_name || resourceSecondaryLabel(row) || row.request_path;
  return (
    [label, targetTypeLabel(row.target_type), row.resource_id].filter(Boolean).join(' / ') ||
    t('audit.common.unknownResource')
  );
}

export function requestIdForRecord(row: AuditLogListItem) {
  return row.request_id || metadataLookup(row, 'request_id') || '-';
}

export function sessionIdForRecord(row: AuditLogListItem) {
  return row.session_id || metadataLookup(row, 'session_id') || '-';
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

function sourceForRecord(row: AuditLogListItem): AuditSourceValue {
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

export function sourceLabel(row: AuditLogListItem, t: Translate) {
  return t(`audit.common.source.${row.source || sourceForRecord(row)}`);
}

function translateIfPresent(t: Translate, key: string, fallback: string) {
  const translated = t(key);
  return translated === key ? fallback : translated;
}

export function actionCategoryLabel(row: AuditLogListItem, t: Translate) {
  return sourceLabel(row, t);
}

export function actionTitle(row: AuditLogListItem, t: Translate) {
  const actionKey = row.action?.trim();
  if (!actionKey) {
    return t('audit.common.unknownResource');
  }

  return translateIfPresent(t, `audit.actionLabel.${actionKey}`, actionCategoryLabel(row, t));
}

export function metadataLookup(row: AuditLogListItem, key: string) {
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

function targetTypeLabel(value?: string | null) {
  switch (value) {
    case 'USER':
      return '用户';
    case 'ROLE':
      return '角色';
    case 'PERMISSION':
      return '权限';
    case 'AUDIT':
      return '审计';
    case 'SERVER_STATUS':
      return '服务器状态';
    case 'AUTH':
      return '认证';
    default:
      return value || '';
  }
}

export function formatAuditTimestamp(value?: string | null, locale?: string) {
  return formatLocaleDateTime(value, locale);
}

function includesText(source: string, search: string) {
  return source.toLowerCase().includes(search.trim().toLowerCase());
}

export function matchesAuditRow(row: AuditLogListItem, filters: AuditClientFilterState, t: Translate) {
  const keyword = filters.keyword.trim().toLowerCase();
  const actor = filters.actor.trim().toLowerCase();
  const actorUserId = filters.actorUserId.trim();
  const success = filters.success;
  const action = filters.action.trim().toLowerCase();
  const actionPrefix = filters.actionPrefix.trim().toLowerCase();
  const actionPrefixes = filters.actionPrefixes.map((item) => item.trim().toLowerCase()).filter(Boolean);
  const actionKeywords = filters.actionKeywords.map((item) => item.trim().toLowerCase()).filter(Boolean);
  const requestPathPrefixes = filters.requestPathPrefixes.map((item) => item.trim().toLowerCase()).filter(Boolean);
  const source = filters.source.trim().toUpperCase();
  const resourceType = filters.resourceType.trim().toLowerCase();
  const resourceTypes = filters.resourceTypes.map((item) => item.trim().toLowerCase()).filter(Boolean);
  const resourceName = filters.resourceName.trim().toLowerCase();
  const resourceId = filters.resourceId.trim().toLowerCase();
  const results = filters.results;
  const riskLevels = filters.riskLevels;
  const session = filters.session.trim().toLowerCase();
  const requestId = filters.requestId.trim().toLowerCase();
  if (keyword) {
    const keywordSource = [
      row.action,
      actionTitle(row, t),
      row.request_id,
      row.message,
      actorLabel(row, t),
      resourceLabel(row, t),
      row.resource_type,
      row.resource_id,
    ]
      .filter(Boolean)
      .join(' ')
      .toLowerCase();

    if (!keywordSource.includes(keyword)) {
      return false;
    }
  }

  if (actor && !includesText(`${actorLabel(row, t)} ${actorSecondaryLabel(row)}`, actor)) {
    return false;
  }

  if (actorUserId && String(row.actor_user_id ?? '') !== actorUserId) {
    return false;
  }

  if (success !== 'all' && row.success !== (success === 'true')) {
    return false;
  }

  if (action && !includesText(row.action, action)) {
    return false;
  }

  if (actionPrefix && !row.action.toLowerCase().startsWith(actionPrefix)) {
    return false;
  }

  if (actionPrefixes.length && !actionPrefixes.some((prefix) => row.action.toLowerCase().startsWith(prefix))) {
    return false;
  }

  if (actionKeywords.length && !actionKeywords.some((keywordItem) => row.action.toLowerCase().includes(keywordItem))) {
    return false;
  }

  if (source && (row.source || sourceForRecord(row)) !== source) {
    return false;
  }

  if (resourceType && !includesText(row.resource_type || row.target_type || '', resourceType)) {
    return false;
  }

  if (
    resourceTypes.length &&
    !resourceTypes.some((resourceTypeItem) =>
      includesText(`${row.resource_type || ''} ${row.target_type || ''}`, resourceTypeItem),
    )
  ) {
    return false;
  }

  if (resourceName && !includesText(`${resourceDetailLabel(row, t)} ${row.message}`, resourceName)) {
    return false;
  }

  if (resourceId && !includesText(row.resource_id || '', resourceId)) {
    return false;
  }

  if (results.length && !results.includes(row.result || 'FAILED')) {
    return false;
  }

  if (filters.result !== 'all' && row.result !== filters.result) {
    return false;
  }

  if (riskLevels.length && !riskLevels.includes(row.risk_level || 'LOW')) {
    return false;
  }

  if (filters.riskLevel !== 'all' && (row.risk_level || 'LOW') !== filters.riskLevel) {
    return false;
  }

  if (session && !includesText(sessionIdForRecord(row), session)) {
    return false;
  }

  if (requestId && !includesText(requestIdForRecord(row), requestId)) {
    return false;
  }

  if (requestPathPrefixes.length) {
    const requestPath = (row.request_path || metadataLookup(row, 'request_path')).toLowerCase();
    if (!requestPathPrefixes.some((prefix) => requestPath.startsWith(prefix))) {
      return false;
    }
  }

  return true;
}
