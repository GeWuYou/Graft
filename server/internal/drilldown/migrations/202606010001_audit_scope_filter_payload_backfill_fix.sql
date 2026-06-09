-- Copyright (c) 2025-2026 GeWuYou
-- SPDX-License-Identifier: Apache-2.0

UPDATE "system_drilldown_scope"
SET
  "filter_payload" = NULL,
  "updated_at" = NOW()
WHERE "module" = 'audit'
  AND "scope" IN (
    'failed_operations',
    'high_risk_operations',
    'sensitive_operations',
    'auth_failures',
    'permission_denials',
    'rbac_changes',
    'critical_security'
  );
