import { AUDIT_SCOPE, type AuditScope } from './scopes';

export type AuditQuickPresetKey =
  | 'all'
  | 'rbac-changes'
  | 'permission-denied'
  | 'sensitive-ops'
  | 'auth-failed'
  | 'high-risk';

export type AuditQuickPresetDefinition = {
  key: AuditQuickPresetKey;
  titleKey: string;
  scope: AuditScope | '';
};

const AUDIT_PRESET_DEFINITIONS: readonly AuditQuickPresetDefinition[] = [
  { key: 'all', titleKey: 'audit.logList.presets.all', scope: '' },
  { key: 'rbac-changes', titleKey: 'audit.logList.presets.rbacChanges', scope: AUDIT_SCOPE.RBAC_CHANGES },
  {
    key: 'permission-denied',
    titleKey: 'audit.logList.presets.permissionDenied',
    scope: AUDIT_SCOPE.PERMISSION_DENIALS,
  },
  { key: 'sensitive-ops', titleKey: 'audit.logList.presets.sensitiveOps', scope: AUDIT_SCOPE.SENSITIVE_OPERATIONS },
  { key: 'auth-failed', titleKey: 'audit.logList.presets.authFailed', scope: AUDIT_SCOPE.AUTH_FAILURES },
  { key: 'high-risk', titleKey: 'audit.logList.presets.highRisk', scope: AUDIT_SCOPE.HIGH_RISK_EVENTS },
] as const;

const AUDIT_PRESET_KEY_SET = new Set<AuditQuickPresetKey>(AUDIT_PRESET_DEFINITIONS.map((preset) => preset.key));

const AUDIT_PRESET_ALIASES: Record<string, AuditQuickPresetKey> = {
  'failed-auth': 'auth-failed',
};

export function listAuditPresets() {
  return AUDIT_PRESET_DEFINITIONS;
}

export function getAuditPresetScope(key: AuditQuickPresetKey): AuditScope | '' {
  return AUDIT_PRESET_DEFINITIONS.find((preset) => preset.key === key)?.scope ?? '';
}

export function resolveAuditPresetKeyFromScope(scope: string): AuditQuickPresetKey {
  const matched = AUDIT_PRESET_DEFINITIONS.find((preset) => preset.scope === scope);
  return matched?.key ?? 'all';
}

export function resolveAuditPresetKey(value: string): AuditQuickPresetKey {
  if (AUDIT_PRESET_KEY_SET.has(value as AuditQuickPresetKey)) {
    return value as AuditQuickPresetKey;
  }

  return AUDIT_PRESET_ALIASES[value] ?? 'all';
}
