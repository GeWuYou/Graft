export const AUDIT_SCOPE = {
  ALL_LOGS: 'all_logs',
  FAILED_OPERATIONS: 'failed_operations',
  HIGH_RISK_EVENTS: 'high_risk_events',
  SENSITIVE_OPERATIONS: 'sensitive_operations',
  CRITICAL_SECURITY: 'critical_security',
  HIGH_RISK_OPERATIONS: 'high_risk_operations',
  AUTH_FAILURES: 'auth_failures',
  PERMISSION_DENIALS: 'permission_denials',
  RBAC_CHANGES: 'rbac_changes',
} as const;

export type AuditScope = (typeof AUDIT_SCOPE)[keyof typeof AUDIT_SCOPE];
