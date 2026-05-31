import type { AuditClientFilterState } from '../shared/presentation';
import type { AuditResult, AuditRiskLevel } from '../types/audit';

export type AuditQuickPresetKey =
  | 'all'
  | 'failed-operations'
  | 'rbac-changes'
  | 'permission-denied'
  | 'sensitive-ops'
  | 'auth-failed'
  | 'high-risk';

export type AuditQuickPresetDefinition = {
  key: AuditQuickPresetKey;
  titleKey: string;
};

const AUDIT_PRESET_DEFINITIONS: readonly AuditQuickPresetDefinition[] = [
  { key: 'all', titleKey: 'audit.logList.presets.all' },
  { key: 'failed-operations', titleKey: 'audit.logList.presets.failedOperations' },
  { key: 'rbac-changes', titleKey: 'audit.logList.presets.rbacChanges' },
  { key: 'permission-denied', titleKey: 'audit.logList.presets.permissionDenied' },
  { key: 'sensitive-ops', titleKey: 'audit.logList.presets.sensitiveOps' },
  { key: 'auth-failed', titleKey: 'audit.logList.presets.authFailed' },
  { key: 'high-risk', titleKey: 'audit.logList.presets.highRisk' },
] as const;

const AUDIT_PRESET_KEY_SET = new Set<AuditQuickPresetKey>(AUDIT_PRESET_DEFINITIONS.map((preset) => preset.key));

const AUDIT_PRESET_ALIASES: Record<string, AuditQuickPresetKey> = {
  'failed-auth': 'auth-failed',
};

const RBAC_ACTION_PREFIXES = ['rbac.', 'role.', 'permission.'] as const;
const SENSITIVE_ACTION_KEYWORDS = [
  'delete',
  'reset',
  'grant',
  'assign',
  'revoke',
  'remove',
  'replace',
  'update_role',
  'update_permission',
] as const;
const AUTH_RESOURCE_TYPES = ['auth', 'session'] as const;
const AUTH_ACTION_KEYWORDS = ['auth', 'login'] as const;
const AUTH_REQUEST_PATH_PREFIXES = ['/api/auth'] as const;
const HIGH_RISK_LEVELS: AuditRiskLevel[] = ['HIGH', 'CRITICAL'];
const PERMISSION_DENIED_RESULTS: AuditResult[] = ['DENIED'];

export function listAuditPresets() {
  return AUDIT_PRESET_DEFINITIONS;
}

export function resolveAuditPresetKey(value: string): AuditQuickPresetKey {
  if (AUDIT_PRESET_KEY_SET.has(value as AuditQuickPresetKey)) {
    return value as AuditQuickPresetKey;
  }

  return AUDIT_PRESET_ALIASES[value] ?? 'all';
}

export function applyAuditPresetFilters(
  preset: AuditQuickPresetKey,
  current: AuditClientFilterState,
  createDefaultFilters: () => AuditClientFilterState,
): AuditClientFilterState {
  const base = createDefaultFilters();

  const next: AuditClientFilterState = {
    ...base,
    keyword: current.keyword,
    requestId: current.requestId,
    sorters: current.sorters,
  };

  switch (preset) {
    case 'failed-operations':
      next.success = 'false';
      return next;
    case 'rbac-changes':
      next.actionPrefixes = [...RBAC_ACTION_PREFIXES];
      return next;
    case 'permission-denied':
      next.results = [...PERMISSION_DENIED_RESULTS];
      return next;
    case 'sensitive-ops':
      next.actionKeywords = [...SENSITIVE_ACTION_KEYWORDS];
      return next;
    case 'auth-failed':
      next.success = 'false';
      next.resourceTypes = [...AUTH_RESOURCE_TYPES];
      next.actionKeywords = [...AUTH_ACTION_KEYWORDS];
      next.requestPathPrefixes = [...AUTH_REQUEST_PATH_PREFIXES];
      return next;
    case 'high-risk':
      next.riskLevels = [...HIGH_RISK_LEVELS];
      return next;
    default:
      return next;
  }
}
