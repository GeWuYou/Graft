// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { AuditBusinessCategory } from '../types/audit';

export type AuditQuickPresetKey =
  | 'all'
  | 'security-events'
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
  { key: 'security-events', titleKey: 'audit.logList.presets.securityEvents' },
  { key: 'failed-operations', titleKey: 'audit.logList.presets.failedOperations' },
  { key: 'rbac-changes', titleKey: 'audit.logList.presets.rbacChanges' },
  { key: 'permission-denied', titleKey: 'audit.logList.presets.permissionDenied' },
  { key: 'sensitive-ops', titleKey: 'audit.logList.presets.sensitiveOps' },
  { key: 'auth-failed', titleKey: 'audit.logList.presets.authFailed' },
  { key: 'high-risk', titleKey: 'audit.logList.presets.highRisk' },
] as const;

export const AUDIT_DRILLDOWN_SCOPE = {
  FAILED_OPERATIONS: 'failed_operations',
  HIGH_RISK_OPERATIONS: 'high_risk_operations',
  SENSITIVE_OPERATIONS: 'sensitive_operations',
  AUTH_FAILURES: 'auth_failures',
  PERMISSION_DENIALS: 'permission_denials',
  RBAC_CHANGES: 'rbac_changes',
  CRITICAL_SECURITY: 'critical_security',
} as const;

export const AUDIT_BUSINESS_CATEGORY = AUDIT_DRILLDOWN_SCOPE satisfies Record<string, AuditBusinessCategory>;

export function listAuditPresets() {
  return AUDIT_PRESET_DEFINITIONS;
}
