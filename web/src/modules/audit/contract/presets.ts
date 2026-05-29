import type { AuditClientFilterState } from '../shared/presentation';

export type AuditPresetKey =
  | 'all'
  | 'today-anomalies'
  | 'permission-denied'
  | 'sensitive-ops'
  | 'auth-failed'
  | 'high-risk';

export type AuditPresetDefinition = {
  key: AuditPresetKey;
  titleKey: string;
  defaults: Partial<AuditClientFilterState>;
};

const AUDIT_PRESET_DEFINITIONS: readonly AuditPresetDefinition[] = [
  {
    key: 'all',
    titleKey: 'audit.logList.presets.all',
    defaults: {},
  },
  {
    key: 'today-anomalies',
    titleKey: 'audit.logList.presets.todayAnomalies',
    defaults: {
      source: 'SECURITY_EVENT',
      result: 'ERROR',
      riskLevel: 'HIGH',
    },
  },
  {
    key: 'permission-denied',
    titleKey: 'audit.logList.presets.permissionDenied',
    defaults: {
      source: 'SECURITY_EVENT',
      result: 'DENIED',
      riskLevel: 'CRITICAL',
    },
  },
  {
    key: 'sensitive-ops',
    titleKey: 'audit.logList.presets.sensitiveOps',
    defaults: {
      riskLevel: 'HIGH',
    },
  },
  {
    key: 'auth-failed',
    titleKey: 'audit.logList.presets.authFailed',
    defaults: {
      source: 'REQUEST',
      result: 'FAILED',
      resourceType: 'auth',
      riskLevel: 'HIGH',
    },
  },
  {
    key: 'high-risk',
    titleKey: 'audit.logList.presets.highRisk',
    defaults: {
      source: 'SECURITY_EVENT',
      riskLevel: 'CRITICAL',
    },
  },
] as const;

const AUDIT_PRESET_KEY_SET = new Set<AuditPresetKey>(AUDIT_PRESET_DEFINITIONS.map((preset) => preset.key));

const AUDIT_PRESET_ALIASES: Record<string, AuditPresetKey> = {
  'failed-auth': 'auth-failed',
  'rbac-changes': 'permission-denied',
};

export function listAuditPresets() {
  return AUDIT_PRESET_DEFINITIONS;
}

export function getAuditPresetDefaults(key: AuditPresetKey): Partial<AuditClientFilterState> {
  return AUDIT_PRESET_DEFINITIONS.find((preset) => preset.key === key)?.defaults ?? {};
}

export function resolveAuditPresetKey(value: string): AuditPresetKey {
  if (AUDIT_PRESET_KEY_SET.has(value as AuditPresetKey)) {
    return value as AuditPresetKey;
  }

  return AUDIT_PRESET_ALIASES[value] ?? 'all';
}
