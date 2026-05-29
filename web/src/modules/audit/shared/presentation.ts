import type { AuditLogListItem } from '../types/audit';
import type { AuditResult as AuditResultEnum, AuditRiskLevel as AuditRiskLevelEnum } from '../types/audit';

type Translate = (key: string, params?: Record<string, unknown>) => string;

export type AuditRiskValue = 'all' | AuditRiskLevelEnum;
export type AuditResultValue = 'all' | AuditResultEnum;

export type AuditClientFilterState = {
  keyword: string;
  actor: string;
  action: string;
  actionPrefix: string;
  source: string;
  createdRange: string[];
  resourceType: string;
  resourceName: string;
  resourceId: string;
  result: AuditResultValue;
  riskLevel: 'all' | AuditRiskValue;
  session: string;
  requestId: string;
  traceId: string;
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

export function traceIdForRecord(row: AuditLogListItem) {
  return row.trace_id || metadataLookup(row, 'trace_id') || row.request_id || '-';
}

export function requestIdForRecord(row: AuditLogListItem) {
  return row.request_id || metadataLookup(row, 'request_id') || traceIdForRecord(row);
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

export function metadataLookup(row: AuditLogListItem, key: string) {
  const metadata = row.metadata;
  if (!metadata || typeof metadata !== 'object' || !(key in metadata)) {
    return '';
  }

  const value = metadata[key];
  return typeof value === 'string' || typeof value === 'number' ? String(value) : JSON.stringify(value);
}

export function metadataDetail(metadata: AuditLogListItem['metadata']) {
  if (!metadata || typeof metadata !== 'object' || Object.keys(metadata).length === 0) {
    return '-';
  }

  return JSON.stringify(metadata, null, 2);
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

export function formatAuditTimestamp(value?: string | null) {
  if (!value) {
    return '-';
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date);
}

function includesText(source: string, search: string) {
  return source.toLowerCase().includes(search.trim().toLowerCase());
}

export function matchesAuditRow(row: AuditLogListItem, filters: AuditClientFilterState, t: Translate) {
  const keyword = filters.keyword.trim().toLowerCase();
  const actor = filters.actor.trim().toLowerCase();
  const action = filters.action.trim().toLowerCase();
  const actionPrefix = filters.actionPrefix.trim().toLowerCase();
  const source = filters.source.trim().toUpperCase();
  const resourceType = filters.resourceType.trim().toLowerCase();
  const resourceName = filters.resourceName.trim().toLowerCase();
  const resourceId = filters.resourceId.trim().toLowerCase();
  const session = filters.session.trim().toLowerCase();
  const requestId = filters.requestId.trim().toLowerCase();
  const traceId = filters.traceId.trim().toLowerCase();

  if (keyword) {
    const keywordSource = [
      row.action,
      row.request_id,
      row.trace_id,
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

  if (action && !includesText(row.action, action)) {
    return false;
  }

  if (actionPrefix && !row.action.toLowerCase().startsWith(actionPrefix)) {
    return false;
  }

  if (source && (row.source || sourceForRecord(row)) !== source) {
    return false;
  }

  if (resourceType && !includesText(row.resource_type || row.target_type || '', resourceType)) {
    return false;
  }

  if (resourceName && !includesText(`${resourceDetailLabel(row, t)} ${row.message}`, resourceName)) {
    return false;
  }

  if (resourceId && !includesText(row.resource_id || '', resourceId)) {
    return false;
  }

  if (filters.result !== 'all' && row.result !== filters.result) {
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

  if (traceId && !includesText(traceIdForRecord(row), traceId)) {
    return false;
  }

  return true;
}
