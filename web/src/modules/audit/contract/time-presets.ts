export const AUDIT_TIME_PRESET = {
  LAST_24H: 'last_24h',
  LAST_7D: 'last_7d',
  LAST_30D: 'last_30d',
} as const;

export type AuditTimePreset = (typeof AUDIT_TIME_PRESET)[keyof typeof AUDIT_TIME_PRESET];
