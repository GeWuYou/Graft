import type { AuditLogListItem } from '../types/audit';

type Translate = (key: string, params?: Record<string, unknown>) => string;

export type AuditRiskValue = 'high' | 'sensitive' | 'normal';
export type AuditResultValue = 'all' | 'success' | 'failed';

export type AuditClientFilterState = {
  keyword: string;
  actor: string;
  action: string;
  createdRange: string[];
  resource: string;
  result: AuditResultValue;
  riskLevel: 'all' | AuditRiskValue;
  session: string;
  traceId: string;
};

export function actorLabel(row: AuditLogListItem, t: Translate) {
  return row.actor_display_name || row.actor_username || t('audit.common.unknownActor');
}

export function actorSecondaryLabel(row: AuditLogListItem) {
  return row.actor_username || row.actor_user_id?.toString() || '-';
}

export function resourceLabel(row: AuditLogListItem, t: Translate) {
  return (
    row.resource_name ||
    resourceSecondaryLabel(row) ||
    metadataLookup(row, 'request_path') ||
    t('audit.common.unknownResource')
  );
}

export function resourceSecondaryLabel(row: AuditLogListItem) {
  const secondary = [row.resource_type, row.resource_id].filter(Boolean);
  return secondary.join(' / ') || '-';
}

export function resourceDetailLabel(row: AuditLogListItem, t: Translate) {
  const label = row.resource_name || resourceSecondaryLabel(row) || metadataLookup(row, 'request_path');
  return [label, row.resource_type, row.resource_id].filter(Boolean).join(' / ') || t('audit.common.unknownResource');
}

export function traceIdForRecord(row: AuditLogListItem) {
  return metadataLookup(row, 'trace_id') || row.request_id || '-';
}

export function sessionIdForRecord(row: AuditLogListItem) {
  return metadataLookup(row, 'session_id') || '-';
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
  const action = row.action.toLowerCase();
  return ['delete', 'role', 'permission', 'reset', 'grant', 'assign'].some((keyword) => action.includes(keyword));
}

function riskValue(row: AuditLogListItem): AuditRiskValue {
  if (!row.success) {
    return 'high';
  }
  if (isSensitiveAction(row)) {
    return 'sensitive';
  }
  return 'normal';
}

export function riskTone(row: AuditLogListItem) {
  const level = riskValue(row);

  if (level === 'high') {
    return 'danger' as const;
  }
  if (level === 'sensitive') {
    return 'warning' as const;
  }
  return 'success' as const;
}

export function riskLabel(row: AuditLogListItem, t: Translate) {
  const level = riskValue(row);
  return t(`audit.common.risk.${level}`);
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
  const resource = filters.resource.trim().toLowerCase();
  const session = filters.session.trim().toLowerCase();
  const traceId = filters.traceId.trim().toLowerCase();

  if (keyword) {
    const keywordSource = [
      row.action,
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

  if (action && !includesText(row.action, action)) {
    return false;
  }

  if (resource && !includesText(`${resourceDetailLabel(row, t)} ${row.message}`, resource)) {
    return false;
  }

  if (filters.result === 'success' && !row.success) {
    return false;
  }
  if (filters.result === 'failed' && row.success) {
    return false;
  }

  if (filters.riskLevel !== 'all' && riskValue(row) !== filters.riskLevel) {
    return false;
  }

  if (session && !includesText(sessionIdForRecord(row), session)) {
    return false;
  }

  if (traceId && !includesText(traceIdForRecord(row), traceId)) {
    return false;
  }

  return true;
}
