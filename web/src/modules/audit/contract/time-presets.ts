export const AUDIT_TIME_PRESET = {
  LAST_24H: 'last_24h',
  LAST_7D: 'last_7d',
  LAST_30D: 'last_30d',
} as const;

export type AuditTimePreset = (typeof AUDIT_TIME_PRESET)[keyof typeof AUDIT_TIME_PRESET];

const AUDIT_TIME_PRESET_SET = new Set<AuditTimePreset>(Object.values(AUDIT_TIME_PRESET));

export function resolveAuditTimePreset(value: string): AuditTimePreset {
  return AUDIT_TIME_PRESET_SET.has(value as AuditTimePreset) ? (value as AuditTimePreset) : AUDIT_TIME_PRESET.LAST_24H;
}
